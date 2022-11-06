package repo

import (
	"errors"

	"github.com/hood-chat/core/entity"
	"github.com/hood-chat/core/store"
)

var ErrNotImplemented = errors.New("not implemented")
var ErrNotSupported = errors.New("not supported")

type IRepo[C any] interface {
	Get() (C, error)
	GetByID(id string) (C, error)
	GetAll() ([]C, error)
	Set(C) error
	Add(C) error
}

func NewChatRepo(store *store.Store) IRepo[entity.ChatInfo] {
	return ChatRepo{
		store: store,
	}
}

type ChatRepo struct {
	store *store.Store
}

func (c ChatRepo) GetAll() ([]entity.ChatInfo, error) {
	chl, err := c.store.ChatList()
	if err != nil {
		return nil, err
	}
	ci := make([]entity.ChatInfo, 0)
	for _, val := range chl {
		members := make([]entity.Contact, 0)
		m, _ := c.store.ContactByIDs(val.Members)
		for _, me := range m {
			members = append(members, entity.Contact{
				ID:   me.ID,
				Name: me.Name,
			})
		}
		ci = append(ci, entity.ChatInfo{
			ID:      val.ID,
			Name:    val.Name,
			Members: members,
		})
	}
	return ci, nil
}

func (c ChatRepo) GetByID(id string) (entity.ChatInfo, error) {
	ct, err := c.store.ChatByID(id)
	if err != nil {
		return entity.ChatInfo{}, err
	}
	members := make([]entity.Contact, 0)
	m, _ := c.store.ContactByIDs(ct.Members)
	for _, me := range m {
		members = append(members, entity.Contact{
			ID:   me.ID,
			Name: me.Name,
		})
	}
	return entity.ChatInfo{
		ID:      ct.ID,
		Name:    ct.Name,
		Members: members,
	}, nil
}

func (c ChatRepo) Add(chat entity.ChatInfo) error {
	m := []string{}
	for _, val := range chat.Members {
		m = append(m, val.ID)
	}
	ci := store.BHChat{
		ID:      chat.ID,
		Name:    chat.Name,
		Members: m,
	}
	err := c.store.InsertChat(ci)
	if err != nil {
		return err
	}
	return nil
}

func (c ChatRepo) Set(chat entity.ChatInfo) error {
	return ErrNotImplemented
}

func (c ChatRepo) Get() (entity.ChatInfo, error) {
	return entity.ChatInfo{}, ErrNotSupported
}

type MessageRepo struct {
	store  store.Store
	chatID string
}

func NewMessageRepo(store *store.Store, chatID string) IRepo[entity.Message] {
	return MessageRepo{
		store:  *store,
		chatID: chatID,
	}
}

func (m MessageRepo) Add(msg entity.Message) error {
	tmsg := store.BHTextMessage{
		ID:        msg.ID,
		ChatID:    m.chatID,
		CreatedAt: msg.CreatedAt,
		Text:      msg.Text,
		Status:    store.Status(msg.Status),
		Author:    store.BHContact(msg.Author),
	}
	err := m.store.InsertTextMessage(tmsg)
	if err != nil {
		return err
	}
	return nil
}
func (m MessageRepo) Set(msg entity.Message) error {
	return ErrNotImplemented
}
func (m MessageRepo) GetByID(id string) (entity.Message, error) {
	return entity.Message{}, ErrNotImplemented
}
func (m MessageRepo) GetAll() ([]entity.Message, error) {
	messages := make([]entity.Message, 0)
	bhm, err := m.store.ChatMessages(m.chatID)
	if err != nil {
		return nil, err
	}
	for _, m := range bhm {
		messages = append(messages, entity.Message{
			ID:        m.ID,
			CreatedAt: m.CreatedAt,
			Text:      m.Text,
			Status:    entity.Status(m.Status),
			Author: entity.Contact{
				ID:   m.Author.ID,
				Name: m.Author.Name,
			},
		})
	}
	return messages, nil
}

func (m MessageRepo) Get() (entity.Message, error) {
	return entity.Message{}, ErrNotSupported
}

type ContactRepo struct {
	store store.Store
}

func NewContactRepo(store *store.Store) IRepo[entity.Contact] {
	return ContactRepo{
		store: *store,
	}
}

func (c ContactRepo) Add(con entity.Contact) error {
	err := c.store.InsertContact(store.BHContact{
		Name: con.Name,
		ID:   con.ID,
	})
	if err != nil {
		return err
	}
	return nil
}
func (c ContactRepo) Set(cont entity.Contact) error {
	return ErrNotImplemented
}
func (c ContactRepo) GetByID(id string) (entity.Contact, error) {
	con, err := c.store.ContactByID(id)
	if err != nil {
		return entity.Contact{}, err
	}
	return entity.Contact{
		ID:   con.ID,
		Name: con.Name,
	}, nil
}
func (c ContactRepo) GetAll() ([]entity.Contact, error) {
	cons := make([]entity.Contact, 0)
	bhcl, err := c.store.AllContacts()
	if err != nil {
		return nil, err
	}
	for _, val := range bhcl {
		cons = append(cons, entity.Contact{
			Name: val.Name,
			ID:   val.ID,
		})
	}
	return cons, nil
}

func (c ContactRepo) Get() (entity.Contact, error) {
	return entity.Contact{}, ErrNotSupported
}

type IdentityRepo struct {
	store store.Store
}

func NewIdentityRepo(store *store.Store) IRepo[entity.Identity] {
	return IdentityRepo{
		store: *store,
	}
}

func (i IdentityRepo) Add(con entity.Identity) error {
	return ErrNotImplemented
}
func (i IdentityRepo) Set(iden entity.Identity) error {
	err := i.store.SetIdentity(store.BHIdentity{
		ID:   iden.ID,
		Name: iden.Name,
		Key:  iden.PrivKey,
	})
	if err != nil {
		return err
	}
	return nil
}
func (i IdentityRepo) GetByID(id string) (entity.Identity, error) {
	return entity.Identity{}, ErrNotImplemented
}
func (i IdentityRepo) GetAll() ([]entity.Identity, error) {
	id, err := i.store.GetIdentity()
	if err != nil {
		return nil, err
	}
	return []entity.Identity{{
		ID:      id.ID,
		Name:    id.Name,
		PrivKey: id.Key,
	}}, nil
}

func (i IdentityRepo) Get() (entity.Identity, error) {
	id, err := i.store.GetIdentity()
	if err != nil {
		return entity.Identity{}, err
	}
	return entity.Identity{
		ID:      id.ID,
		Name:    id.Name,
		PrivKey: id.Key,
	}, nil
}
