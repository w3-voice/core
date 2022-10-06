package store

import (
	"log"
	"time"

	"github.com/dgraph-io/badger/v3"
	"github.com/timshannon/badgerhold/v4"
)

type Status int

const (
    Pending Status = iota
    Sent
    Seen
)

type Contact struct {
	ID      int
	Name    string
}

type TextMessage struct {
	ID            int
	AuthorID      int `badgerhold:"index"`
	CreatedAt     time.Time
	Text          string
	Status        Status
}

type Repo struct {
	store badgerhold.Store
}


func NewRepo(path string) (*Repo, error) {
	opt := badgerhold.DefaultOptions
	opt.Dir = path
	opt.ValueDir = path
	store, err := badgerhold.Open(opt)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	return &Repo {
		store: *store,
	}, nil

}

func (r *Repo) AddContact() {

}

func (r *Repo) GetContact() {

}

func (r *Repo) AddTextMessage() {

}

func (r *Repo) GetTextMessage() {

}

