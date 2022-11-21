package core

import (
	"context"
	"time"

	"github.com/hood-chat/core/pb"
	"github.com/hood-chat/core/utils"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-msgio/protoio"
)

const (
	MessageTimeout = time.Second * 60

	ID = "/chat/pm/1.0.0"

	ServiceName = "chat.pm"

	MaxMsgSize = 10 * 1024 // 4K

	StreamTimeout  = time.Minute
	ConnectTimeout = 30 * time.Second
)

type PMService struct {
	Host host.Host
	cb   func(*pb.Message)
}

func NewPMService(h host.Host, cb func(*pb.Message)) *PMService {
	pms := &PMService{h, cb}
	h.SetStreamHandler(ID, pms.PMHandler)
	log.Debug("service PMS created")
	return pms
}

func (pms *PMService) AddPeer() {

}

func (pms *PMService) Send(env *Envelop) {
	pbmsg := &pb.Message{
		Text:      env.Msg.Text,
		Id:        env.Msg.ID,
		ChatId:    env.chatID,
		CreatedAt: time.Now().Unix(),
		Type:      "text",
		Sig:       "",
		Author: &pb.Contact{
			Id:   env.Msg.Author.ID,
			Name: env.Msg.Author.Name,
		},
	}
	p, err := peer.Decode(env.To)
	if err != nil {
		log.Errorf("can not parse peerID: %s", err)
		return
	}
	adderInfo, err := peer.AddrInfoFromString("/p2p/" + env.To)
	if err != nil {
		log.Errorf("can not parse adderInfo: %s", err)
		return
	}
	err = pms.Host.Connect(context.Background(), *adderInfo)
	if err != nil {
		log.Errorf("can not connect to peer: %s reason: %s", env.To, err.Error())
		return
	}
	send(context.Background(), pms.Host, pbmsg, p)
}

func (c *PMService) PMHandler(str network.Stream) {
	if err := str.Scope().SetService(ServiceName); err != nil {
		log.Debugf("error attaching stream to ping service: %s", err)
		str.Reset()
		return
	}

	if err := str.Scope().ReserveMemory(MaxMsgSize, network.ReservationPriorityAlways); err != nil {
		log.Debugf("error reserving memory for Private Message stream: %s", err)
		str.Reset()
		return
	}
	defer str.Scope().ReleaseMemory(MaxMsgSize)

	rd := utils.NewDelimitedReader(str, MaxMsgSize)
	defer rd.Close()

	str.SetDeadline(time.Now().Add(StreamTimeout))

	var msg pb.Message

	err := rd.ReadMsg(&msg)
	if err != nil {
		log.Debugf("error reading message: %s", err)
		str.Reset()
	}
	log.Debugf("message received ... %s", msg.GetText())
	c.cb(&msg)
	defer rd.Close()

}

func send(ctx context.Context, h host.Host, msg *pb.Message, to peer.ID) {
	s, err := h.NewStream(ctx, to, ID)
	if err != nil {
		log.Errorf("new stream failed: %s", err)
		return
	}
	if err := s.Scope().ReserveMemory(MaxMsgSize, network.ReservationPriorityAlways); err != nil {
		log.Debugf("error reserving memory for message stream: %s", err)
		s.Reset()
		return
		// return 0, err
	}
	defer s.Scope().ReleaseMemory(MaxMsgSize)
	wr := protoio.NewDelimitedWriter(s)
	defer func() {
		wr.Close()
	}()
	if err != nil {
		log.Errorf("error connecting %s", err)
		return
	}
	log.Debugf("message text: %s", msg.GetText())
	wr.WriteMsg(msg)
	if err != nil {
		log.Errorf("write err %s", err)
		return
	}
}
