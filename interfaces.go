package core

import (
	"github.com/hood-chat/core/entity"
	"github.com/libp2p/go-libp2p/core/peer"
)

// provide api for managing contacts
type ContactBookAPI interface {
	// return list of contact
	List(skip int, limit int) (entity.ContactSlice, error)
	// return a contact by id
	Get(id entity.ID) (entity.Contact, error)
	// create or update contact
	Put(c entity.Contact) error
}

// provide api for managing identity
type IdentityAPI interface {
	IsLogin() bool
	SignUp(name string) (*entity.Identity, error)
	Get() (entity.Identity, error)
	PeerID() (peer.ID, error)
}

// provide api to use chat
type ChatAPI interface {
	ChatInfo(id entity.ID) (entity.ChatInfo, error)
	ChatInfos(skip int, limit int) (entity.ChatSlice, error)
	Join(entity.ChatInfo) error
	Find(opt SearchChatOpt) (entity.ChatSlice, error)
	New(opt NewChatOpt) (entity.ChatInfo, error)
	Send(chatID entity.ID, content string) (*entity.Message, error)
	Seen(chatID entity.ID) error
	Message(ID entity.ID) (entity.Message, error)
	Messages(chatID entity.ID, skip int, limit int) (entity.MessageSlice, error)
	Invite(chID entity.ID, cons entity.ContactSlice) error
	updateMessageStatus(msgID entity.ID, status entity.Status) error
	received(msg entity.Message) error
}

type MessengerAPI interface {
	ContactBookAPI()  ContactBookAPI
	ChatAPI()         ChatAPI
	IdentityAPI()     IdentityAPI
	EventBus()        Bus
	Start()     
	Stop()
}




type DirectService interface {
	Send(nvlop *Envelop)
	Stop()
}

type PubSubService interface {
	Send(PubSubEnvelop)
	Stop()
	Join(chatId entity.ID, members []entity.Contact)
}

