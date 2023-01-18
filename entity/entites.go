package entity

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/libp2p/go-libp2p/core/crypto"
	ic "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/hood-chat/core/pb"
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
)

type Identity struct {
	ID      ID
	Name    string
	PrivKey string
}

// DecodePrivateKey is a helper to decode the users PrivateKey
func (i *Identity) DecodePrivateKey(passphrase string) (ic.PrivKey, error) {
	pkb, err := base64.StdEncoding.DecodeString(i.PrivKey)
	if err != nil {
		return nil, err
	}

	// currently storing key unencrypted. in the future we need to encrypt it.
	// TODO(security)
	return ic.UnmarshalPrivateKey(pkb)
}

func (i *Identity) ToContact() *Contact {
	return &Contact{
		ID: i.ID,
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

type Message struct {
	ID        ID
	ChatID    ID
	CreatedAt int64
	Text      string
	Status    Status
	Author    Contact
}

type Contact struct {
	ID   ID
	Name string
}

func (c Contact) AdderInfo() (*peer.AddrInfo, error){
	p, err := c.PeerID()
	if err != nil {
		return nil, err
	}
	return peer.AddrInfoFromString("/p2p/" + p.String())
}

func (c Contact) PeerID() (peer.ID, error) {
	return peer.Decode(string(c.ID))
}

type Envelop struct {
	To Contact
	Message Message
}

func (n Envelop) Proto() *pb.Message {
	msg := n.Message
	return &pb.Message{
		Text:      msg.Text,
		Id:        msg.ID.String(),
		ChatId:    msg.ChatID.String(),
		CreatedAt: msg.CreatedAt,
		Type:      "text",
		Sig:       "",
		Author: &pb.Contact{
			Id:   msg.Author.ID.String(),
			Name: msg.Author.Name,
		},
	}
}


type ChatInfo struct {
	ID          ID
	Name        string
	Members     []Contact
	Type        ChatType
	Unread      uint64
	LatestText  string
}
