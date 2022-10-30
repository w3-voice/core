package core

import (
	"time"

	"github.com/bee-messenger/core/entity"
	"github.com/bee-messenger/core/pb"
	"github.com/bee-messenger/core/repo"
	"github.com/bee-messenger/core/store"
	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p-core/host"
)

type Handler func(msg Envelop) error
type Messenger struct {
	repo     repo.Repo
	identity entity.Contact
	pms      PMService
}

type Envelop struct {
	msg    entity.Message
	chatID string
	To     string
}

func MessengerBuilder(path string, h host.Host) Messenger {
	s, err := store.NewStore(path + "/store")
	if err != nil {
		panic(err)
	}

	user := entity.Contact{
		ID:   h.ID().String(),
		Name: "user",
	}

	chrepo, err := repo.NewChatRepo(*s)
	if err != nil {
		panic(err)
	}

	m := Messenger{
		repo:     *chrepo,
		identity: user,
	}
	pms := NewPMService(h, m.MessageHandler)
	m.pms = *pms

	return m
}

func (m *Messenger) GetChat(id string) (*Chat, error) {
	chat := Chat{
		me:   &m.identity,
		repo: &m.repo,
		c:    nil,
	}
	return chat.ByID(id)
}

func (m *Messenger) CreateChat(id string, members []entity.Contact, name string) *Chat {
	cht := entity.ChatInfo{
		ID:      id,
		Name:    name,
		Members: members,
	}

	err := m.repo.CreateChat(cht)
	if err != nil {
		panic("its failed")
	}
	return &Chat{
		me: &m.identity,
		repo: &m.repo,
		c: &entity.Chat{
			Info:     cht,
			Messages: []entity.Message{},
		},
	}
}

func (c *Messenger) SendMessage(env Envelop) {
	c.pms.Send(&env)
}

func (c *Messenger) MessageHandler(msg *pb.Message) {
	con := entity.Contact{
		ID:   msg.Author.Id,
		Name: msg.Author.Name,
	}
	ch := Chat{
		repo: &c.repo,
		c:    nil,
	}
	chat, err := ch.ByID(msg.GetChatId())
	if err != nil {
		chat = c.CreateChat(msg.GetChatId(), []entity.Contact{c.identity, con}, msg.Author.Id)
	}

	newMsg := entity.Message{
		ID:        msg.Id,
		CreatedAt: time.Now(),
		Text:      msg.GetText(),
		Status:    entity.Sent,
		Author:    con,
	}

	err = chat.AddMessage(newMsg)
	if err != nil {
		panic(err)
	}
}

type Chat struct {
	repo *repo.Repo
	c    *entity.Chat
	me   *entity.Contact
}

func (c *Chat) NewMessage(content string) (*Envelop, error) {
	msg := &entity.Message{
		ID:        uuid.New().String(),
		CreatedAt: time.Now(),
		Text:      content,
		Status:    entity.Pending,
		Author:    *c.me,
	}
	err := c.repo.AddMessage(c.c.ID(), *msg)
	to := []string{}
	for _, val := range c.c.Info.Members {
		if val.ID != msg.Author.ID {
			to = append(to, val.ID)
		}
	}

	return &Envelop{msg: *msg, To: to[0], chatID: c.c.ID()}, err
}

func (c *Chat) ID() string {
	return c.c.ID()
}

func (c *Chat) AddMessage(msg entity.Message) error {
	return c.repo.AddMessage(c.c.ID(), msg)
}

func (c *Chat) GetMessages() ([]entity.Message, error) {
	chat, err := c.repo.GetByIDChat(c.c.ID())
	return chat.Messages, err
}

func (c Chat) ByID(chatID string) (*Chat, error) {
	chat, err := c.repo.GetByIDChat(chatID)
	if err != nil {
		return nil, err
		// newChat := &entity.ChatInfo{
		// 	ID:      msg.GetChatId(),
		// 	Name:    con.Name,
		// 	Members: []entity.Contact{*con, c.identity},
		// }
		// err := c.repo.CreateChat(*newChat)
		// if err != nil {
		// 	return err
		// }
		// chat = &entity.Chat{
		// 	Info:     *newChat,
		// 	Messages: nil,
		// }
	}
	return &Chat{
		repo: c.repo,
		c:    chat,
	}, nil
}
