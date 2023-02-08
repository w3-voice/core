package core

import (
	"bytes"
	"context"
	"time"

	"github.com/hood-chat/core/entity"
	"github.com/hood-chat/core/event"
	"github.com/hood-chat/core/pb"
	ma "github.com/multiformats/go-multiaddr"
	"google.golang.org/protobuf/proto"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

var _ PubSubService = (*GPService)(nil)

const (
	JOINED = "joined"
)

type GPService struct {
	ctx       context.Context
	h         host.Host
	connector Connector
	id        IdentityAPI
	rch       chan string
	rooms     map[string]*ChatRoom
	ps        *pubsub.PubSub
	bus       Bus
}

func NewGPService(ctx context.Context, h host.Host, identity IdentityAPI, b Bus, ch chan string, c Connector) PubSubService {
	gpService := new(GPService)
	gpService.ctx = ctx
	gpService.rch = ch
	gpService.connector = c
	gpService.h = h
	gpService.bus = b

	gpService.id = identity

	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		panic(err)
	}
	gpService.ps = ps

	gpService.rooms = make(map[string]*ChatRoom)
	go func() {
		for chID := range gpService.rch {
			gpService.JoinChatRoom(chID)
		}
	}()

	return gpService
}

func (s *GPService) OnlinePeers(chatID string) []peer.ID {
	return s.ps.ListPeers(topicName(s.rooms[chatID].roomName))
}

func (s *GPService) Send(n PubSubEnvelop) {
	room, pres := s.rooms[n.Topic]
	switch n.Message.(type) {
	case entity.Message:
		msg := n.Message.(entity.Message)
		if !pres {
			event.EmitMessageChange(s.bus, entity.Failed, string(msg.ID))
			log.Errorf("chatroom not present")
			return
		}

		err := room.send(n.Message.Proto())
		if err != nil {
			log.Errorf("send failed : %v", err)
			event.EmitMessageChange(s.bus, entity.Failed, string(msg.ID))
			return
		}
		log.Debugf("message send : %s", string(msg.ID))
		event.EmitMessageChange(s.bus, entity.Sent, string(msg.ID))
	default:
		return
	}

}

func (s *GPService) Join(ChatID entity.ID, admins []entity.Contact) {
	for _, v := range admins {
		adder, _ := v.AdderInfo()
		s.h.Peerstore().AddAddrs(adder.ID,
			[]ma.Multiaddr{
				ma.StringCast("/p2p/" + "12D3KooWBFpA7pCMBySBqtduBVkakVQ3bmmaeagB83WHoruBN9s9" + "/p2p-circuit/p2p/" + adder.ID.String()),
				ma.StringCast("/p2p/" + "12D3KooWBFpA7pCMBySBqtduBVkakVQ3bmmaeagB83WHoruBN9s9" + "/p2p-circuit/p2p/" + adder.ID.String()),
			},
			time.Minute*5)
		s.connector.Need(ChatID.String(), *adder)
	}

	s.JoinChatRoom(ChatID.String())
}

// JoinChatRoom tries to subscribe to the PubSub topic for the room name, returning
// a ChatRoom on success.
func (s *GPService) JoinChatRoom(chatID string) error {
	// join the pubsub topic
	topic, err := s.ps.Join(topicName(chatID))
	if err != nil {
		return err
	}

	// and subscribe to it
	sub, err := topic.Subscribe()
	if err != nil {
		return err
	}

	self, err := s.id.PeerID()
	if err != nil {
		return err
	}
	cr := &ChatRoom{
		ctx:   s.ctx,
		topic: topic,
		sub:   sub,
		self:  self,
	}

	s.rooms[chatID] = cr
	// start reading messages from the subscription in a loop
	go cr.readLoop(s.bus)
	cr.topic.Publish(cr.ctx, []byte(JOINED))
	return nil
}

func (s *GPService) Stop() {
	s.ctx.Done()
}

// ChatRoom represents a subscription to a single PubSub topic. Messages
// can be published to the topic with ChatRoom.Publish, and received
// messages are pushed to the Messages channel.
type ChatRoom struct {
	// Messages is a channel of messages received from other peers in the chat room
	ctx   context.Context
	topic *pubsub.Topic
	sub   *pubsub.Subscription

	roomName string
	self     peer.ID
}

// Publish sends a message to the pubsub topic.
func (cr *ChatRoom) send(pbmsg proto.Message) error {
	msgBytes, err := proto.Marshal(pbmsg)
	if err != nil {
		return err
	}
	return cr.topic.Publish(cr.ctx, msgBytes)
}

// readLoop pulls messages from the pubsub topic and pushes them onto the Messages channel.
func (cr *ChatRoom) readLoop(bus Bus) {
	for {
		msg, err := cr.sub.Next(cr.ctx)
		if err != nil {
			return
		}
		// only forward messages delivered by others
		if msg.ReceivedFrom == cr.self {
			continue
		}

		if bytes.Equal(msg.Data, []byte(JOINED)) {
			log.Debugf("new peer joined: %v", msg.ID)
			continue
		}
		log.Debugf("new group message arrived: %v", msg.ID)
		cm := new(pb.Message)
		err = proto.Unmarshal(msg.Data, cm)
		if err != nil {
			continue
		}
		// send valid messages onto the Messages channel

		event.EmitNewMessage(bus, entity.ToMessage(cm))
		if err != nil {
			panic("bus not working")
		}
	}
}

func topicName(roomName string) string {
	return "chat-room:" + roomName
}
