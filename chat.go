package core

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/hood-chat/core/entity"
	"github.com/hood-chat/core/pb"
	rp "github.com/hood-chat/core/repo"
	st "github.com/hood-chat/core/store"
	"github.com/timshannon/badgerhold/v4"
)

var _ ChatAPI = (*Chat)(nil)

type ChatRepo = rp.IRepo[entity.ChatInfo]
type MessageRepo = rp.IRepo[entity.Message]

type Chat struct {
	chRepo   ChatRepo
	mRepo    MessageRepo
	book     ContactBookAPI
	pms      MessengerService
	gps      GroupChatService
	Identity IdentityAPI
}

func NewChatAPI(store *st.Store, b ContactBookAPI, p MessengerService, g GroupChatService,i IdentityAPI) ChatAPI {
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

func (c *Chat) ChatInfos(skip int, limit int) ([]entity.ChatInfo, error) {
	rChat := c.chRepo
	opt := rp.NewOption(skip, limit)
	return rChat.GetAll(opt)
}

func (c *Chat) New(opt NewChatOpt) (entity.ChatInfo, error) {

	me, err := c.Identity.Get()
	if err != nil {
		return entity.ChatInfo{}, err
	}

	switch opt.Type {
	case entity.Private:
		chat := entity.NewPrivateChat(opt.Members[0], *me.ToContact())
		err = c.chRepo.Add(chat)
		return chat, err
	case entity.Group:
		members := append(opt.Members, *me.ToContact())
		chat := entity.NewGroupChat(opt.Name, members)
		err := c.chRepo.Add(chat)
		c.gps.Join(chat.ID, members)
		return chat, err
	default:
		return entity.ChatInfo{}, errors.New("type not supported")
	}

}

func (c *Chat) Messages(chatID entity.ID, skip int, limit int) ([]entity.Message, error) {
	opt := rp.NewOption(skip, limit)
	opt.AddFilter("ChatID", string(chatID))
	return c.mRepo.GetAll(opt)
}

func (c *Chat) Message(ID entity.ID) (entity.Message, error) {
	return c.mRepo.GetByID(ID)
}

func (c *Chat) Find(opt SearchChatOpt) ([]entity.ChatInfo, error) {
	if opt.Type == entity.Private {
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
				c.pms.Send(entity.Envelop{To: to, Message: msg})
				log.Debugf("outboxed message")
			}
		}
		return &msg, nil
	case entity.Group:
		c.gps.Send(entity.Envelop{To: entity.Contact{
			Name: chat.Name,
			ID: chatID,
		}, Message: msg})
		return &msg, nil
	default:
		return nil, errors.New("Chat type not supported")

	}

}

func (c Chat) received(msg *pb.Message) error {
	mAuthorID := entity.ID(msg.Author.Id)
	msgID := entity.ID(msg.GetId())
	chatID := entity.ID(msg.ChatId)

	con, err := c.book.Get(mAuthorID)
	if err == badgerhold.ErrNotFound {
		log.Errorf("fail to get contact %s", err.Error())
		con = entity.Contact{
			ID:   mAuthorID,
			Name: msg.Author.Name,
		}
		err = nil
	}
	if err != nil {
		return err
	}

	chat, err := c.ChatInfo(chatID)
	if err != nil {
		log.Errorf("can not find chat %s", err.Error())
		opt := NewChatOpt{
			msg.Author.Name,
			[]entity.Contact{con},
			entity.Private,
		}
		chat, err = c.New(opt)
		if err != nil {
			log.Errorf("fail to handle new message %s", err.Error())
			return err
		}
	}

	newMsg := entity.Message{
		ID:        msgID,
		ChatID:    chat.ID,
		CreatedAt: msg.GetCreatedAt(),
		Text:      msg.GetText(),
		Status:    entity.Received,
		Author:    con,
		ChatType:  entity.ChatType(msg.ChatType),
	}

	rmsg := c.mRepo
	err = rmsg.Add(newMsg)
	log.Debugf("new message %s ", newMsg)
	if err != nil {
		log.Errorf("Can not add message %s , %d", err.Error(), newMsg)
		return err
	}
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

func (c *Chat) Join(id entity.ID, name string, chatType entity.ChatType, members ...entity.Contact) error {
	chat := &entity.ChatInfo{id, name,members, chatType,0,""}
	err := c.chRepo.Add(*chat)
	if err != nil {
		return err
	}
	c.gps.Join(id,members)
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
