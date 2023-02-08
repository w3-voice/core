package entity

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sort"
	"strings"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

type Status int
type ChatType int
type ID string

func (id ID) String() string {
	return string(id)
}

const (
	Pending Status = iota
	Sent
	Seen
	Received
	Failed
)

const (
	Private ChatType = iota
	Group
)

type Identity struct {
	ID      ID			`json:"_id"`
	Name    string		`json:"name"`
	PrivKey string		`json:"-"`
}

func (c Identity) PeerID() (peer.ID, error) {
	return peer.Decode(string(c.ID))
}

// DecodePrivateKey is a helper to decode the users PrivateKey
func (i *Identity) DecodePrivateKey(passphrase string) (crypto.PrivKey, error) {
	pkb, err := base64.StdEncoding.DecodeString(i.PrivKey)
	if err != nil {
		return nil, err
	}

	// currently storing key unencrypted. in the future we need to encrypt it.
	// TODO(security)
	return crypto.UnmarshalPrivateKey(pkb)
}

func (i *Identity) ToContact() *Contact {
	return &Contact{
		ID:   i.ID,
		Name: i.Name,
	}
}

func CreateIdentity(name string) (Identity, error) {
	ident := Identity{}

	var sk crypto.PrivKey
	var pk crypto.PubKey

	fmt.Printf("generating ED25519 keypair...")
	priv, pub, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return ident, err
	}

	sk = priv
	pk = pub

	fmt.Print("done\n")

	// currently storing key unencrypted. in the future we need to encrypt it.
	// TODO(security)
	skbytes, err := crypto.MarshalPrivateKey(sk)
	if err != nil {
		return ident, err
	}
	ident.PrivKey = base64.StdEncoding.EncodeToString(skbytes)

	id, err := peer.IDFromPublicKey(pk)
	if err != nil {
		return ident, err
	}
	ident.ID = ID(id.String())
	ident.Name = name
	fmt.Printf("peer identity: %s\n", ident.ID)
	return ident, nil
}

var _ JsonMessage = (*Message)(nil)

type Message struct {
	ID        ID			`json:"_id"`
	ChatID    ID			`json:"chatId"`
	CreatedAt int64			`json:"createdAt"`
	Text      string		`json:"text"`
	Status    Status		`json:"status"`
	Author    Contact		`json:"user"`
	ChatType  ChatType		`json:"chatType"`
}

type Contact struct {
	ID   ID			`json:"_id"`
	Name string		`json:"name"`
}

func (c Contact) AdderInfo() (*peer.AddrInfo, error) {
	p, err := c.PeerID()
	if err != nil {
		return nil, err
	}
	return peer.AddrInfoFromString("/p2p/" + p.String())
}

func (c Contact) PeerID() (peer.ID, error) {
	return peer.Decode(string(c.ID))
}

var _ JsonMessage = (*ChatInfo)(nil)

type ChatInfo struct {
	ID         ID			`json:"_id"`
	Name       string		`json:"name"`
	Members    []Contact	`json:"members"`
	Admins     []Contact	`json:"admins"`
	Type       ChatType		`json:"type"`
	Unread     uint64		`json:"unread"`
	LatestText string		`json:"latestText"`
}

func NewPrivateChat(creator Contact, con Contact) ChatInfo {
	chatID := generatePMChatID(con, creator)
	return ChatInfo{ID: chatID, Name: con.Name, Members: []Contact{creator, con}, Type: Private}
}

func generatePMChatID(creator Contact, con Contact) ID {
	cons := []string{con.ID.String(), creator.ID.String()}
	sort.Strings(cons)
	return ID(strings.Join(cons, ""))
}

func NewGroupChat(name string, members []Contact, admins []Contact) ChatInfo {
	var pk crypto.PubKey

	fmt.Printf("generating ED25519 keypair...")
	_, pub, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		panic("can not generate key")
	}
	pk = pub
	id, err := peer.IDFromPublicKey(pk)
	if err != nil {
		panic("can not generate key")
	}
	return ChatInfo{ID: ID(id.String()), Name: name, Members: members, Type: Group, Admins: admins}
}
