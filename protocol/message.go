package protocol

import (
	"time"

	"github.com/hood-chat/core/pb"
)

type MessageProtocol = Protocol[*pb.Message]
type messageProtocol = protocol[*pb.Message]


func NewMessageProtocol() MessageProtocol {
	meta := new(Meta)
	meta.MessageTimeout = time.Second * 60
	meta.ID = "/chat/message/0.0.1"
	meta.ServiceName = "chat.message"
	meta.MaxMsgSize = 10 * 1024 // 4K
	meta.StreamTimeout  = time.Minute
	meta.ConnectTimeout = 30 * time.Second
	return messageProtocol{
		meta: *meta,
		m: func () *pb.Message {
			return &pb.Message{}
		},
	}
}

var Message = NewMessageProtocol()