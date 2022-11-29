package event

import (
	"github.com/hood-chat/core/pb"
)

type EvtMessageReceived struct {
	Msg *pb.Message
}
