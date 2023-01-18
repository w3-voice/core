package core

import (
	"errors"

	"github.com/hood-chat/core/entity"
	rp "github.com/hood-chat/core/repo"
	st "github.com/hood-chat/core/store"
)

var _ IdentityAPI = (*Identity)(nil)

type IdentityRepo = rp.IRepo[entity.Identity]

type Identity struct {
	repo      rp.IRepo[entity.Identity]
	identity  *entity.Identity
}

func NewIdentityAPI(store *st.Store) IdentityAPI {
	repo := rp.NewIdentityRepo(store)
	identity, err := repo.Get()
	if err != nil {
		return &Identity{repo, nil}
	}
	return &Identity{repo, &identity}
}

func (i *Identity) IsLogin() bool {
	return i.identity != nil
}

func (i *Identity) SignUp(name string) (*entity.Identity, error) {
	rIdentity := i.repo
	iden, err := entity.CreateIdentity(name)
	if err != nil {
		return nil, err
	}
	err = rIdentity.Put(iden)
	i.identity = &iden
	if err != nil {
		return nil, err
	}
	return &iden, nil
}

func (i *Identity) Get() (entity.Identity, error) {
	if i.identity==nil {
		return entity.Identity{}, errors.New("Not Login")
	}
	return *i.identity, nil
}
