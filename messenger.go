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
	"github.com/libp2p/go-libp2p/core/host"
)

var log = logging.Logger("msgr-core")

type Messenger struct {
	Host     host.Host
	store    *store.Store
	identity entity.Identity
	pms      PMService
	hb       HostBuilder
	opt      Option
}

func MessengerBuilder(path string, opt Option, hb HostBuilder) Messenger {
	if hb == nil {
		hb = DefaultRoutedHost{}
	}

	err := checkWritable(path)
	if err != nil {
		panic("path is not writable ")
	}
	s, err := store.NewStore(path + "/store")
	if err != nil {
		panic(err)
	}
	rIdentity := repo.NewIdentityRepo(s)
	id, err := rIdentity.Get()
	if err != nil {
		return Messenger{
			store: s,
			hb:    hb,
			opt:   opt,
		}
	}
	m := Messenger{
		store:    s,
		identity: id,
		hb:       hb,
		opt:      opt,
	}

	m.Start()
	return m
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

func (m Messenger) getMessageRepo(chatID entity.ID) repo.IRepo[entity.Message] {
	return repo.NewMessageRepo(m.store, chatID)
}

func (m *Messenger) Start() {
	m.opt.SetIdentity(&m.identity)
	h, err := m.hb.Create(m.opt)
	if err != nil {
		panic(err)
	}
	m.Host = h
	pms := NewPMService(h)
	m.pms = *pms
	sub, err := h.EventBus().Subscribe(new(event.EvtMessageReceived))
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
	return rContact.GetAll()
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
	return rChat.GetAll()
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
			log.Errorf("can not find chat %s", err.Error())
			return
		}
	}
	newMsg := entity.Message{
		ID:        msgID,
		CreatedAt: time.Unix(msg.GetCreatedAt(), 0),
		Text:      msg.GetText(),
		Status:    entity.Sent,
		Author:    con,
	}
	rmsg := m.getMessageRepo(chat.ID)
	err = rmsg.Add(newMsg)
	log.Debugf("new message %s ", newMsg)
	if err != nil {
		log.Errorf("Can not add message %s , %d", err.Error(), newMsg)
		return
	}
}

func (m *Messenger) SendPM(chatID entity.ID, content string) (*entity.Message, error) {
	msg := entity.Message{
		ID:        entity.ID(uuid.New().String()),
		CreatedAt: time.Now().UTC(),
		Text:      content,
		Status:    entity.Pending,
		Author:    *m.identity.Me(),
	}
	rmsg := m.getMessageRepo(chatID)
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
	return m.getMessageRepo(chatID).GetAll()
}

func (m *Messenger) generatePMChatID(con entity.Contact) entity.ID {
	cons := []string{con.ID.String(), m.identity.Me().ID.String()}
	sort.Strings(cons)
	return entity.ID(strings.Join(cons, ""))
}
