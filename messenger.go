package core

import (
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hood-chat/core/entity"
	"github.com/hood-chat/core/event"
	"github.com/hood-chat/core/pb"
	"github.com/hood-chat/core/repo"
	"github.com/hood-chat/core/store"
	logging "github.com/ipfs/go-log/v2"
	lpevt "github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/p2p/host/eventbus"
)

var log = logging.Logger("msgr-core")

type Messenger struct {
	Host     host.Host
	store    *store.Store
	identity entity.Identity
	pms      PMService
	hb       HostBuilder
	opt      Option
	bus      lpevt.Bus
}

func MessengerBuilder(path string, opt Option, hb HostBuilder) Messenger {
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
	rIdentity := repo.NewIdentityRepo(s)
	id, err := rIdentity.Get()
	if err != nil {
		return msgr
	}
	msgr.identity = id

	msgr.Start()
	return msgr
}

func (m Messenger) getContactRepo() repo.IRepo[entity.Contact] {
	return repo.NewContactRepo(m.store)
}

func (m Messenger) getIdentityRepo() repo.IRepo[entity.Identity] {
	return repo.NewIdentityRepo(m.store)
}

func (m Messenger) getChatRepo() repo.IRepo[entity.ChatInfo] {
	return repo.NewChatRepo(m.store)
}

func (m Messenger) getMessageRepo() repo.IRepo[entity.Message] {
	return repo.NewMessageRepo(m.store)
}

func (m *Messenger) Start() {
	m.opt.SetIdentity(&m.identity)
	h, err := m.hb.Create(m.opt)
	if err != nil {
		panic(err)
	}
	m.Host = h
	pms := NewPMService(h, m.bus)
	m.pms = *pms

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
			m.MessageHandler(msg)
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
				if *evt.Action() == entity.Sent {
					m.updateMessageStatus(*evt.Payload(), *evt.Action())
				}
			}

		}
	}()
}

func (m *Messenger) IsLogin() bool {
	rIdentity := m.getIdentityRepo()
	_, err := rIdentity.Get()
	return err == nil
}

func (m *Messenger) SignUp(name string) (*entity.Identity, error) {
	rIdentity := m.getIdentityRepo()
	iden, err := entity.CreateIdentity(name)
	if err != nil {
		return nil, err
	}
	err = rIdentity.Set(iden)
	m.identity = iden
	if err != nil {
		return nil, err
	}
	m.Start()
	return &iden, nil
}

func (m *Messenger) GetIdentity() (entity.Identity, error) {
	return m.identity, nil
}

func (m *Messenger) GetContacts() ([]entity.Contact, error) {
	rContact := m.getContactRepo()
	return rContact.GetAll(nil)
}

func (m *Messenger) GetContact(id entity.ID) (entity.Contact, error) {
	rContact := m.getContactRepo()
	return rContact.GetByID(id)
}

func (m *Messenger) AddContact(c entity.Contact) error {
	rContact := m.getContactRepo()
	return rContact.Add(c)
}

func (m *Messenger) GetChat(id entity.ID) (entity.ChatInfo, error) {
	rChat := m.getChatRepo()
	ci := entity.ChatInfo{}
	c, err := rChat.GetByID(id)
	if err != nil {
		return ci, err
	}
	return c, nil
}

func (m *Messenger) GetChats() ([]entity.ChatInfo, error) {
	rChat := repo.NewChatRepo(m.store)
	return rChat.GetAll(nil)
}

func (m *Messenger) CreatePMChat(contactID entity.ID) (entity.ChatInfo, error) {
	c, err := m.GetContact(contactID)
	chatID := m.generatePMChatID(c)
	if err != nil {
		return entity.ChatInfo{}, err
	}
	chat := m.CreateChat(chatID, []entity.Contact{*m.identity.Me(), c}, c.Name)
	err = m.getChatRepo().Add(chat)
	return chat, err
}

func (m *Messenger) GetPMChat(contactID entity.ID) (entity.ChatInfo, error) {
	c, err := m.GetContact(contactID)
	chatID := m.generatePMChatID(c)
	if err != nil {
		return entity.ChatInfo{}, err
	}
	chat, err := m.getChatRepo().GetByID(chatID)
	return chat, err
}

func (m *Messenger) CreateChat(id entity.ID, members []entity.Contact, name string) entity.ChatInfo {
	return entity.ChatInfo{
		ID:      entity.ID(id),
		Name:    name,
		Members: members,
	}
}

func (m *Messenger) MessageHandler(msg *pb.Message) {
	mAuthorID := entity.ID(msg.Author.Id)
	msgID := entity.ID(msg.GetId())
	chatID := entity.ID(msg.ChatId)
	rCon := m.getContactRepo()
	con, err := rCon.GetByID(mAuthorID)
	if err != nil {
		log.Errorf("fail to get contact %s", err.Error())
		con = entity.Contact{
			ID:   mAuthorID,
			Name: msg.Author.Name,
		}
		err := rCon.Add(con)
		if err != nil {
			log.Errorf("fail to add contact %s", err.Error())
			return
		}
	}

	rchat := m.getChatRepo()
	chat, err := rchat.GetByID(entity.ID(msg.GetChatId()))

	if err != nil {
		log.Errorf("can not find chat %s", err.Error())
		chat = m.CreateChat(chatID, []entity.Contact{*m.identity.Me(), con}, msg.Author.Name)
		err := rchat.Add(chat)
		if err != nil {
			log.Errorf("fail to handle new message %s", err.Error())
			return
		}
	}
	newMsg := entity.Message{
		ID:        msgID,
		ChatID:    chat.ID,
		CreatedAt: time.Unix(msg.GetCreatedAt(), 0),
		Text:      msg.GetText(),
		Status:    entity.Sent,
		Author:    con,
	}
	rmsg := m.getMessageRepo()
	err = rmsg.Add(newMsg)
	log.Debugf("new message %s ", newMsg)
	if err != nil {
		log.Errorf("Can not add message %s , %d", err.Error(), newMsg)
		return
	}

	em, _ := m.bus.Emitter(new(event.EvtObject))
	defer em.Close()
	evgrp := event.NewMessagingEventGroup()
	ev, _ := evgrp.Make("ChangeMessageStatus", entity.Received, msgID)
	em.Emit(*ev)
}

func (m *Messenger) SendPM(chatID entity.ID, content string) (*entity.Message, error) {
	msg := entity.Message{
		ID:        entity.ID(uuid.New().String()),
		ChatID:    chatID,
		CreatedAt: time.Now().UTC(),
		Text:      content,
		Status:    entity.Pending,
		Author:    *m.identity.Me(),
	}
	rmsg := m.getMessageRepo()
	err := rmsg.Add(msg)
	if err != nil {
		log.Errorf("Can not add message %s", err.Error())
		return nil, err
	}
	rchat := m.getChatRepo()
	chat, err := rchat.GetByID(chatID)
	if err != nil {
		log.Errorf("Can not get chat %s", err.Error())
		return nil, err
	}
	to := []entity.ID{}
	for _, val := range chat.Members {
		if val.ID != msg.Author.ID {
			to = append(to, val.ID)
		}
	}
	pbmsg := &pb.Message{
		Text:      msg.Text,
		Id:        msg.ID.String(),
		ChatId:    chatID.String(),
		CreatedAt: msg.CreatedAt.Unix(),
		Type:      "text",
		Sig:       "",
		Author: &pb.Contact{
			Id:   msg.Author.ID.String(),
			Name: msg.Author.Name,
		},
	}
	go m.pms.Send(pbmsg, to[0])
	return &msg, nil
}

func (m *Messenger) GetMessages(chatID entity.ID) ([]entity.Message, error) {
	filter := make(repo.Filter, 0)
	filter["chatID"] = string(chatID)
	return m.getMessageRepo().GetAll(filter)
}

func (m *Messenger) generatePMChatID(con entity.Contact) entity.ID {
	cons := []string{con.ID.String(), m.identity.Me().ID.String()}
	sort.Strings(cons)
	return entity.ID(strings.Join(cons, ""))
}

func (m *Messenger) updateMessageStatus(msgID entity.ID, status entity.Status) error {
	rmsg := m.getMessageRepo()
	msg, err := rmsg.GetByID(msgID)
	if err != nil {
		return err
	}
	msg.Status = status
	err = rmsg.Set(msg)
	if err != nil {
		return err
	}
	return nil
}

func (m *Messenger) EventBus() lpevt.Bus {
	return m.bus
}
