package invite

import (
	"time"

	"github.com/hood-chat/core/pb"

	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-msgio/pbio"
)

var log = logging.Logger("chat-invite")

const (
	MessageTimeout = time.Second * 60

	ID = "/chat/invite/0.0.1"

	ServiceName = "chat.invite"

	MaxMsgSize = 4 * 1024 // 4K

	StreamTimeout  = time.Minute
	ConnectTimeout = 30 * time.Second
)


func SetInviteHandler(h host.Host, cb func(m *pb.Request)){
	h.SetStreamHandler(ID, func(s network.Stream) {
		msg, err  := read(s)
		if err != nil {
			log.Error("failed to read the message: ", err)
		}
		cb(msg)
	})
}

func Send(s network.Stream, pbmsg *pb.Request) error {
	log.Debug("direct: sending message")
	if err := s.Scope().ReserveMemory(MaxMsgSize, network.ReservationPriorityAlways); err != nil {
		log.Debugf("error reserving memory for message stream: %s", err)
		s.Reset()
		return err
	}
	defer s.Scope().ReleaseMemory(MaxMsgSize)
	wr := pbio.NewDelimitedWriter(s)
	defer func() {
		wr.Close()
	}()
	log.Debugf("chat request send for: %s", pbmsg.GetId())
	err := wr.WriteMsg(pbmsg)
	if err != nil {
		log.Errorf("write err %s", err)
		return err
	}
	return nil
}

func read(str network.Stream) (*pb.Request,error) {
	defer str.Close()
	if err := str.Scope().SetService(ServiceName); err != nil {
		log.Debugf("error attaching stream to ping service: %s", err)
		str.Reset()
		return nil, err
	}

	if err := str.Scope().ReserveMemory(MaxMsgSize, network.ReservationPriorityAlways); err != nil {
		log.Debugf("error reserving memory for Private Message stream: %s", err)
		str.Reset()
		return nil, err
	}
	defer str.Scope().ReleaseMemory(MaxMsgSize)

	rd := pbio.NewDelimitedReader(str, MaxMsgSize)
	defer rd.Close()

	str.SetDeadline(time.Now().Add(StreamTimeout))

	msg := new(pb.Request)

	err := rd.ReadMsg(msg)
	if err != nil {
		log.Errorf("error reading message: %s", err.Error())
		str.Reset()
		return nil, err
	}
	return msg, nil
}