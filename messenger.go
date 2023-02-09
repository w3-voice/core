package core

import (

	"github.com/hood-chat/core/entity"
	"github.com/hood-chat/core/event"
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
	pms      DirectService
	// gps      PubSubService
	chat     ChatAPI
	hb       Builder
	opt      Option
	bus      Bus
	connector Connector
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
	m.connector = NewConnector(h)
	gpCh := make(chan string)
	m.pms = NewDirectMessaging(h, m.bus, m.connector,make(chan*Envelop))
	// m.gps = NewGPService(context.Background(),h,m.IdentityAPI(),m.bus,gpCh,m.connector)
	m.chat = NewChatAPI(m.store, m.book, m.pms, m.identity)

	chats,err := m.chat.ChatInfos(0,0)
	if err != nil {
		panic("cant load groups chats")
	}
	for _, c := range chats {
		if c.Type == entity.Group {
			gpCh <- c.ID.String()
		}
	}

	msgSub, err := m.bus.Subscribe(new(event.MessageEventObj))
	if err != nil {
		panic(err)
	}
	go func() {
		defer msgSub.Close()
		for e := range msgSub.Out() {
			evt := e.(event.MessageEventObj)
			switch msg := e.(event.MessageEventObj).GetPayload().(type) {
			case entity.ID:
				if evt.GetAction() == entity.Sent || evt.GetAction() == entity.Failed {
					m.chat.updateMessageStatus(msg, evt.GetAction())
				}
			case entity.Message:
				m.messageHandler(msg)
			}
		}
	}()

	chatSub, err := m.bus.Subscribe(new(event.ChatEventObj))
	if err != nil {
		panic(err)
	}
	go func() {
		defer chatSub.Close()
		for e := range chatSub.Out() {
			evt := e.(event.ChatEventObj)
			switch evt.GetAction() {
			case event.InviteReceived:
				m.ChatAPI().Join(evt.GetPayload().(entity.ChatInfo))
			case event.InviteSent:
				// change Chat info so it keep track of chat request's
				break;
			}
		}
	}()
}

func (m *Messenger) messageHandler(msg entity.Message) {
	err := m.chat.received(msg)
	if err != nil {
		return
	}
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
