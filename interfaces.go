package core

import "github.com/hood-chat/core/entity"

// provide api for managing contacts
type ContactBookAPI interface {
	// return list of contact 
	ContactList(skip int, limit int) ([]entity.Contact, error)
	// return a contact by id
	GetContact(id entity.ID) (entity.Contact, error)
	// create or update contact
	PutContact(c entity.Contact) error 
}

// provide api for managing identity
type IdentityAPI interface {
	IsLogin() bool
	SignUp(name string) (*entity.Identity, error)
	Get() (entity.Identity, error)
}

// provide api to use chat
type ChatAPI interface {
	Find(a interface{}) (entity.ChatInfo, error)
	New(a interface{}) (entity.ChatInfo, error)
	Send(chatID entity.ID, content string) (*entity.Message, error)
	Seen(chatID entity.ID) error
}

type MessengerAPI interface {
	ContactBook() ContactBookAPI
	ChatAPI()     ChatAPI
	IdentityAPI() IdentityAPI
}