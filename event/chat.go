package event

import (

	"github.com/hood-chat/core/entity"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/p2p/host/eventbus"
)

// Event group name
const ChatGroup = "ChatEvent"
// Event Names
const Invite = "INVITE"
// Event Actions
const InviteSent       = "SENT"
const InviteReceived   = "RECEIVED"

type ChatEvent = IEvent[string, interface{}]
type ChatEventGroup = IEventGroup[string, interface{}]
type ChatEventObj = EvtObject[string, interface{}]

var NewChatEvent = NewEvtObj[string, interface{}]

var ChatEG = NewChatEventGroup()

type chatEG struct {
	Actions map[string]Empty
	Names   map[string]Empty
}


func NewChatEventGroup() ChatEventGroup {
	return &chatEG{
		Actions: map[string]Empty{
			InviteSent:     {},
			InviteReceived: {},
		},
		Names: map[string]Empty{
			Invite: {},
		},
	}
}


func (e *chatEG) NewEvent(name string, action string, payload interface{}) (ChatEvent, error) {
	_, pres := e.Names[name]
	if !pres {
		return nil, ErrNotSupported
	}
	_, pres = e.Actions[action]
	if !pres {
		return nil, ErrNotSupported
	}

	switch payload.(type) {
	case entity.ID:
		break
	case entity.ChatInfo:
		break
	default:
		return nil, ErrNotSupported
	}

	evt := NewChatEvent(name, ChatGroup, action, payload)
	return evt, nil
}

func (e *chatEG) Validate(evt ChatEvent) bool {
	if evt.GetGroup() != ChatGroup {
		return false
	}
	_, pres := e.Names[evt.GetName()]
	if !pres {
		return false
	}
	_, pres = e.Actions[evt.GetAction()]
	return pres
}

func EmitInvite(bus event.Bus, action string, chat entity.ChatInfo) {
	emitter, err := bus.Emitter(new(ChatEventObj), 	eventbus.Stateful)
	if err != nil {
		panic("create emitter failed")
	}
	defer emitter.Close()
	ev, err := ChatEG.NewEvent(Invite, action, chat)
	if err != nil {
		panic(err)
	}
	err = emitter.Emit(ev)
	if err != nil {
		panic("emit event failed")
	}
}
