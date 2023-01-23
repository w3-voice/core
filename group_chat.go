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
	"github.com/libp2p/go-libp2p/p2p/host/eventbus"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
)


var _ GroupChatService = (*GPService)(nil)

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
	emitters  struct {
		evtMessageReceived      Emitter
		evtMessageStatusChanged Emitter
	}
}

func NewGPService(ctx context.Context,h host.Host, identity IdentityAPI, b Bus, ch chan string, c Connector) GroupChatService {
	gpService := new(GPService)
	gpService.ctx = ctx
	gpService.rch = ch
	gpService.connector = c
	gpService.h=h

	gpService.id = identity

	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		panic(err)
	}
	gpService.ps = ps

	gpService.emitters.evtMessageStatusChanged, err = b.Emitter(new(event.EvtObject), eventbus.Stateful)
	if err != nil {
		log.Errorf("error reading message: %s", err.Error())
		panic("failed to create message service")
	}
	gpService.emitters.evtMessageReceived, err = b.Emitter(new(event.EvtMessageReceived), eventbus.Stateful)
	if err != nil {
		log.Errorf("error reading message: %s", err.Error())
		panic("failed to create message service")
	}

	gpService.rooms = make(map[string]*ChatRoom)
	go func() {
		for chID := range gpService.rch {
			gpService.JoinChatRoom(chID, gpService.emitters.evtMessageReceived)
		}
	}()

	return gpService
}

func (s *GPService) OnlinePeers(chatID string) []peer.ID {
	return s.ps.ListPeers(topicName(s.rooms[chatID].roomName))
}

func (s *GPService) Send(n entity.Envelop) {
	room, pres := s.rooms[string(n.To.ID)]
	if !pres {
		log.Errorf("chatroom not present")
		event.EmitMessageChange(s.emitters.evtMessageStatusChanged, entity.Failed, string(n.Message.ID))
		return
	}

	err := room.send(n.Message)
	if err != nil {
		log.Errorf("send failed : %v", err)
		event.EmitMessageChange(s.emitters.evtMessageStatusChanged, entity.Failed, string(n.Message.ID))
		return
	}
	log.Debugf("message send : %s", string(n.Message.ID))
	event.EmitMessageChange(s.emitters.evtMessageStatusChanged, entity.Sent, string(n.Message.ID))
}

func (s *GPService) Join(ChatID entity.ID, members []entity.Contact) {
	for _,v := range members {
		adder, _ := v.AdderInfo()
		s.h.Peerstore().AddAddrs(adder.ID,
			[]ma.Multiaddr{
				ma.StringCast("/p2p/" + "12D3KooWBFpA7pCMBySBqtduBVkakVQ3bmmaeagB83WHoruBN9s9" + "/p2p-circuit/p2p/" + adder.ID.String()),
				ma.StringCast("/p2p/" + "12D3KooWBFpA7pCMBySBqtduBVkakVQ3bmmaeagB83WHoruBN9s9" + "/p2p-circuit/p2p/" + adder.ID.String()),
			},
			time.Minute*5)
		s.connector.Need(ChatID.String(), *adder)
	}
	
	s.JoinChatRoom(ChatID.String(), s.emitters.evtMessageReceived)
}


// JoinChatRoom tries to subscribe to the PubSub topic for the room name, returning
// a ChatRoom on success.
func (s *GPService) JoinChatRoom(chatID string, emitter Emitter) error {
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

	self,err := s.id.PeerID()
	if err != nil {
		return err
	}
	cr := &ChatRoom{
		ctx:      s.ctx,
		topic:    topic,
		sub:      sub,
		self:     self,
	}


	s.rooms[chatID]=cr
	// start reading messages from the subscription in a loop
	go cr.readLoop(emitter)
	cr.topic.Publish(cr.ctx, []byte(JOINED))
	return nil
}

func (s *GPService) Stop() {
	s.emitters.evtMessageReceived.Close()
	s.emitters.evtMessageStatusChanged.Close()
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
func (cr *ChatRoom) send(message entity.Message) error {
	msgBytes, err := proto.Marshal(message.Proto())
	if err != nil {
		return err
	}
	return cr.topic.Publish(cr.ctx, msgBytes)
}


// readLoop pulls messages from the pubsub topic and pushes them onto the Messages channel.
func (cr *ChatRoom) readLoop(em Emitter) {
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

		err = em.Emit(event.EvtMessageReceived{Msg: cm})
		if err != nil {
			panic("bus not working")
		}
	}
}

func topicName(roomName string) string {
	return "chat-room:" + roomName
}

