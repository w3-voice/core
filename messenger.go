package core

import (
	"github.com/hood-chat/core/entity"
	"github.com/hood-chat/core/event"
	"github.com/hood-chat/core/pb"
	"github.com/hood-chat/core/store"
	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/p2p/host/eventbus"
)

var log = logging.Logger("msgr-core")

var _ MessengerAPI = (*Messenger)(nil)

type Messenger struct {
	Host     host.Host
	store    *store.Store
	identity IdentityAPI
	book     ContactBookAPI
	pms      PMService
	chat     ChatAPI
	hb       Builder
	opt      Option
	bus      Bus
}

func NewMessengerAPI(path string, opt Option, hb Builder) MessengerAPI {
	if hb == nil {
		hb = DefaultRoutedHost{}
	}
	msgr := Messenger{
		bus: eventbus.NewBus(),
		hb:  hb,
		opt: opt,
	}

	err := checkWritable(path)
	if err != nil {
		panic("path is not writable ")
	}
	s, err := store.NewStore(path + "/store")
	if err != nil {
		panic(err)
	}
	msgr.store = s
	msgr.book = NewContactBook(s)
	identity := NewIdentityAPI(s)
	msgr.identity = identity
	if !identity.IsLogin() {
		return &msgr
	}
	msgr.Start()
	return &msgr
}

func (m *Messenger) Start() {
	id, err := m.identity.Get()
	if err != nil {
		panic(err)
	}
	m.opt.SetIdentity(&id)
	h, err := m.hb.Create(m.opt)
	if err != nil {
		panic(err)
	}
	m.Host = h
	m.pms = NewPMService(h, m.bus)
	m.chat = NewChatAPI(m.store, m.book, m.pms, m.identity)

	sub, err := m.bus.Subscribe(new(event.EvtMessageReceived))
	if err != nil {
		panic(err)
	}
	go func() {
		defer sub.Close()
		for e := range sub.Out() {
			log.Debug("EvtMessageReceived received")
			msg := e.(event.EvtMessageReceived).Msg
			log.Debugf("EvtMessageReceived received %s", msg)
			m.messageHandler(msg)
		}
	}()
	subStaus, err := m.bus.Subscribe(new(event.EvtObject))
	if err != nil {
		panic(err)
	}
	go func() {
		defer subStaus.Close()
		for e := range subStaus.Out() {
			log.Debug("EvtObject received")
			evt := e.(event.EvtObject)
			meg := event.NewMessagingEventGroup()
			if meg.Validate(&evt) {
				evt, _ := meg.Parse(&evt)
				log.Debugf("MessagingEvent received %s", evt)
				if *evt.Action() == entity.Sent || *evt.Action() == entity.Failed {
					m.chat.updateMessageStatus(*evt.Payload(), *evt.Action())
				}
			}
		}
	}()
}

func (m *Messenger) messageHandler(msg *pb.Message) {
	err := m.chat.received(msg)
	if err != nil {
		return
	}
	em, _ := m.bus.Emitter(new(event.EvtObject))
	defer em.Close()
	evgrp := event.NewMessagingEventGroup()
	ev, _ := evgrp.Make("ChangeMessageStatus", entity.Received, entity.ID(msg.Id))
	em.Emit(*ev)
}

func (m *Messenger) ContactBookAPI() ContactBookAPI {
	return m.book
}

func (m *Messenger) IdentityAPI() IdentityAPI {
	return m.identity
}

func (m *Messenger) ChatAPI() ChatAPI {
	return m.chat
}

func (m *Messenger) EventBus() Bus {
	return m.bus
}

func (m *Messenger) Stop() {
	m.store.Close()
	m.pms.Stop()
	m.Host.Close()
}
