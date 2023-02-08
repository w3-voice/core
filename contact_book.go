package core

import (
	"github.com/hood-chat/core/entity"
	rp "github.com/hood-chat/core/repo"
	st "github.com/hood-chat/core/store"
)


var _ ContactBookAPI = (*ContactBook)(nil)

type ContactBook struct {
	repo  rp.IRepo[entity.Contact]
}

func NewContactBook(store *st.Store) ContactBookAPI {
	return &ContactBook{rp.NewContactRepo(store)}
}

func (c *ContactBook) List(skip int, limit int) (entity.ContactSlice, error) {
	rContact := c.repo
	opt := rp.NewOption(skip, limit)
	return rContact.GetAll(opt)
}

func (c *ContactBook) Get(id entity.ID) (entity.Contact, error) {
	rContact := c.repo
	return rContact.GetByID(id)
}

func (c *ContactBook) Put(con entity.Contact) error {
	rContact := c.repo
	return rContact.Put(con)
}