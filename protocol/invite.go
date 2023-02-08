package protocol

import (
	"time"

	"github.com/hood-chat/core/pb"

)

type InviteProtocol = Protocol[*pb.Request]
type inviteProtocol = protocol[*pb.Request]


func NewInviteProtocol() InviteProtocol {
	meta := new(Meta)

	meta.MessageTimeout = time.Second * 60
	meta.ID = "/chat/invite/0.0.1"
	meta.ServiceName = "chat.invite"
	meta.MaxMsgSize = 10 * 1024 // 4K
	meta.StreamTimeout  = time.Minute
	meta.ConnectTimeout = 30 * time.Second
	return inviteProtocol{
		meta: *meta,
		m: func () *pb.Request {
			return &pb.Request{}
		},
	}
}

var Invite = NewInviteProtocol()