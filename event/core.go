package event

import (
	"errors"

	"github.com/hood-chat/core/entity"
	"github.com/libp2p/go-libp2p/core/event"
)

var ErrNotSupported = errors.New("not implemented")

type CoreEvent[A any, P any] interface {
	Name() string
	Group() string
	Action() *A
	Payload() *P
}

type Core[C any, P any] interface {
	Make(name string, action C, payload P) (*EvtObject, error)
	Parse(*EvtObject) (CoreEvent[C, P], error)
	Validate(*EvtObject) bool
}

type EvtObject struct {
	Name    string
	Group   string
	Action  string
	Payload string
}

func NewEvtObj(n string, g string, a string, p string) *EvtObject {
	return &EvtObject{n, g, a, p}
}

type MessagingEventGroup struct {
	Actions map[entity.Status]string
	Names   map[string]string
}

func NewMessagingEventGroup() MessagingEventGroup {
	return MessagingEventGroup{
		Actions: map[entity.Status]string{entity.Seen: "seen", entity.Sent: "sent", entity.Pending: "pending", entity.Received: "received", entity.Failed: "failed"},
		Names:   map[string]string{"ChangeMessageStatus": "ChangeMessageStatus"},
	}
}

func (e MessagingEventGroup) Make(name string, action entity.Status, payload entity.ID) (*EvtObject, error) {
	n, pres := e.Names[name]
	if !pres {
		return nil, ErrNotSupported
	}
	a, pres := e.Actions[action]
	if !pres {
		return nil, ErrNotSupported
	}
	evt := NewEvtObj(n, "Messaging", a, payload.String())
	return evt, nil
}

func (e MessagingEventGroup) Parse(evt *EvtObject) (CoreEvent[entity.Status, entity.ID], error) {
	n, pres := e.Names[evt.Name]
	if !pres {
		return nil, ErrNotSupported
	}
	a, pres := mapkey(e.Actions, evt.Action)
	if !pres {
		return nil, ErrNotSupported
	}
	msgEvent := MessagingEvent{
		name:    n,
		grp:     "Messaging",
		action:  a,
		payload: entity.ID(evt.Payload),
	}
	return CoreEvent[entity.Status, entity.ID](msgEvent), nil

}

func (e MessagingEventGroup) Validate(evt *EvtObject) bool {
	_, pres := e.Names[evt.Name]
	if !pres {
		return false
	}
	_, pres = mapkey(e.Actions, evt.Action)
	return pres
}

type MessagingEvent struct {
	name    string
	grp     string
	action  entity.Status
	payload entity.ID
}

func (e MessagingEvent) Name() string {
	return e.name
}
func (e MessagingEvent) Group() string {
	return e.grp
}
func (e MessagingEvent) Action() *entity.Status {
	return &e.action
}
func (e MessagingEvent) Payload() *entity.ID {
	return &e.payload
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


func EmitMessageChange(emitter event.Emitter, status entity.Status, msgID string) {
	evgrp := NewMessagingEventGroup()
	ev, err := evgrp.Make("ChangeMessageStatus", status, entity.ID(msgID))
	if err != nil {
		panic("bus has problem")
	}
	err = emitter.Emit(*ev)
	if err != nil {
		panic("bus has problem")
	}
}