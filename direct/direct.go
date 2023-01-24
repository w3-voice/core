package direct

import (
	"time"

	"github.com/hood-chat/core/pb"
	"github.com/hood-chat/core/utils"

	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-msgio/protoio"
)

var log = logging.Logger("chat-direct")

const (
	MessageTimeout = time.Second * 60

	ID = "/chat/direct/0.0.1"

	ServiceName = "chat.direct"

	MaxMsgSize = 10 * 1024 // 4K

	StreamTimeout  = time.Minute
	ConnectTimeout = 30 * time.Second
)


func SetMessageHandler(h host.Host,cb func(m *pb.Message)){
	h.SetStreamHandler(ID, func(s network.Stream) {
		msg, err  := read(s)
		if err != nil {
			log.Error("failed to read the message: ", err)
		}
		cb(msg)
	})
}

func Send(s network.Stream, pbmsg *pb.Message) error {
	log.Debug("direct: sending message")
	if err := s.Scope().ReserveMemory(MaxMsgSize, network.ReservationPriorityAlways); err != nil {
		log.Debugf("error reserving memory for message stream: %s", err)
		s.Reset()
		return err
	}
	defer s.Scope().ReleaseMemory(MaxMsgSize)
	wr := protoio.NewDelimitedWriter(s)
	defer func() {
		wr.Close()
	}()
	log.Debugf("text sent with message text: %s", pbmsg.GetText())
	err := wr.WriteMsg(pbmsg)
	if err != nil {
		log.Errorf("write err %s", err)
		return err
	}
	return nil
}

func read(str network.Stream) (*pb.Message,error) {
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

	rd := utils.NewDelimitedReader(str, MaxMsgSize)
	defer rd.Close()

	str.SetDeadline(time.Now().Add(StreamTimeout))

	msg := new(pb.Message)

	err := rd.ReadMsg(msg)
	if err != nil {
		log.Errorf("error reading message: %s", err.Error())
		str.Reset()
		return nil, err
	}
	return msg, nil
}