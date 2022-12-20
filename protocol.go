package core

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/hood-chat/core/entity"
	"github.com/hood-chat/core/event"
	"github.com/hood-chat/core/pb"
	"github.com/hood-chat/core/utils"
	lpevent "github.com/libp2p/go-libp2p/core/event"
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

type PMService interface {
	Send(entity.Envelop)
	Handler(str network.Stream)
	Stop()
}

// NewNATManager creates a NAT manager.
func NewPMService(h host.Host, ebus lpevent.Bus) PMService {
	return newPMService(h, ebus)
}

type pmService struct {
	host      host.Host
	connector Connector
	backoff   bf.BackoffFactory
	nvlpCh    chan entity.Envelop

	emitters struct {
		evtMessageReceived      lpevent.Emitter
		evtMessageStatusChanged lpevent.Emitter
	}
	mux    sync.Mutex
	outbox map[peer.ID][]*pb.Message
}

func newPMService(h host.Host, ebus lpevent.Bus) PMService {
	pms := &pmService{}
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
	h.SetStreamHandler(ID, pms.Handler)
	log.Debug("service PMS created")
	pms.nvlpCh = make(chan entity.Envelop)
	pms.outbox = make(map[peer.ID][]*pb.Message)
	pms.backoff = bf.NewPolynomialBackoff(time.Second, time.Second*33, bf.NoJitter, time.Second, []float64{0.5, 2, 3}, rand.NewSource(0))
	pms.connector = NewConnector(h)
	pms.host.Network().Notify((*pmsNotifiee)(pms))
	go pms.background(context.Background(), pms.nvlpCh)
	return pms
}

func (c *pmService) sendWithBackoff(ctx context.Context, p peer.ID, pbmsg *pb.Message) {
	bfk := c.backoff()
	ticker := time.NewTicker(1 * time.Second)
	maxtry := 5
	for {
		select {
		case <-ticker.C:
			ticker.Stop()
			err := c.send(p, pbmsg)
			if err != nil && maxtry > 0 {
				ticker.Reset(bfk.Delay())
				continue
			}
			c.failed(pbmsg.Id, p)
			return

		case <-ctx.Done():
			return
		}
	}

}

func (c *pmService) send(p peer.ID, pbmsg *pb.Message) error {
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

func (c *pmService) Send(nvlop entity.Envelop) {

	c.nvlpCh <- nvlop

}

func (c *pmService) background(ctx context.Context, nvlpCh <-chan entity.Envelop) {
	ticker := time.NewTicker(1 * time.Minute)
	for {
		select {
		case t := <-ticker.C:
			c.mux.Lock()
			tmp := make(map[peer.ID][]*pb.Message)
			for k, v := range c.outbox {
				for _, m := range v {
					if m.CreatedAt+(60*5) >= t.UTC().Unix() {
						c.failed(m.Id, k)
					} else {
						tmp[k] = append(tmp[k], m)
					}
				}
			}
			c.outbox = tmp
			c.mux.Unlock()
		case nvlp := <-nvlpCh:
			h := c.host
			pi, err := nvlp.To.AdderInfo()
			if err != nil {
				continue
			}

			if pi.ID == c.host.ID() || pi.ID == "" {
				continue
			}
			cns := h.Network().Connectedness(pi.ID)
			switch cns {
			case network.Connected:

				err := c.send(pi.ID, nvlp.Proto())
				if err != nil {
					c.outboxed(pi, nvlp.Proto())
				}

			default:
				c.outboxed(pi, nvlp.Proto())
			}
		case <-ctx.Done():
			log.Errorf("context error broke sender")
		}

	}
}

func (c *pmService) outboxed(to *peer.AddrInfo, msg *pb.Message) {
	c.mux.Lock()
	c.outbox[to.ID] = append(c.outbox[to.ID], msg)
	c.mux.Unlock()
	go c.connector.Need(ID, *to)
}

func (c *pmService) Handler(str network.Stream) {
	log.Debugf("Handler called")
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

func (c *pmService) Stop() {
	c.host.RemoveStreamHandler(ID)
	c.emitters.evtMessageReceived.Close()
	c.emitters.evtMessageStatusChanged.Close()
}

func (c *pmService) done(msgID string, pid peer.ID) {
	c.emitMessageChange(entity.Sent, msgID)
	c.connector.Done(ID, pid)
}

func (c *pmService) failed(msgID string, pid peer.ID) {
	c.emitMessageChange(entity.Failed, msgID)
	c.connector.Done(ID, pid)
}

func (c *pmService) emitMessageChange(status entity.Status, msgID string) {
	evgrp := event.NewMessagingEventGroup()
	ev, err := evgrp.Make("ChangeMessageStatus", status, entity.ID(msgID))
	if err != nil {
		log.Errorf("can not create event. reason: %s", err)
		panic("bus has problem")
	}
	c.emitters.evtMessageStatusChanged.Emit(*ev)
}

type pmsNotifiee pmService

func (pm *pmsNotifiee) pmService() *pmService {
	return (*pmService)(pm)
}

func (pm *pmsNotifiee) Listen(network.Network, ma.Multiaddr)       {}
func (pm *pmsNotifiee) ListenClose(network.Network, ma.Multiaddr)  {}
func (pm *pmsNotifiee) Disconnected(network.Network, network.Conn) {}
func (pm *pmsNotifiee) Connected(n network.Network, c network.Conn) {
	log.Debug("connected")
	pm.mux.Lock()
	msgs, ok := pm.outbox[c.RemotePeer()]
	log.Debugf("have %s messages", len(msgs))
	delete(pm.outbox, c.RemotePeer())
	pm.mux.Unlock()
	if ok {
		for _, val := range msgs {
			go pm.pmService().sendWithBackoff(context.Background(), c.RemotePeer(), val)
		}
	}
}
