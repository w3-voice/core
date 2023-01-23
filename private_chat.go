package core

import (
	"context"
	"math/rand"
	"time"

	"github.com/hood-chat/core/entity"
	"github.com/hood-chat/core/event"
	"github.com/hood-chat/core/pb"
	"github.com/hood-chat/core/utils"
	"github.com/libp2p/go-libp2p/core/host"

	// "github.com/libp2p/go-libp2p/p2p/discovery/backoff"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	bf "github.com/libp2p/go-libp2p/p2p/discovery/backoff"
	"github.com/libp2p/go-libp2p/p2p/host/eventbus"
	"github.com/libp2p/go-msgio/protoio"
	ma "github.com/multiformats/go-multiaddr"
)

const (
	MessageTimeout = time.Second * 60

	ID = "/chat/pm/1.0.0"

	ServiceName = "chat.pm"

	MaxMsgSize = 10 * 1024 // 4K

	StreamTimeout  = time.Minute
	ConnectTimeout = 30 * time.Second
)



// NewNATManager creates a NAT manager.
func NewPMService(h host.Host, ebus Bus, connector Connector) MessengerService {
	return newPMService(h, ebus, connector)
}

type PMService struct {
	host      host.Host
	connector Connector
	backoff   bf.BackoffFactory
	nvlpCh    chan entity.Envelop
	outbox    *outbox
	emitters  struct {
		evtMessageReceived      Emitter
		evtMessageStatusChanged Emitter
	}
}

func newPMService(h host.Host, ebus Bus, connector Connector) MessengerService {
	pms := &PMService{}
	var err error
	pms.emitters.evtMessageStatusChanged, err = ebus.Emitter(new(event.EvtObject), eventbus.Stateful)
	if err != nil {
		log.Errorf("error reading message: %s", err.Error())
		panic("failed to create message service")
	}
	pms.emitters.evtMessageReceived, err = ebus.Emitter(new(event.EvtMessageReceived), eventbus.Stateful)
	if err != nil {
		log.Errorf("error reading message: %s", err.Error())
		panic("failed to create message service")
	}
	pms.host = h
	h.SetStreamHandler(ID, pms.handler)
	log.Debug("service PMS created")
	pms.nvlpCh = make(chan entity.Envelop)
	pms.outbox = newOutBox()
	pms.backoff = bf.NewPolynomialBackoff(time.Second*5, time.Second*10, bf.NoJitter, time.Second, []float64{5, 7, 10}, rand.NewSource(0))
	pms.connector = connector
	// set static relay as needed connection
	relayInfo, err  := peer.AddrInfoFromString(StaticRelays[0])
	if err != nil {
		panic("failed to create message service: "+ err.Error())
	}
	relayInfo.Addrs = []ma.Multiaddr{}
	pms.connector.Need(ServiceName, *relayInfo)
	pms.host.Network().Notify((*pmsNotifiee)(pms))
	go pms.background(context.Background(), pms.nvlpCh)
	return pms
}

func (c *PMService) send(p peer.ID, pbmsg *pb.Message) error {
	nctx := network.WithUseTransient(context.Background(), "just a chat")
	s, err := c.host.NewStream(nctx, p, ID)
	if err != nil {
		log.Errorf("new stream failed: %s", err)
		return err
	}
	if err := s.Scope().ReserveMemory(MaxMsgSize, network.ReservationPriorityAlways); err != nil {
		log.Debugf("error reserving memory for message stream: %s", err)
		s.Reset()
		return err
		// return 0, err
	}
	defer s.Scope().ReleaseMemory(MaxMsgSize)
	wr := protoio.NewDelimitedWriter(s)
	defer func() {
		wr.Close()
	}()
	if err != nil {
		log.Errorf("error connecting %s", err)
		return err
	}
	log.Debugf("text sent with message text: %s", pbmsg.GetText())
	wr.WriteMsg(pbmsg)
	if err != nil {
		log.Errorf("write err %s", err)
		return err
	}
	c.done(pbmsg.Id, p)
	return nil
}

func (c *PMService) Send(nvlop entity.Envelop) {

	c.nvlpCh <- nvlop

}

func (c *PMService) background(ctx context.Context, nvlpCh <-chan entity.Envelop) {
	for {
		select {
		case m := <-c.outbox.failed:
			c.failed(m.Message.Proto().Id, peer.ID(m.To.ID))
		case nvlp := <-nvlpCh:
			h := c.host

			pi, err := nvlp.To.AdderInfo()
			if err != nil {
				continue
			}

			// hack to use relay v1
			h.Peerstore().AddAddrs(pi.ID,
				[]ma.Multiaddr{
					ma.StringCast("/p2p/" + "12D3KooWBFpA7pCMBySBqtduBVkakVQ3bmmaeagB83WHoruBN9s9" + "/p2p-circuit/p2p/" + nvlp.To.ID.String()),
					ma.StringCast("/p2p/" + "12D3KooWBFpA7pCMBySBqtduBVkakVQ3bmmaeagB83WHoruBN9s9" + "/p2p-circuit/p2p/" + nvlp.To.ID.String()),
				},
				time.Minute*5)
			if pi.ID == c.host.ID() || pi.ID == "" {
				continue
			}
			c.connector.Need(nvlp.Message.Proto().Id, *pi)
			cns := h.Network().Connectedness(pi.ID)
			switch cns {
			case network.Connected:
				err := c.send(pi.ID, nvlp.Message.Proto())
				if err != nil {
					c.outbox.put(pi.ID, &nvlp)
				}

			default:
				c.outbox.put(pi.ID, &nvlp)
			}
		case <-ctx.Done():
			log.Errorf("context error broke sender")
		}

	}
}

func (c *PMService) handler(str network.Stream) {
	log.Debugf("Handler called")
	defer str.Close()
	if err := str.Scope().SetService(ServiceName); err != nil {
		log.Debugf("error attaching stream to ping service: %s", err)
		str.Reset()
		return
	}

	if err := str.Scope().ReserveMemory(MaxMsgSize, network.ReservationPriorityAlways); err != nil {
		log.Debugf("error reserving memory for Private Message stream: %s", err)
		str.Reset()
		return
	}
	defer str.Scope().ReleaseMemory(MaxMsgSize)

	rd := utils.NewDelimitedReader(str, MaxMsgSize)
	defer rd.Close()

	str.SetDeadline(time.Now().Add(StreamTimeout))

	var msg pb.Message

	err := rd.ReadMsg(&msg)
	if err != nil {
		log.Errorf("error reading message: %s", err.Error())
		str.Reset()
		return
	}
	log.Debugf("message received ... %s", msg.GetText())
	err = c.emitters.evtMessageReceived.Emit(event.EvtMessageReceived{Msg: &msg})
	if err != nil {
		log.Errorf("failed to emit event: %s", err.Error())
		str.Reset()
		return
	}

}

func (c *PMService) Stop() {
	c.host.RemoveStreamHandler(ID)
	c.emitters.evtMessageReceived.Close()
	c.emitters.evtMessageStatusChanged.Close()
}

func (c *PMService) done(msgID string, pid peer.ID) {
	event.EmitMessageChange(c.emitters.evtMessageStatusChanged,entity.Sent, msgID)
	c.connector.Done(msgID, pid)
}

func (c *PMService) failed(msgID string, pid peer.ID) {
	event.EmitMessageChange(c.emitters.evtMessageStatusChanged, entity.Failed, msgID)
	c.connector.Done(msgID, pid)
}

func (c *PMService) onConnected(pid peer.ID) {
	msgs := c.outbox.pop(pid)
	go func(msgs []*entity.Envelop) {
		for _, val := range msgs {
			err := c.send(pid, val.Message.Proto())
			if err != nil {
				c.outbox.put(pid, val)
			}
		}
	}(msgs)

}

type pmsNotifiee PMService

func (pm *pmsNotifiee) pmService() *PMService {
	return (*PMService)(pm)
}

func (pm *pmsNotifiee) Listen(network.Network, ma.Multiaddr)       {}
func (pm *pmsNotifiee) ListenClose(network.Network, ma.Multiaddr)  {}
func (pm *pmsNotifiee) Disconnected(network.Network, network.Conn) {}
func (pm *pmsNotifiee) Connected(n network.Network, c network.Conn) {
	pm.pmService().onConnected(c.RemotePeer())
}
