package protocol

import (
	"time"

	"github.com/hood-chat/core/pb"
	"google.golang.org/protobuf/proto"

	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	pl "github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-msgio/pbio"
)

type Meta struct {
	MessageTimeout time.Duration

	ID pl.ID

	ServiceName string

	MaxMsgSize int // 4K

	StreamTimeout  time.Duration
	ConnectTimeout time.Duration
}

var log = logging.Logger("chat-protocols")

type Protocol[M any] interface {
	SetHandler(h host.Host, cb func(m M))
	Send(s network.Stream, pbmsg M) error
	GetMeta() *Meta
	ID() pl.ID
	read(str network.Stream) (M, error)
}

type protocol[M proto.Message] struct {
	Protocol[M]
	meta Meta
	m    func() M
}

func (p protocol[M]) ID() pl.ID {
	return p.meta.ID
}

func (p protocol[M]) SetHandler(h host.Host, cb func(m M)) {
	h.SetStreamHandler(p.meta.ID, func(s network.Stream) {
		msg, err := p.read(s)
		if err != nil {
			log.Error("failed to read the message: ", err)
		}
		cb(msg)
	})
}

func (p protocol[M]) Send(s network.Stream, pbmsg M) error {
	log.Debug("direct: sending message")
	if err := s.Scope().ReserveMemory(p.meta.MaxMsgSize, network.ReservationPriorityAlways); err != nil {
		log.Debugf("error reserving memory for message stream: %s", err)
		s.Reset()
		return err
	}
	defer s.Scope().ReleaseMemory(p.meta.MaxMsgSize)
	wr := pbio.NewDelimitedWriter(s)
	rd := pbio.NewDelimitedReader(s, p.meta.MaxMsgSize)
	defer func() {
		wr.Close()
		rd.Close()
	}()
	// log.Debugf("text sent with message text: %s", pbmsg.GetText())
	err := wr.WriteMsg(pbmsg)
	if err != nil {
		log.Errorf("write err %s", err)
		s.Reset()
		return err
	}

	err = rd.ReadMsg(&pb.Ack{})
	if err != nil {
		log.Errorf("error reading message: %s", err.Error())
		s.Reset()
		return err
	}

	return nil
}

func (p protocol[M]) read(str network.Stream) (M, error) {
	msg := p.m()
	defer str.Close()
	str.SetDeadline(time.Now().Add(p.meta.StreamTimeout))

	if err := str.Scope().SetService(p.meta.ServiceName); err != nil {
		log.Debugf("error attaching stream to ping service: %s", err)
		str.Reset()
		return msg, err
	}

	if err := str.Scope().ReserveMemory(p.meta.MaxMsgSize, network.ReservationPriorityAlways); err != nil {
		log.Debugf("error reserving memory for Private Message stream: %s", err)
		str.Reset()
		return msg, err
	}
	defer str.Scope().ReleaseMemory(p.meta.MaxMsgSize)

	rd := pbio.NewDelimitedReader(str, p.meta.MaxMsgSize)
	wr := pbio.NewDelimitedWriter(str)

	defer func() {
		wr.Close()
		rd.Close()
	}()

	err := rd.ReadMsg(msg)
	if err != nil {
		log.Errorf("error reading message: %s", err.Error())
		str.Reset()
		return msg, err
	}

	err = wr.WriteMsg(&pb.Ack{})
	if err != nil {
		log.Errorf("write err %s", err)
		str.Reset()
		return msg, err
	}

	return msg, nil
}

func (p protocol[M]) GetMeta() *Meta {
	return &p.meta
}
