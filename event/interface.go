package event

import (
	"errors"
)

var ErrNotSupported = errors.New("not implemented")

type IEvent[A any, P any] interface {
	Name() string
	Group() string
	Action() A
	Payload() P
}

type IEventGroup[A any, P any] interface {
	NewEvent(name string, action A, payload P) (IEvent[A,P], error)
	Validate(IEvent[A,P]) bool
}

type EvtObject[A any, P any] struct {
	name    string
	group   string
	action  A
	payload P
}

func NewEvtObj[A any, P any](n string, g string, a A, p P) IEvent[A,P] {
	return EvtObject[A,P]{n, g, a, p}
}

func (e EvtObject[A , P]) Name() string {
	return e.name
}
func (e EvtObject[A , P]) Group() string {
	return e.group
}
func (e EvtObject[A , P]) Action() A {
	return e.action
}
func (e EvtObject[A , P]) Payload() P {
	return e.payload
}


type ExternalEvent = IEvent[string,string]
type ExternalEventObj   = EvtObject[string,string]
var NewExternalEvent = NewEvtObj[string,string]

type Publishable interface {
	Cast() ExternalEvent
}