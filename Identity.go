package core

import (
	"errors"

	"github.com/hood-chat/core/entity"
	rp "github.com/hood-chat/core/repo"
	st "github.com/hood-chat/core/store"
	"github.com/libp2p/go-libp2p/core/peer"
)

var _ IdentityAPI = (*identityAPI)(nil)

type IdentityRepo = rp.IRepo[entity.Identity]

type identityAPI struct {
	repo      rp.IRepo[entity.Identity]
	identity  *entity.Identity
}

func NewIdentityAPI(store *st.Store) IdentityAPI {
	repo := rp.NewIdentityRepo(store)
	identity, err := repo.Get()
	if err != nil {
		return &identityAPI{repo, nil}
	}
	return &identityAPI{repo, &identity}
}

func (i *identityAPI) IsLogin() bool {
	return i.identity != nil
}

func (i *identityAPI) SignUp(name string) (*entity.Identity, error) {
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

func (i *identityAPI) Get() (entity.Identity, error) {
	if i.identity==nil {
		return entity.Identity{}, errors.New("Not Login")
	}
	return *i.identity, nil
}

func (i *identityAPI) PeerID() (peer.ID, error) {
	if i.identity==nil {
		return peer.ID(""), errors.New("Not Login")
	}
	return i.identity.PeerID()
}