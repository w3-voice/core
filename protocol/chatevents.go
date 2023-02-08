package protocol

import (
	"time"

	"github.com/hood-chat/core/pb"
)

type ChatEventProtocol = Protocol[*pb.ChatEvent]
type chatEventProtocol = protocol[*pb.ChatEvent]


func NewCEProtocol() ChatEventProtocol {
	meta := new(Meta)

	meta.MessageTimeout = time.Second * 60
	meta.ID = "/chat/chat_event/0.0.1"
	meta.MaxMsgSize = 10 * 1024 // 4K
	meta.ServiceName = "chat.event"
	meta.StreamTimeout  = time.Minute
	meta.ConnectTimeout = 30 * time.Second
	return chatEventProtocol{
		meta: *meta,
		m: func () *pb.ChatEvent {
			return &pb.ChatEvent{}
		},
	}
}

var ChatEvent = NewCEProtocol()