package repo

import (
	"github.com/bee-messenger/core/entity"
	"github.com/bee-messenger/core/store"
)

type ContactRepo struct {
	store store.Store
}

func NewContactRepo(path string) (*ContactRepo, error) {
	s, err := store.NewStore(path)
	if err != nil {
		return nil, err
	}
	return &ContactRepo{
		store: *s,
	}, nil
}

func (c ContactRepo) GetAll() ([]entity.Contact, error) {
	cons := make([]entity.Contact, 0)
	bhcl, err := c.store.AllContacts()
	if err != nil {
		return nil, err
	}
	for _, val := range bhcl {
		cons = append(cons, entity.Contact{
			Name: val.Name,
			ID:   val.ID,
		})
	}
	return cons, nil
}

func (c ContactRepo) GetByID(id string) (*entity.Contact, error) {
	con, err := c.store.ContactByID(id)
	if err != nil {
		return nil, err
	}
	return &entity.Contact{
		ID:   con.ID,
		Name: con.Name,
	}, nil
}

func (c ContactRepo) Add(con entity.Contact) error {
	err := c.store.InsertContact(store.BHContact{
		Name: con.Name,
		ID:   con.ID,
	})
	if err != nil {
		return err
	}
	return nil
}
