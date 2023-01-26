package event

import (
	"errors"
)

var ErrNotSupported = errors.New("not implemented")

type IEvent[A any, P any] interface {
	GetName() string
	GetGroup() string
	GetAction() A
	GetPayload() P
}

type IEventGroup[A any, P any] interface {
	NewEvent(name string, action A, payload P) (IEvent[A,P], error)
	Validate(IEvent[A,P]) bool
}

type EvtObject[A any, P any] struct {
	Name    string
	Group   string
	Action  A
	Payload P
}

func NewEvtObj[A any, P any](n string, g string, a A, p P) IEvent[A,P] {
	return EvtObject[A,P]{n, g, a, p}
}

func (e EvtObject[A , P]) GetName() string {
	return e.Name
}
func (e EvtObject[A , P]) GetGroup() string {
	return e.Group
}
func (e EvtObject[A , P]) GetAction() A {
	return e.Action
}
func (e EvtObject[A , P]) GetPayload() P {
	return e.Payload
}
