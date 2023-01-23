package core

import (
	"github.com/hood-chat/core/entity"
	"github.com/hood-chat/core/pb"
	"github.com/libp2p/go-libp2p/core/peer"
)

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
	PeerID() (peer.ID, error)
}

// provide api to use chat
type ChatAPI interface {
	ChatInfo(id entity.ID) (entity.ChatInfo, error)
	ChatInfos(skip int, limit int) ([]entity.ChatInfo, error)
	Join(id entity.ID, name string, chatType entity.ChatType, members ...entity.Contact) error
	Find(opt SearchChatOpt) ([]entity.ChatInfo, error)
	New(opt NewChatOpt) (entity.ChatInfo, error)
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


type SearchChatOpt struct {
	Name    string
	Members []entity.ID
	Type    entity.ChatType
}

func WithPrivateChatContact(contactID entity.ID) SearchChatOpt {
	return SearchChatOpt{"", []entity.ID{contactID}, entity.Private}
}


type NewChatOpt struct {
	Name    string
	Members []entity.Contact
	Type    entity.ChatType
}

func NewPrivateChat(contact entity.Contact) NewChatOpt {
	return NewChatOpt{"", []entity.Contact{contact}, entity.Private}
}

type MessengerService interface {
	Send(entity.Envelop)
	Stop()
}

type GroupChatService interface {
	MessengerService
	Join(chatId entity.ID, members []entity.Contact)
}
