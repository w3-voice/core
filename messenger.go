package core

import (
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hood-chat/core/entity"
	"github.com/hood-chat/core/pb"
	"github.com/hood-chat/core/repo"
	"github.com/hood-chat/core/store"
	"github.com/libp2p/go-libp2p-core/host"
)

type Envelop struct {
	Msg    entity.Message
	chatID string
	To     string
}

type Handler func(msg Envelop) error

type Messenger struct {
	Host     host.Host
	store    *store.Store
	identity entity.Identity
	pms      PMService
	hb       HostBuilder
	opt      Option
}

func MessengerBuilder(path string, opt Option, hb HostBuilder) Messenger {
	if hb != nil {
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

func (m Messenger) getMessageRepo(chatID string) repo.IRepo[entity.Message] {
	return repo.NewMessageRepo(m.store, chatID)
}

func (m *Messenger) Start() {
	m.opt.SetIdentity(&m.identity)
	h, err := m.hb.Create(m.opt)
	if err != nil {
		panic(err)
	}
	m.Host = h
	pms := NewPMService(h, m.MessageHandler)
	m.pms = *pms
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

func (m *Messenger) GetContact(id string) (entity.Contact, error) {
	rContact := m.getContactRepo()
	return rContact.GetByID(id)
}

func (m *Messenger) AddContact(c entity.Contact) error {
	rContact := m.getContactRepo()
	return rContact.Add(c)
}

func (m *Messenger) GetChat(id string) (entity.ChatInfo, error) {
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

func (m *Messenger) CreatePMChat(contactID string) (entity.ChatInfo, error) {
	c, err := m.GetContact(contactID)
	chatID := m.generatePMChatID(c)
	if err != nil {
		return entity.ChatInfo{}, err
	}
	chat := m.CreateChat(chatID, []entity.Contact{*m.identity.Me(), c}, c.Name)
	err = m.getChatRepo().Add(chat)
	return chat, err
}

func (m *Messenger) GetPMChat(contactID string) (entity.ChatInfo, error) {
	c, err := m.GetContact(contactID)
	chatID := m.generatePMChatID(c)
	if err != nil {
		return entity.ChatInfo{}, err
	}
	chat, err := m.getChatRepo().GetByID(chatID)
	return chat, err
}

func (m *Messenger) CreateChat(id string, members []entity.Contact, name string) entity.ChatInfo {
	return entity.ChatInfo{
		ID:      id,
		Name:    name,
		Members: members,
	}
}

func (c *Messenger) SendMessage(env Envelop) {
	c.pms.Send(&env)
}

func (m *Messenger) MessageHandler(msg *pb.Message) {
	rCon := m.getContactRepo()
	con, err := rCon.GetByID(msg.Author.Id)
	if err != nil {
		con = entity.Contact{
			ID:   msg.Author.Id,
			Name: msg.Author.Name,
		}
		err := rCon.Add(con)
		if err != nil {
			panic(err)
		}
	}

	rchat := m.getChatRepo()
	chat, err := rchat.GetByID(msg.GetChatId())
	if err != nil {
		chat = m.CreateChat(msg.GetChatId(), []entity.Contact{*m.identity.Me(), con}, msg.Author.Id)
		rchat.Add(chat)
	}
	newMsg := entity.Message{
		ID:        msg.Id,
		CreatedAt: time.Now(),
		Text:      msg.GetText(),
		Status:    entity.Sent,
		Author:    con,
	}
	rmsg := m.getMessageRepo(chat.ID)
	err = rmsg.Add(newMsg)
	if err != nil {
		panic(err)
	}
}

func (m *Messenger) NewMessage(chatID string, content string) (*Envelop, error) {

	msg := entity.Message{
		ID:        uuid.New().String(),
		CreatedAt: time.Now(),
		Text:      content,
		Status:    entity.Pending,
		Author:    *m.identity.Me(),
	}
	rmsg := m.getMessageRepo(chatID)
	err := rmsg.Add(msg)
	if err != nil {
		return nil, err
	}

	rchat := m.getChatRepo()
	chat, err := rchat.GetByID(chatID)
	to := []string{}
	for _, val := range chat.Members {
		if val.ID != msg.Author.ID {
			to = append(to, val.ID)
		}
	}
	m.SendMessage(Envelop{Msg: msg, To: to[0], chatID: chatID})
	return &Envelop{Msg: msg, To: to[0], chatID: chatID}, err
}

func (m *Messenger) GetMessages(chatID string) ([]entity.Message, error) {
	return m.getMessageRepo(chatID).GetAll()
}

func (m *Messenger) generatePMChatID(con entity.Contact) string {
	cons := []string{con.ID, m.identity.Me().ID}
	sort.Strings(cons)
	return strings.Join(cons, "")
}
