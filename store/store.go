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

type BHContact struct {
	ID      string
	Name    string
}

type BHChat struct {
	Name string
	ID string
	Members []string
}

type BHTextMessage struct {
	ID            string
	ChatID        string `badgerhold:"index"`
	CreatedAt     time.Time
	Text          string
	Status        Status
	Author        BHContact
}

type Store struct {
	bh badgerhold.Store
}

func NewStore(path string) (*Store, error) {
	opt := badgerhold.DefaultOptions
	opt.Dir = path
	opt.ValueDir = path
	store, err := badgerhold.Open(opt)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	return &Store {
		bh: *store,
	}, nil

}

func (s *Store) InsertContact(contact BHContact) error {
	err := s.bh.Insert(contact.ID, contact)
	return err
}

func (s *Store) InsertTextMessage(tm BHTextMessage) error {
	err := s.bh.Insert(tm.ID, tm)
	return err
}

func (s *Store) InsertChat(ch BHChat) error {
	err := s.bh.Insert(ch.ID, ch)
	return err
}

func (s *Store) ChatList() ([]BHChat, error){
	var res []BHChat
	q := &badgerhold.Query{}
	err :=  s.bh.Find(&res, q.Limit(50))
	return res, err
}

func (s *Store) ChatMessages(id string) ([]BHTextMessage, error) {
	var res []BHTextMessage
	q := &badgerhold.Query{}
	err :=  s.bh.Find(&res, q.Limit(50))
	return res, err
}

func (s *Store) AllContacts() ([]BHContact, error) {
	var res []BHContact
	q := &badgerhold.Query{}
	err :=  s.bh.Find(&res, q.Limit(50))
	return res, err
}

func (s *Store) ContactByIDs(ids []string) ([]BHContact, error) {
	var res []BHContact
	q :=  badgerhold.Where("ID").In(ids)
	err :=  s.bh.Find(&res, q.Limit(50))
	return res, err
}

func (s *Store) ContactByID(id string) (BHContact, error) {
	var res BHContact
	err := s.bh.FindOne(&res, badgerhold.Where("ID").Eq(id))
	return res, err
}

func (s *Store) ChatByID(id string) (BHChat, error) {
	var res BHChat
	err := s.bh.FindOne(&res, badgerhold.Where("ID").Eq(id))
	return res, err
}


