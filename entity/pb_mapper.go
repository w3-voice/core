package entity

import (
	"github.com/hood-chat/core/pb"
	"google.golang.org/protobuf/proto"
)

type ProtoMessage interface {
	Proto() proto.Message
}

func (m Message) Proto() proto.Message {
	msg := m
	return &pb.Message{
		Text:      msg.Text,
		Id:        msg.ID.String(),
		ChatId:    msg.ChatID.String(),
		CreatedAt: msg.CreatedAt,
		Type:      "text",
		Sig:       "",
		Author: &pb.Contact{
			Id:   msg.Author.ID.String(),
			Name: msg.Author.Name,
		},
		ChatType: pb.CHAT_TYPES(msg.ChatType),
	}
}

func ToMessage(pbmsg *pb.Message) Message {
	mAuthorID := ID(pbmsg.Author.Id)
	msgID := ID(pbmsg.GetId())
	chatID := ID(pbmsg.ChatId)
	con := Contact{
		ID:   mAuthorID,
		Name: pbmsg.Author.Name,
	}
	return Message{
		ID:        msgID,
		ChatID:    chatID,
		CreatedAt: pbmsg.GetCreatedAt(),
		Text:      pbmsg.GetText(),
		Status:    Received,
		Author:    con,
		ChatType:  ChatType(pbmsg.ChatType),
	}
}

func ToChatInfo(pbmsg *pb.Request) ChatInfo {
	ci := new(ChatInfo)
	ci.ID = ID(pbmsg.Id)
	ci.Name = pbmsg.Name
	for _,v := range pbmsg.Members {
		ci.Members = append(ci.Members, Contact{ID(v.Id),v.Name})
	}
	for _,v := range pbmsg.Admins {
		ci.Admins = append(ci.Admins, Contact{ID(v.Id),v.Name})
	}
	return *ci
}

func (m ChatInfo) Proto() proto.Message {
	r := &pb.Request{
		Id: m.ID.String(),
		ChatType: pb.CHAT_TYPES(m.Type),
	}
	for _,v := range m.Members {
		r.Members = append(r.Members, &pb.Contact{Name:v.Name, Id: v.ID.String()})
	}
	for _,v := range m.Admins {
		r.Admins = append(r.Admins, &pb.Contact{Name:v.Name, Id: v.ID.String()})
	}
	return r
}