package event

import (
	"github.com/hood-chat/core/entity"
	"github.com/hood-chat/core/pb"
)

type EvtMessageStatusChange struct {
	ID     entity.ID
	Status entity.Status
}

type EvtMessageReceived struct {
	Msg *pb.Message
}
