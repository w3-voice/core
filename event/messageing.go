package event

import (
	"github.com/hood-chat/core/entity"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/p2p/host/eventbus"
)

const MessageGroup = "Messaging"

type Empty struct{}

const ChangeStatus = "ChangeStatus"
const NewMessage = "NewMessage"

type MessageEvent = IEvent[entity.Status, interface{}]
type MessageEventGroup = IEventGroup[entity.Status, interface{}]
type MessageEventObj = EvtObject[entity.Status, interface{}]

var NewMessageEvent = NewEvtObj[entity.Status, interface{}]

var MessagingEG = NewMessagingEventGroup()

type messagingEG struct {
	Actions map[entity.Status]string
	Names   map[string]Empty
}

func NewMessagingEventGroup() *messagingEG {
	return &messagingEG{
		Actions: map[entity.Status]string{
			entity.Seen:     "Seen",
			entity.Sent:     "Sent",
			entity.Pending:  "Pending",
			entity.Received: "Received",
			entity.Failed:   "Failed",
		},
		Names: map[string]Empty{
			ChangeStatus: {},
			NewMessage:   {},
		},
	}
}

func (e *messagingEG) NewEvent(name string, action entity.Status, payload interface{}) (MessageEvent, error) {
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
	case entity.Message:
		break
	default:
		return nil, ErrNotSupported
	}

	evt := NewMessageEvent(name, MessageGroup, action, payload)
	return evt, nil
}

func (e *messagingEG) Validate(evt MessageEvent) bool {
	if evt.GetGroup() != MessageGroup {
		return false
	}
	_, pres := e.Names[evt.GetName()]
	if !pres {
		return false
	}
	_, pres = e.Actions[evt.GetAction()]
	return pres
}

func EmitMessageChange(bus event.Bus, status entity.Status, msgID string) {
	emitter, err := bus.Emitter(new(MessageEventObj), eventbus.Stateful)
	if err != nil {
		panic("bus has problem")
	}
	defer emitter.Close()
	ev, err := MessagingEG.NewEvent(ChangeStatus, status, entity.ID(msgID))
	if err != nil {
		panic("bus has problem")
	}
	err = emitter.Emit(ev)
	if err != nil {
		panic("bus has problem")
	}
}

func EmitNewMessage(bus event.Bus, msg entity.Message) {
	emitter, err := bus.Emitter(new(MessageEventObj), eventbus.Stateful)
	if err != nil {
		panic("bus has problem")
	}
	defer emitter.Close()
	ev, err := MessagingEG.NewEvent(NewMessage, entity.Received, msg)
	if err != nil {
		panic("bus has problem")
	}
	err = emitter.Emit(ev)
	if err != nil {
		panic("bus has problem")
	}
}
