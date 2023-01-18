package core

import (
	"testing"
	"time"

	"github.com/hood-chat/core/entity"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
)

func getPeers(n int) []peer.AddrInfo {
	peers := make([]peer.AddrInfo, 0)
	for i := 0; i < n; i++ {
		p, _ := entity.CreateIdentity("asd")
		a, _ := p.ToContact().AdderInfo()
		peers = append(peers, *a)
	}
	return peers
}

func TestPeerSet(t *testing.T) {
	needed := NewPeerSet()
	peers := getPeers(5)

	// test add and remove and all done
	needed.Add("t1", peers[0])
	require.False(t, needed.Empty())
	needed.Add("t1", peers[0])
	require.False(t, needed.Empty())
	needed.Remove("t1", peers[0].ID)
	require.False(t, needed.Empty())
	needed.Remove("t1", peers[0].ID)
	require.True(t, needed.Empty())

	// test turn and done
	needed.Add("t1", peers[0])
	turn := needed.Turn(time.Now())
	require.Exactly(t, peers[0], turn[0])
	turn = needed.Turn(time.Now())
	require.Empty(t, turn)
	require.False(t, needed.Empty())
	needed.Done(peers[0].ID)
	turn = needed.Turn(time.Now())
	require.Empty(t, turn)
	needed.Remove("t1", peers[0].ID)
	require.True(t, needed.Empty())

	// test force
	needed.Add("t1", peers[0])
	needed.Force(peers[0].ID)
	turn = needed.Turn(time.Now())
	require.Empty(t, turn)
	needed.Remove("t1", peers[0].ID)
	require.True(t, needed.Empty())

	// test fail
	needed.Add("t1", peers[0])
	needed.Force(peers[0].ID)
	needed.Failed(peers[0].ID)
	require.Empty(t, needed.Turn(time.Now()))
	require.Eventually(t, func() bool { return len(needed.Turn(time.Now())) != 0 }, 10*time.Second, 1*time.Second)
	require.Empty(t, needed.Turn(time.Now()))
	needed.Remove("t1", peers[0].ID)
	require.True(t, needed.Empty())

	// test counter
	needed.Add("t1", peers[0])
	require.False(t, needed.Empty())
	needed.Add("t1", peers[0])
	require.False(t, needed.Empty())
	needed.Remove("t1", peers[0].ID)
	require.False(t, needed.Empty())
	needed.Remove("t1", peers[0].ID)
	require.True(t, needed.Empty())
	needed.Remove("t1", peers[0].ID)
	// needed.Remove("t1", peers[0].ID)
}

func TestPeerSetRemove(t *testing.T) {
	needed := NewPeerSet()
	peers := getPeers(5)
	needed.Add("t1", peers[0])
	require.False(t, needed.Empty())
	needed.Remove("t1", peers[0].ID)
	needed.Remove("t1", peers[0].ID)
	// needed.Remove("t1", peers[0].ID)
}