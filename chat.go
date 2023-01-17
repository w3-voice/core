package core

import (
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hood-chat/core/entity"
	"github.com/hood-chat/core/pb"
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
	pms      PMService
	Identity IdentityAPI
}

func NewChatAPI(store *st.Store, b ContactBookAPI, p PMService, i IdentityAPI) ChatAPI {
	ch := rp.NewChatRepo(store)
	m := rp.NewMessageRepo(store)
	return &Chat{ch, m, b, p, i}
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

func (c *Chat) New(opt ChatOpt) (entity.ChatInfo, error) {
	if opt.Type == entity.Private {
		con, err := c.book.Get(entity.ID(opt.Members[0]))
		if err != nil {
			return entity.ChatInfo{}, err
		}
		me, err := c.Identity.Get()
		if err != nil {
			return entity.ChatInfo{}, err
		}
		chatID := generatePMChatID(con, *me.ToContact())
		if err != nil {
			return entity.ChatInfo{}, err
		}
		chat := entity.ChatInfo{ID: chatID, Name: con.Name, Members: []entity.Contact{*me.ToContact(), con}, Type: opt.Type}
		err = c.chRepo.Add(chat)
		return chat, err
	}
	return entity.ChatInfo{}, errors.New("type not supported")
}

func (c *Chat) Messages(chatID entity.ID, skip int, limit int) ([]entity.Message, error) {
	opt := rp.NewOption(skip, limit)
	opt.AddFilter("chatID", string(chatID))
	return c.mRepo.GetAll(opt)
}

func (c *Chat) Message(ID entity.ID) (entity.Message, error) {
	return c.mRepo.GetByID(ID)
}

func (c *Chat) Find(opt ChatOpt) ([]entity.ChatInfo, error) {
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
		chatID := generatePMChatID(con, *me.ToContact())
		if err != nil {
			return nil, err
		}
		chat, err := c.chRepo.GetByID(chatID)
		return []entity.ChatInfo{chat}, err
	}
	return nil, errors.New("type not supported")
}

func (c *Chat) Send(chatID entity.ID, content string) (*entity.Message, error) {
	me, err := c.Identity.Get()
	if err != nil {
		return nil, err
	}
	msg := entity.Message{
		ID:        entity.ID(uuid.New().String()),
		ChatID:    chatID,
		CreatedAt: time.Now().UTC().Unix(),
		Text:      content,
		Status:    entity.Pending,
		Author:    *me.ToContact(),
	}
	rmsg := c.mRepo
	err = rmsg.Add(msg)
	if err != nil {
		log.Errorf("Can not add message %s", err.Error())
		return nil, err
	}
	rchat := c.chRepo
	chat, err := rchat.GetByID(chatID)
	if err != nil {
		log.Errorf("Can not get chat %s", err.Error())
		return nil, err
	}
	for _, to := range chat.Members {
		if to.ID != msg.Author.ID {
			log.Debugf("outbox message")
			c.pms.Send(entity.Envelop{To: to, Message: msg})
			log.Debugf("outboxed message")
		}
	}
	return &msg, nil
}

func (c Chat) received(msg *pb.Message) error {
	mAuthorID := entity.ID(msg.Author.Id)
	msgID := entity.ID(msg.GetId())
	chatID := entity.ID(msg.ChatId)

	con, err := c.book.Get(mAuthorID)
	if err != nil {
		log.Errorf("fail to get contact %s", err.Error())
		con = entity.Contact{
			ID:   mAuthorID,
			Name: msg.Author.Name,
		}
		err := c.book.Put(con)
		if err != nil {
			log.Errorf("fail to add contact %s", err.Error())
			return err
		}
	}

	chat, err := c.ChatInfo(chatID)
	if err != nil {
		log.Errorf("can not find chat %s", err.Error())
		opt := ChatOpt{
			msg.Author.Name,
			[]entity.ID{con.ID},
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
	opt := rp.NewOption(0, 10000)
	opt.AddFilter("chatID", string(chatID))
	opt.AddFilter("status", []entity.Status{entity.Received})
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

func generatePMChatID(con entity.Contact, me entity.Contact) entity.ID {
	cons := []string{con.ID.String(), me.ID.String()}
	sort.Strings(cons)
	return entity.ID(strings.Join(cons, ""))
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