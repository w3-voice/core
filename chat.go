package core

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/hood-chat/core/entity"
	"github.com/hood-chat/core/protocol/invite"
	rp "github.com/hood-chat/core/repo"
	st "github.com/hood-chat/core/store"
)

var _ ChatAPI = (*Chat)(nil)

type ChatRepo = rp.IRepo[entity.ChatInfo]
type MessageRepo = rp.IRepo[entity.Message]

type Chat struct {
	chRepo   ChatRepo
	mRepo    MessageRepo
	book     ContactBookAPI
	pms      DirectService
	gps      PubSubService
	Identity IdentityAPI
}

func NewChatAPI(store *st.Store, b ContactBookAPI, p DirectService, g PubSubService,i IdentityAPI) ChatAPI {
	ch := rp.NewChatRepo(store)
	m := rp.NewMessageRepo(store)
	return &Chat{ch, m, b, p, g,i}
}

func (c *Chat) ChatInfo(id entity.ID) (entity.ChatInfo, error) {
	rChat := c.chRepo
	ci := entity.ChatInfo{}
	chat, err := rChat.GetByID(id)
	if err != nil {
		return ci, err
	}
	return chat, nil
}

func (c *Chat) ChatInfos(skip int, limit int) (entity.ChatSlice, error) {
	rChat := c.chRepo
	opt := rp.NewOption(skip, limit)
	return rChat.GetAll(opt)
}

func (c *Chat) New(opt NewChatOpt) (entity.ChatInfo, error) {

	me, err := c.Identity.Get()
	if err != nil {
		return entity.ChatInfo{}, err
	}

	switch *opt.Type {
	case entity.Private:
		chat := entity.NewPrivateChat(opt.Members[0], *me.ToContact())
		err = c.chRepo.Add(chat)
		return chat, err
	case entity.Group:
		members := append(opt.Members, *me.ToContact())
		chat := entity.NewGroupChat(*opt.Name, members, []entity.Contact{*me.ToContact()})
		err := c.chRepo.Add(chat)
		c.gps.Join(chat.ID, members)
		return chat, err
	default:
		return entity.ChatInfo{}, errors.New("type not supported")
	}

}

func (c *Chat) Messages(chatID entity.ID, skip int, limit int) (entity.MessageSlice, error) {
	opt := rp.NewOption(skip, limit)
	opt.AddFilter("ChatID", string(chatID))
	return c.mRepo.GetAll(opt)
}

func (c *Chat) Message(ID entity.ID) (entity.Message, error) {
	return c.mRepo.GetByID(ID)
}

func (c *Chat) Find(opt SearchChatOpt) (entity.ChatSlice, error) {
	if *opt.Type == entity.Private {
		contactID := opt.Members[0]
		con, err := c.book.Get(contactID)
		if err != nil {
			return nil, err
		}
		me, err := c.Identity.Get()
		if err != nil {
			return nil, err
		}
		chatID := entity.NewPrivateChat(con, *me.ToContact())
		if err != nil {
			return nil, err
		}
		chat, err := c.chRepo.GetByID(chatID.ID)
		return []entity.ChatInfo{chat}, err
	}
	return nil, errors.New("type not supported")
}

func (c *Chat) Send(chatID entity.ID, content string) (*entity.Message, error) {
	me, err := c.Identity.Get()
	if err != nil {
		return nil, err
	}

	rchat := c.chRepo
	chat, err := rchat.GetByID(chatID)
	if err != nil {
		log.Errorf("Can not get chat %s", err.Error())
		return nil, err
	}

	msg := entity.Message{
		ID:        entity.ID(uuid.New().String()),
		ChatID:    chatID,
		CreatedAt: time.Now().UTC().Unix(),
		Text:      content,
		Status:    entity.Pending,
		Author:    *me.ToContact(),
		ChatType:  chat.Type,
	}
	rmsg := c.mRepo
	err = rmsg.Add(msg)
	if err != nil {
		log.Errorf("Can not add message %s", err.Error())
		return nil, err
	}

	switch chat.Type {
	case entity.Private:
		for _, to := range chat.Members {
			if to.ID != msg.Author.ID {
				log.Debugf("outbox message")
			    n,_ := NewMessageEnvelop(to, msg)
				c.pms.Send(n)
				log.Debugf("outboxed message")
			}
		}
		return &msg, nil
	case entity.Group:
		c.gps.Send(PubSubEnvelop{Topic: chatID.String(), Message: msg})
		return &msg, nil
	default:
		return nil, errors.New("Chat type not supported")

	}

}

func (c Chat) received(msg entity.Message) error {
	_, err := c.ChatInfo(msg.ChatID)
	if err != nil {
		log.Errorf("can not find chat %s", err.Error())
		opt := NewChatOpt{
			&msg.Author.Name,
			[]entity.Contact{msg.Author},
			Of(entity.Private),
		}
		_, err = c.New(opt)
		if err != nil {
			log.Errorf("fail to handle new message %s", err.Error())
			return err
		}
	}

	rmsg := c.mRepo
	err = rmsg.Add(msg)
	if err != nil {
		log.Errorf("Can not add message %s , %d", err.Error(), msg)
		return err
	}
	log.Debugf("new message %s ", msg)
	return nil
}

func (c *Chat) Seen(chatID entity.ID) error {
	opt := rp.NewOption(0, 0)
	opt.AddFilter("ChatID", string(chatID))
	opt.AddFilter("Status", []entity.Status{entity.Received})
	unread, err := c.mRepo.GetAll(opt)
	if err != nil {
		return err
	}
	for _, msg := range unread {
		msg.Status = entity.Seen
		err = c.mRepo.Put(msg)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Chat) Join(ci entity.ChatInfo) error {
	err := c.chRepo.Add(ci)
	if err != nil {
		return err
	}
	c.gps.Join(ci.ID, ci.Admins)
	return nil
}

func (c *Chat) Invite(chatID entity.ID, cons entity.ContactSlice) error {
	chat, err := c.chRepo.GetByID(chatID)
	if err != nil {
		return err
	}
	for _, con := range cons {
		c.pms.Send(&Envelop{To: con,Message: chat,ID: chat.ID.String(),CreatedAt: time.Now().Unix(),Protocol: invite.ID})
	}
	return nil
}

func (c *Chat) updateMessageStatus(msgID entity.ID, status entity.Status) error {
	rmsg := c.mRepo
	msg, err := rmsg.GetByID(msgID)
	if err != nil {
		return err
	}
	msg.Status = status
	err = rmsg.Put(msg)
	if err != nil {
		return err
	}
	return nil
}
