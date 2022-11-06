package entity

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p-core/crypto"
	ic "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
)

type Status int

const (
	Pending Status = iota
	Sent
	Seen
)

type Identity struct {
	ID      string
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

func (i *Identity) Me() *Contact {
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
	ident.ID = id.Pretty()
	ident.Name = name
	fmt.Printf("peer identity: %s\n", ident.ID)
	return ident, nil
}

type Message struct {
	ID        string
	CreatedAt time.Time
	Text      string
	Status    Status
	Author    Contact
}

type Contact struct {
	ID   string
	Name string
}

type ChatInfo struct {
	ID      string
	Name    string
	Members []Contact
}
