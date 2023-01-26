package event

import (
	"encoding/json"

	"github.com/hood-chat/core/entity"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/p2p/host/eventbus"
)

const Group = "messaging"
type Empty struct{}
const ChangeStatus = "ChangeStatus"
const NewMessage   = "NewMessage"

type MessageEvent = IEvent[entity.Status, interface{}]
type MessageEventGroup = IEventGroup[entity.Status, interface{}]
type MessageEventObj = EvtObject[entity.Status, interface{}]

var NewMessageEvent = NewEvtObj[entity.Status, interface{}]

var MessagingEG = NewMessagingEventGroup()

type MessageStatusEG struct {
	Actions map[entity.Status]string
	Names   map[string]Empty
}

func NewMessagingEventGroup() MessageEventGroup {
	return &MessageStatusEG{
		Actions: map[entity.Status]string{
			entity.Seen:     "seen",
			entity.Sent:     "sent",
			entity.Pending:  "pending",
			entity.Received: "received",
			entity.Failed:   "failed",
		},
		Names: map[string]Empty{
			ChangeStatus: {},
			NewMessage:   {},
		},
	}
}

func (e MessageStatusEG) NewEvent(name string, action entity.Status, payload interface{}) (MessageEvent, error) {
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

	evt := NewMessageEvent(name, Group, action, payload)
	return evt, nil
}

func (e MessageStatusEG) Validate(evt MessageEvent) bool {
	if evt.Group() != Group {
		return false
	}
	_, pres := e.Names[evt.Name()]
	if !pres {
		return false
	}
	_, pres = e.Actions[evt.Action()]
	return pres
}

func (e MessageStatusEG) Cast(evt MessageEvent) ExternalEvent {
	var payload string
	switch p:= evt.Payload().(type) {
	case entity.ID:
		payload = p.String()
	case entity.Message:
		b, err := json.Marshal(p)
		if err != nil {
			return nil
		}
		payload = string(b)
	default:
		return nil
	}
	msgEvent := NewExternalEvent(evt.Name(), Group, e.Actions[evt.Action()], payload)
	return msgEvent

}

func mapkey(m map[entity.Status]string, value string) (key entity.Status, ok bool) {
	for k, v := range m {
		if v == value {
			key = k
			ok = true
			return
		}
	}
	return
}

func EmitMessageChange(bus event.Bus, status entity.Status, msgID string) {
	emitter, err := bus.Emitter(new(MessageEventObj), 	eventbus.Stateful)
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
	emitter, err := bus.Emitter(new(MessageEventObj), 	eventbus.Stateful)
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
