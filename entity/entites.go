package entity

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sort"
	"strings"

	"github.com/hood-chat/core/pb"
	"github.com/hood-chat/core/protocol/message"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"google.golang.org/protobuf/proto"
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
	ID      ID
	Name    string
	PrivKey string
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

type ProtoMessage interface {
	Proto() proto.Message
}


type Message struct {
	ID        ID
	ChatID    ID
	CreatedAt int64
	Text      string
	Status    Status
	Author    Contact
	ChatType  ChatType
}

func (m Message) Proto() proto.Message {
	msg := m
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
		ChatType: pb.CHAT_TYPES(msg.ChatType),
	}
}

func ToMessage(pbmsg *pb.Message) Message {
	mAuthorID := ID(pbmsg.Author.Id)
	msgID := ID(pbmsg.GetId())
	chatID := ID(pbmsg.ChatId)
	con := Contact{
		ID:   mAuthorID,
		Name: pbmsg.Author.Name,
	}
	return Message{
		ID:        msgID,
		ChatID:    chatID,
		CreatedAt: pbmsg.GetCreatedAt(),
		Text:      pbmsg.GetText(),
		Status:    Received,
		Author:    con,
		ChatType:  ChatType(pbmsg.ChatType),
	}
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
	To        Contact
	Message   ProtoMessage
	ID        string
	CreatedAt int64
	Protocol  protocol.ID
}

func (e Envelop) PeerID() peer.ID {
	pi, _ := e.To.PeerID()
	return pi
}

type PubSubEnvelop struct {
	Topic     string
	Message   ProtoMessage
}

func NewMessageEnvelop(c Contact, m Message) (*Envelop, error) {
	_, err := c.PeerID()
	if err != nil {
		return nil, err
	}
	return &Envelop{
		To: c,
		Message: m,
		ID: m.ID.String(),
		CreatedAt: m.CreatedAt,
		Protocol: direct.ID,
	}, nil
}


type ChatInfo struct {
	ID          ID
	Name        string
	Members     []Contact
	Type        ChatType
	Unread      uint64
	LatestText  string
	Admins      []Contact
}

func ToChatInfo(pbmsg *pb.Request) ChatInfo {
	ci := new(ChatInfo)
	ci.ID = ID(pbmsg.Id)
	ci.Name = pbmsg.Name
	for _,v := range pbmsg.Members {
		ci.Members = append(ci.Members, Contact{ID(v.Id),v.Name})
	}
	for _,v := range pbmsg.Admins {
		ci.Admins = append(ci.Admins, Contact{ID(v.Id),v.Name})
	}
	return *ci
}

func (m ChatInfo) Proto() proto.Message {
	r := &pb.Request{
		Id: m.ID.String(),
		ChatType: pb.CHAT_TYPES(m.Type),
	}
	for _,v := range m.Members {
		r.Members = append(r.Members, &pb.Contact{Name:v.Name, Id: v.ID.String()})
	}
	for _,v := range m.Admins {
		r.Admins = append(r.Admins, &pb.Contact{Name:v.Name, Id: v.ID.String()})
	}
	return r
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