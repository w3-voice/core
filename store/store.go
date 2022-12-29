package store

import (
	"github.com/timshannon/badgerhold/v4"

	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("msgr-core-store")

type Status int

const (
	Pending Status = iota
	Sent
	Seen
	Received
	Failed
)

type BHIdentity struct {
	ID   string `badgerhold:"unique"`
	Name string
	Key  string
}

type BHContact struct {
	ID   string `badgerhold:"unique"`
	Name string
}

type BHChat struct {
	Name    string
	ID      string `badgerhold:"unique"`
	Members []string
}

type BHTextMessage struct {
	// BID       int64 `badgerhold:"key"`
	ID        string `badgerhold:"unique"`
	ChatID    string `badgerhold:"index"`
	CreatedAt int64
	Text      string
	Status    Status
	Author    BHContact
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

	return &Store{
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

func (s *Store) ChatList(skip int, limit int) ([]BHChat, error) {
	var res []BHChat
	q := &badgerhold.Query{}
	q.Limit(limit)
	q.Skip(skip)
	err := s.bh.Find(&res, q)
	return res, err
}

func (s *Store) ChatMessages(id string, skip int, limit int) ([]BHTextMessage, error) {
	var res []BHTextMessage
	q := badgerhold.Where("ChatID").Eq(id).SortBy("CreatedAt").Reverse()
	q.Limit(limit)
	q.Skip(skip)
	err := s.bh.Find(&res, q)
	return res, err
}

func (s *Store) MsgByID(id string) (BHTextMessage, error) {
	var res BHTextMessage
	q := badgerhold.Where("ID").Eq(id)
	err := s.bh.FindOne(&res, q)
	return res, err
}

func (s *Store) UpdateMessage(msg BHTextMessage) error {
	// tx := s.bh.Badger().NewTransaction(true)
	return s.bh.Update(msg.ID, msg)
}

func (s *Store) AllContacts(skip int, limit int) ([]BHContact, error) {
	var res []BHContact
	q := &badgerhold.Query{}
	q.Limit(limit)
	q.Skip(skip)
	err := s.bh.Find(&res, q)
	return res, err
}

func (s *Store) ContactByIDs(ids []string) ([]BHContact, error) {
	var res []BHContact
	q := badgerhold.Where("ID").In(badgerhold.Slice(ids)...)
	err := s.bh.Find(&res, q.Limit(50))
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

func (s *Store) SetIdentity(id BHIdentity) error {
	err := s.bh.Insert(id.ID, id)
	return err
}

func (s *Store) GetIdentity() (BHIdentity, error) {
	var res BHIdentity
	q := &badgerhold.Query{}
	err := s.bh.FindOne(&res, q)
	if err != nil {
		return BHIdentity{}, err
	}
	return res, err
}

func (s *Store) Close() {
	s.bh.Close()
}