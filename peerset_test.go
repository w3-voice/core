package core

import (
	"testing"
	"time"

	"github.com/hood-chat/core/entity"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/require"
)

func getPeers(n int) []peer.AddrInfo {
	peers := make([]peer.AddrInfo, 0)
	for i := 0; i < n; i++ {
		p, _ := entity.CreateIdentity("asd")
		a, _ := p.Me().AdderInfo()
		peers = append(peers, *a)
	}
	return peers
}

func TestPeerSet(t *testing.T) {
	needed := NewPeerSet()
	peers := getPeers(5)

	// test add and remove and all done
	needed.Add("t1", peers[0])
	require.False(t, needed.empty())
	needed.Add("t1", peers[0])
	require.False(t, needed.empty())
	needed.Remove("t1", peers[0].ID)
	require.False(t, needed.empty())
	needed.Remove("t1", peers[0].ID)
	require.True(t, needed.empty())

	// test turn and done
	needed.Add("t1", peers[0])
	turn := needed.turn(time.Now())
	require.Exactly(t, peers[0], turn[0])
	turn = needed.turn(time.Now())
	require.Empty(t, turn)
	require.False(t, needed.empty())
	needed.done(peers[0].ID)
	turn = needed.turn(time.Now())
	require.Empty(t, turn)
	needed.Remove("t1", peers[0].ID)
	require.True(t, needed.empty())
	
	// test force
	needed.Add("t1", peers[0])
	needed.force(peers[0].ID)
	turn = needed.turn(time.Now())
	require.Empty(t, turn)
	needed.Remove("t1", peers[0].ID)
	require.True(t, needed.empty())
	
	// test fail
	needed.Add("t1", peers[0])
	needed.force(peers[0].ID)
	needed.fail(peers[0].ID)
	require.Empty(t, needed.turn(time.Now()))
	require.Eventually(t, func() bool {return len(needed.turn(time.Now())) != 0},5*time.Second,1*time.Second)
	require.Empty(t, needed.turn(time.Now()))
}
