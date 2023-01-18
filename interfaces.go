package core

import (
	"github.com/hood-chat/core/entity"
	"github.com/hood-chat/core/pb"
	lpevt "github.com/libp2p/go-libp2p/core/event"
)

type Bus = lpevt.Bus

// provide api for managing contacts
type ContactBookAPI interface {
	// return list of contact
	List(skip int, limit int) ([]entity.Contact, error)
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
}

// provide api to use chat
type ChatAPI interface {
	ChatInfo(id entity.ID) (entity.ChatInfo, error)
	ChatInfos(skip int, limit int) ([]entity.ChatInfo, error)
	Find(opt ChatOpt) ([]entity.ChatInfo, error)
	New(opt ChatOpt) (entity.ChatInfo, error)
	Send(chatID entity.ID, content string) (*entity.Message, error)
	Seen(chatID entity.ID) error
	Message(ID entity.ID) (entity.Message, error)
	Messages(chatID entity.ID, skip int, limit int) ([]entity.Message, error)
	updateMessageStatus(msgID entity.ID, status entity.Status) error
	received(msg *pb.Message) error
}

type MessengerAPI interface {
	ContactBookAPI()  ContactBookAPI
	ChatAPI()      ChatAPI
	IdentityAPI()  IdentityAPI
	EventBus()     Bus
	Start()     
	Stop()
}

type ChatOpt struct {
	Name    string
	Members []entity.ID
	Type    entity.ChatType
}

func ForPrivateChat(contactID entity.ID) ChatOpt {
	return ChatOpt{"", []entity.ID{contactID}, entity.Private}
}
