package core

import (
	"context"
	"time"

	"github.com/hood-chat/core/entity"
	"github.com/hood-chat/core/event"
	"github.com/hood-chat/core/pb"
	"github.com/hood-chat/core/utils"
	lpevent "github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-msgio/protoio"
	"github.com/libp2p/go-libp2p/p2p/host/eventbus"
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
	em lpevent.Emitter
}

func NewPMService(h host.Host) *PMService {
	em, err := h.EventBus().Emitter(new(event.EvtMessageReceived),eventbus.Stateful)
	if err != nil {
		log.Errorf("error reading message: %s", err.Error())
		panic("failed to create message service")
	}
	pms := &PMService{h, em}
	h.SetStreamHandler(ID, pms.PMHandler)
	log.Debug("service PMS created")
	return pms
}

func (pms *PMService) AddPeer() {

}

func (pms *PMService) Send(pbmsg *pb.Message, to entity.ID) {
	p, err := peer.Decode(to.String())
	if err != nil {
		log.Errorf("can not parse peerID: %s", err)
		return
	}
	adderInfo, err := peer.AddrInfoFromString("/p2p/" + to.String())
	if err != nil {
		log.Errorf("can not parse adderInfo: %s", err)
		return
	}
	err = pms.Host.Connect(context.Background(), *adderInfo)
	if err != nil {
		log.Errorf("can not connect to peer: %s reason: %s", to.String(), err.Error())
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
		log.Errorf("error reading message: %s", err.Error())
		str.Reset()
		return
	}
	log.Debugf("message received ... %s", msg.GetText())
	err = c.em.Emit(event.EvtMessageReceived{Msg: &msg})
	if err != nil {
		log.Errorf("failed to emit event: %s", err.Error())
		str.Reset()
		return
	}
	// defer emmiter.Close()
}

func send(ctx context.Context, h host.Host, msg *pb.Message, to peer.ID) {
	nctx := network.WithUseTransient(ctx, "just a chat")
	s, err := h.NewStream(nctx, to, ID)
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
