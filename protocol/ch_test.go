package protocol

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/hood-chat/core/pb"
	"github.com/libp2p/go-libp2p/core/host"
	bhost "github.com/libp2p/go-libp2p/p2p/host/blank"
	swarmt "github.com/libp2p/go-libp2p/p2p/net/swarm/testing"
)

func getNetHosts(t *testing.T, n int) []host.Host {
	var out []host.Host

	for i := 0; i < n; i++ {
		netw := swarmt.GenSwarm(t)
		h := bhost.NewBlankHost(netw)
		t.Cleanup(func() { h.Close() })
		out = append(out, h)
	}

	return out
}


func TestChatEvent(t *testing.T) {
	p := NewCEProtocol()
	
	hosts := getNetHosts(t, 5)
	for _, h := range hosts {
		p.SetHandler(h,func(m *pb.ChatEvent) {
			t.Logf("message received %v", m)
		})
	}

	primary := hosts[0]
	err := primary.Connect(context.Background(), hosts[1].Network().Peerstore().PeerInfo(hosts[1].ID()))
	require.NoError(t, err)
	s, err := primary.NewStream(context.Background(), hosts[1].ID(), p.ID())
	require.NoError(t, err)
	p.Send(s, &pb.ChatEvent{ChatId:"1",MsgId: "1", Event:pb.ChatEvent_Deliverd})
	time.Sleep(5 * time.Second)
}