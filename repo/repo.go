package repo

import (
	"errors"

	"github.com/hood-chat/core/entity"
	"github.com/hood-chat/core/store"
)

var ErrNotImplemented = errors.New("not implemented")
var ErrNotSupported = errors.New("not supported")


//TODO: after adding IOption look like we can remove GetAll and GetByID
type IRepo[C any] interface {
	Get() (C, error)
	GetByID(id entity.ID) (C, error)
	GetAll(opt IOption) ([]C, error)
	Put(C) error
	Add(C) error
}

type IOption interface {
	Skip() int
	Limit() int
	AddFilter(field string, value interface{})
	Filters() Filter
}

type Filter map[string]interface{}

type Option struct {
	skip    int
	limit   int
	filters Filter
}

func NewOption(skip int, limit int) IOption {
	return Option{skip, limit, make(Filter)}
}

func (o Option) Skip() int {
	return o.skip
}

func (o Option) Limit() int {
	return o.limit
}

func (o Option) AddFilter(n string, s interface{}) {
	o.filters[n] = s
}

func (o Option) Filters() Filter {
	return o.filters
}

func NewChatRepo(store *store.Store) IRepo[entity.ChatInfo] {
	return ChatRepo{
		store: store,
	}
}

type ChatRepo struct {
	store *store.Store
}

func (c ChatRepo) GetAll(opt IOption) ([]entity.ChatInfo, error) {
	chl, err := c.store.ChatList(opt.Skip(), opt.Limit())
	if err != nil {
		return nil, err
	}
	ci := make([]entity.ChatInfo, 0)
	for _, val := range chl {
		unread, err := c.store.ChatUnreadCount(val.ID);if err != nil {unread = 0}

		latestText := ""
		latestMsg, err := c.store.ChatMessages(val.ID,0,1);
		if err == nil && len(latestMsg) > 0 {
			latestText = latestMsg[0].Text
		}

		members := make([]entity.Contact, 0)
		for _, me := range val.Members {
			members = append(members, entity.Contact{
				ID:   entity.ID(me.ID),
				Name: me.Name,
			})
		}
		ci = append(ci, entity.ChatInfo{
			ID:      entity.ID(val.ID),
			Name:    val.Name,
			Members: members,
			Type: val.Type,
			Unread: unread,
			LatestText: latestText,
		})
	}
	return ci, nil
}

func (c ChatRepo) GetByID(id entity.ID) (entity.ChatInfo, error) {
	ct, err := c.store.ChatByID(string(id))
	if err != nil {
		return entity.ChatInfo{}, err
	}
	unread, err := c.store.ChatUnreadCount(string(id));if err != nil {unread = 0}

	latestText := ""
	latestMsg, err := c.store.ChatMessages(id.String(),0,1);
	if err == nil && len(latestMsg) > 0 {
		latestText = latestMsg[0].Text
	}

	members := make([]entity.Contact, 0)
	for _, me := range ct.Members {
		members = append(members, entity.Contact{
			ID:   entity.ID(me.ID),
			Name: me.Name,
		})
	}
	return entity.ChatInfo{
		ID:      entity.ID(ct.ID),
		Name:    ct.Name,
		Members: members,
		Type:    ct.Type,
		Unread: unread,
		LatestText: latestText,
	}, nil
}

func (c ChatRepo) Add(chat entity.ChatInfo) error {
	m := []store.BHContact{}
	for _, val := range chat.Members {
		m = append(m, store.BHContact{ID: val.ID.String(),Name: val.Name})
	}
	ci := store.BHChat{
		ID:      string(chat.ID),
		Name:    chat.Name,
		Members: m,
		Type:    chat.Type,
	}
	err := c.store.InsertChat(ci)
	if err != nil {
		return err
	}
	return nil
}

func (c ChatRepo) Put(chat entity.ChatInfo) error {
	return ErrNotImplemented
}

func (c ChatRepo) Get() (entity.ChatInfo, error) {
	return entity.ChatInfo{}, ErrNotSupported
}

type MessageRepo struct {
	store store.Store
}

func NewMessageRepo(store *store.Store) IRepo[entity.Message] {
	return MessageRepo{
		store: *store,
	}
}

func (m MessageRepo) Add(msg entity.Message) error {
	tmsg := store.BHTextMessage{
		ID:        string(msg.ID),
		ChatID:    string(msg.ChatID),
		CreatedAt: msg.CreatedAt,
		Text:      msg.Text,
		Status:    entity.Status(msg.Status),
		Author:    store.BHContact{Name: msg.Author.Name, ID: string(msg.Author.ID)},
	}
	err := m.store.InsertTextMessage(tmsg)
	if err != nil {
		return err
	}
	return nil
}
func (m MessageRepo) Put(msg entity.Message) error {
	tmsg := store.BHTextMessage{
		ID:        string(msg.ID),
		ChatID:    string(msg.ChatID),
		CreatedAt: msg.CreatedAt,
		Text:      msg.Text,
		Status:    entity.Status(msg.Status),
		Author:    store.BHContact{Name: msg.Author.Name, ID: string(msg.Author.ID)},
	}
	return m.store.UpdateMessage(tmsg)
}
func (m MessageRepo) GetByID(id entity.ID) (entity.Message, error) {
	bhmsg, err := m.store.MsgByID(id.String())
	if err != nil {
		return entity.Message{}, err
	}
	msg := entity.Message{
		ID:        entity.ID(bhmsg.ID),
		ChatID:    entity.ID(bhmsg.ChatID),
		CreatedAt: bhmsg.CreatedAt,
		Text:      bhmsg.Text,
		Status:    entity.Status(bhmsg.Status),
		Author: entity.Contact{
			ID:   entity.ID(bhmsg.Author.ID),
			Name: bhmsg.Author.Name,
		},
	}
	return msg, nil
}
func (m MessageRepo) GetAll(opt IOption) ([]entity.Message, error) {
	messages := make([]entity.Message, 0)
	chID, pres := opt.Filters()["ChatID"].(string)
	if !pres {
		return nil, ErrNotSupported
	}
	status, _ := opt.Filters()["Status"].([]entity.Status)
	bhm, err := m.store.ChatMessages(string(chID), opt.Skip(), opt.Limit(), status...)
	if err != nil {
		return nil, err
	}
	for _, m := range bhm {
		messages = append(messages, entity.Message{
			ID:        entity.ID(m.ID),
			ChatID:    entity.ID(m.ChatID),
			CreatedAt: m.CreatedAt,
			Text:      m.Text,
			Status:    entity.Status(m.Status),
			Author: entity.Contact{
				ID:   entity.ID(m.Author.ID),
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
	return c.store.InsertContact(store.BHContact{
		Name: con.Name,
		ID:   string(con.ID),
	})
}

func (c ContactRepo) Put(con entity.Contact) error {
	return c.store.PutContact(store.BHContact{
		Name: con.Name,
		ID:   string(con.ID),
	})
}

func (c ContactRepo) GetByID(id entity.ID) (entity.Contact, error) {
	con, err := c.store.ContactByID(string(id))
	if err != nil {
		return entity.Contact{}, err
	}
	return entity.Contact{
		ID:   entity.ID(con.ID),
		Name: con.Name,
	}, nil
}
func (c ContactRepo) GetAll(opt IOption) ([]entity.Contact, error) {
	cons := make([]entity.Contact, 0)
	bhcl, err := c.store.AllContacts(opt.Skip(), opt.Limit())
	if err != nil {
		return nil, err
	}
	for _, val := range bhcl {
		cons = append(cons, entity.Contact{
			Name: val.Name,
			ID:   entity.ID(val.ID),
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
func (i IdentityRepo) Put(iden entity.Identity) error {
	err := i.store.SetIdentity(store.BHIdentity{
		ID:   string(iden.ID),
		Name: iden.Name,
		Key:  iden.PrivKey,
	})
	if err != nil {
		return err
	}
	return nil
}
func (i IdentityRepo) GetByID(id entity.ID) (entity.Identity, error) {
	return entity.Identity{}, ErrNotImplemented
}
func (i IdentityRepo) GetAll(_ IOption) ([]entity.Identity, error) {
	id, err := i.store.GetIdentity()
	if err != nil {
		return nil, err
	}
	return []entity.Identity{{
		ID:      entity.ID(id.ID),
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
		ID:      entity.ID(id.ID),
		Name:    id.Name,
		PrivKey: id.Key,
	}, nil
}
