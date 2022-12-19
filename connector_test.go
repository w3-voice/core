package core

import (
	"testing"
	"time"

	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p/core/host"
	bhost "github.com/libp2p/go-libp2p/p2p/host/blank"
	swarmt "github.com/libp2p/go-libp2p/p2p/net/swarm/testing"
	"github.com/stretchr/testify/require"
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

func TestPeerSet(t *testing.T) {
	t.Log("start test")
	err := logging.SetLogLevel("msgr-core", "DEBUG")
	require.NoError(t, err)
	hosts := getNetHosts(t, 5)
	primary := hosts[0]
	connector := NewConnector(primary)
	connector.Need("test", hosts[1].Peerstore().PeerInfo(hosts[1].ID()))
	connector.Need("test", hosts[1].Peerstore().PeerInfo(hosts[1].ID()))
	connector.Need("test", hosts[2].Peerstore().PeerInfo(hosts[2].ID()))
	require.Eventually(t, func() bool { return len(primary.Network().Peers()) == 2 }, 5*time.Second, 10*time.Millisecond)
}
