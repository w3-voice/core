package repo

import (
	"github.com/bee-messenger/core/store"
	"github.com/bee-messenger/core/entity"
)




type ChatRepo struct {
	store store.Store
}

func NewRepo(path string) (*ChatRepo, error) {
	s, err := store.NewStore(path)
	if err != nil {
		return nil, err
	}
	return &ChatRepo{
		store: *s,
	}, nil
}

func (c ChatRepo) GetChatList() ([]entity.ChatInfo, error) {
	chl, err := c.store.ChatList()
	ci := make([]entity.ChatInfo,0)
	if err != nil {
		return nil, err
	}
	for _, val := range chl {
		members := make([]entity.Contact, 0)
		m, _ := c.store.ContactByIDs(val.Members)
		for _, me := range m {
			members = append(members, entity.Contact{
				ID: me.ID,
				Name: me.Name,
			})
		}
		ci = append(ci, entity.ChatInfo{
			ID: val.ID,
			Name: val.Name,
			Members: members,
		})
	}
	return ci, nil
}

func (c ChatRepo) GetChat(id string) (*entity.Chat, error) {
	ct, err := c.store.ChatByID(id)
	if err != nil {
		return nil, err
	}
	members := make([]entity.Contact, 0)
	m, _ := c.store.ContactByIDs(ct.Members)
	for _, me := range m {
		members = append(members, entity.Contact{
			ID: me.ID,
			Name: me.Name,
		})
	}
	messages := make([]entity.Message, 0)
	bhm, err := c.store.ChatMessages(id)
	if err != nil {
		return nil, err
	}
	for _, m := range bhm {
		messages = append(messages, entity.Message{
			ID: m.ID,
			CreatedAt: m.CreatedAt,
			Text: m.Text,
			Status: entity.Status(m.Status),
			Author: entity.Contact{
				ID: m.Author.ID,
				Name: m.Author.Name,
			},
		})
	}
	return &entity.Chat{
		Info: entity.ChatInfo{
			ID: ct.ID,
			Name: ct.Name,
			Members: members,
		},
		Messages: messages,
	}, nil
}

func (c ChatRepo) AddChatMessage(chatId string, msg entity.Message) error {
	m := store.BHTextMessage{
		ID: msg.ID,
		ChatID: chatId,
		CreatedAt: msg.CreatedAt,
		Text: msg.Text,
		Status: store.Status(msg.Status),
		Author: store.BHContact(msg.Author),
	}
	err := c.store.InsertTextMessage(m)
	if err != nil {
		return err
	}
	return nil
}
