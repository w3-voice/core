package core

import (
	"encoding/json"
	"time"

	"github.com/hood-chat/core/entity"
	"github.com/hood-chat/core/protocol"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/peer"
	pl "github.com/libp2p/go-libp2p/core/protocol"
)

func Of[E any](e E) *E {
    return &e
}


type Emitter = event.Emitter
type Bus     = event.Bus

type SearchChatOpt struct {
	Name    *string             `json:"name,omitempty"`
	Members []entity.ID			`json:"members,omitempty"`	
	Type    *entity.ChatType	`json:"type,omitempty"`
}

func (m *SearchChatOpt) Json() ([]byte, error) {
	bytes, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}
func (m *SearchChatOpt) DTO() SearchChatOpt {
	return *m
}

func WithPrivateChatContact(contactID entity.ID) SearchChatOpt {
	return SearchChatOpt{nil, []entity.ID{contactID}, Of(entity.Private)}
}

type NewChatOpt struct {
	Name    *string				`json:"name,omitempty"`
	Members []entity.Contact	`json:"members,omitempty"`
	Type    *entity.ChatType	`json:"type,omitempty"`
}

func (m *NewChatOpt) Json() ([]byte, error) {
	bytes, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}
func (m *NewChatOpt) DTO() NewChatOpt {
	return *m
}

func NewPrivateChat(contact entity.Contact) NewChatOpt {
	return NewChatOpt{nil, []entity.Contact{contact}, Of(entity.Private)}
}

func NewGroupChat(n string, c []entity.Contact) NewChatOpt {
	return NewChatOpt{Name: &n, Members: c, Type: Of(entity.Group)}
}

type ChatRequest struct {
	ID          entity.ID
	Name        string
	Members     []entity.Contact
	Type        entity.ChatType
	Admins      []entity.Contact
}

type Envelop struct {
	To        entity.Contact
	Message   entity.ProtoMessage
	ID        string
	CreatedAt int64
	Protocol  pl.ID
}

func (e Envelop) createdAt() time.Time {
	return time.Unix(e.CreatedAt, 0)
}
func (e Envelop) id() string {
	return e.ID
}

func (e Envelop) PeerID() peer.ID {
	pi, _ := e.To.PeerID()
	return pi
}

type PubSubEnvelop struct {
	Topic     string
	Message   entity.ProtoMessage
	CreatedAt int64
}

func (e PubSubEnvelop) createdAt(t time.Time) time.Time  {
	return time.Unix(e.CreatedAt, 0)
}

func NewMessageEnvelop(c entity.Contact, m entity.Message) (*Envelop, error) {
	_, err := c.PeerID()
	if err != nil {
		return nil, err
	}
	return &Envelop{
		To: c,
		Message: m,
		ID: m.ID.String(),
		CreatedAt: m.CreatedAt,
		Protocol: protocol.Message.ID(),
	}, nil
}
