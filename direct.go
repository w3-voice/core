package core

import (
	"context"
	"math/rand"
	"time"

	"github.com/hood-chat/core/entity"
	"github.com/hood-chat/core/event"
	"github.com/hood-chat/core/pb"
	"github.com/hood-chat/core/protocol/message"
	"github.com/hood-chat/core/protocol/invite"
	"github.com/libp2p/go-libp2p/core/host"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	bf "github.com/libp2p/go-libp2p/p2p/discovery/backoff"

	ma "github.com/multiformats/go-multiaddr"
)


var _ DirectService = (*DirectMessaging)(nil)

type DirectMessaging struct {
	host       host.Host
	connector  Connector
	backoff    bf.BackoffFactory
	input      chan *entity.Envelop
	outbox     *outbox
	bus        Bus
}

// NewDirectMessaging creates a Direct messaging service
func NewDirectMessaging(h host.Host, ebus Bus, connector Connector, input chan *entity.Envelop) DirectService {
	dms := &DirectMessaging{}
	var err error
	dms.bus = ebus
	dms.host = h
	// register message protocol
	direct.SetMessageHandler(h, dms.messageHandler)
	// register invite protocol
	invite.SetInviteHandler(h, dms.inviteHandler)
	log.Debug("service PMS created")
	dms.input = input
	dms.outbox = newOutBox()
	dms.backoff = bf.NewPolynomialBackoff(time.Second*5, time.Second*10, bf.NoJitter, time.Second, []float64{5, 7, 10}, rand.NewSource(0))
	dms.connector = connector
	// set static relay as needed connection
	relayInfo, err  := peer.AddrInfoFromString(StaticRelays[0])
	if err != nil {
		panic("failed to create message service: "+ err.Error())
	}
	relayInfo.Addrs = []ma.Multiaddr{}
	dms.connector.Need(direct.ServiceName, *relayInfo)
	dms.host.Network().Notify((*dmsNotifiee)(dms))
	go dms.background(context.Background(), dms.input)
	return dms
}

func (c *DirectMessaging) Send(nvlop *entity.Envelop) {
	c.input <- nvlop
}

// openStreamAndSend opens an stream and send proto message of envelop
func (c *DirectMessaging) openStreamAndSend(nvlop *entity.Envelop) error {
	log.Debug("open stream and send")
	nctx := network.WithUseTransient(context.Background(), "just a chat")
	pi := nvlop.PeerID()
	s, err := c.host.NewStream(nctx, pi, nvlop.Protocol)
	if err != nil {
		log.Error("send failed", err)
		return err
	}
	err = direct.Send(s, nvlop.Message.Proto())
	if err != nil {
		log.Error("send failed",err)
		return err
	}
	c.sendCompleted(nvlop)
	return nil
}


func (c *DirectMessaging) background(ctx context.Context, nvlpCh <-chan *entity.Envelop) {
	for {
		select {
		case m := <-c.outbox.failed:
			c.sendFailed(m)
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
			
			c.connector.Need(string(nvlp.Protocol), *pi)
			cns := h.Network().Connectedness(pi.ID)
			switch cns {
			case network.Connected:
				err := c.openStreamAndSend(nvlp)
				if err != nil {
					c.outbox.put(pi.ID, nvlp)
				}

			default:
				c.outbox.put(pi.ID, nvlp)
			}
		case e:= <-ctx.Done():
			log.Errorf("context error broke sender %v", e)
		}

	}
}

func (c *DirectMessaging) messageHandler(msg *pb.Message) {
	log.Debugf("message received ... %s", msg.GetText())
	event.EmitNewMessage(c.bus, entity.ToMessage(msg))
}

func (c *DirectMessaging) inviteHandler(msg *pb.Request) {
	log.Debugf("invite received %s", msg.Id)
	event.EmitInvite(c.bus, event.InviteReceived, entity.ToChatInfo(msg))
}

func (c *DirectMessaging) Stop() {
	c.host.RemoveStreamHandler(direct.ID)
}

func (c *DirectMessaging) sendCompleted(nvlop *entity.Envelop) {
	switch  msg := nvlop.Message.(type) {
	case entity.Message:
		event.EmitMessageChange(c.bus, entity.Sent, string(msg.ID))
	}
	c.connector.Done(string(nvlop.Protocol), nvlop.PeerID())
}

func (c *DirectMessaging) sendFailed(nvlop *entity.Envelop) {
	switch  msg := nvlop.Message.(type) {
	case entity.Message:
		event.EmitMessageChange(c.bus, entity.Failed, string(msg.ID))
	}
	c.connector.Done(string(nvlop.Protocol), nvlop.PeerID())
}

func (c *DirectMessaging) onConnected(pid peer.ID) {
	msgs := c.outbox.pop(pid)
	go func(msgs []*entity.Envelop) {
		for _, val := range msgs {
			err := c.openStreamAndSend(val)
			if err != nil {
				c.outbox.put(pid, val)
			}
		}
	}(msgs)

}

type dmsNotifiee DirectMessaging

func (pm *dmsNotifiee) dmService() *DirectMessaging {
	return (*DirectMessaging)(pm)
}

func (pm *dmsNotifiee) Listen(network.Network, ma.Multiaddr)       {}
func (pm *dmsNotifiee) ListenClose(network.Network, ma.Multiaddr)  {}
func (pm *dmsNotifiee) Disconnected(network.Network, network.Conn) {}
func (pm *dmsNotifiee) Connected(n network.Network, c network.Conn) {
	pm.dmService().onConnected(c.RemotePeer())
}
