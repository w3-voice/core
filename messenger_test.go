package core_test

import (
	"context"
	"testing"
	"time"

	"github.com/bee-messenger/core"

	"github.com/bee-messenger/core/entity"
	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	bhost "github.com/libp2p/go-libp2p/p2p/host/basic"
	swarmt "github.com/libp2p/go-libp2p/p2p/net/swarm/testing"
	"github.com/stretchr/testify/require"

	ma "github.com/multiformats/go-multiaddr"
)

func TestMessenger(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h1, err := bhost.NewHost(swarmt.GenSwarm(t), nil)
	require.NoError(t, err)
	defer h1.Close()
	h2, err := bhost.NewHost(swarmt.GenSwarm(t), nil)
	require.NoError(t, err)
	defer h2.Close()

	err = h1.Connect(ctx, peer.AddrInfo{
		ID:    h2.ID(),
		Addrs: []ma.Multiaddr{h2.Addrs()[0]},
	})
	require.NoError(t, err)

	user1 := entity.Contact{
		ID:   h1.ID().String(),
		Name: "identity1",
	}

	user2 := entity.Contact{
		ID:   h2.ID().String(),
		Name: "identity2",
	}

	mr1 := core.MessengerBuilder(t.TempDir()+"/h3", h1)
	mr2 := core.MessengerBuilder(t.TempDir()+"/h4", h2)

	h1.Peerstore().AddAddr(h2.ID(), h2.Addrs()[0], peerstore.PermanentAddrTTL)
	// _, err = h1.NewStream(ctx, h2.ID(), core.ID)
	// require.NoError(t, err)

	chat1 := mr1.CreateChat(uuid.New().String(), []entity.Contact{user1, user2}, user2.Name)
	env, err := chat1.NewMessage("hello")
	if err != nil {
		t.Errorf("new message failed %s", err)
	}
	to, err := peer.Decode(env.To)
	require.NoError(t, err)
	if h2.ID() != to {

		t.Errorf("peers are not the same dist: %s, to: %s, host: %s", h2.ID().String(), env.To, h1.ID().String())
	}
	mr1.SendMessage(*env)

	time.Sleep(5 * time.Second)

	t.Log(env)
	chat2, err := mr2.GetChat(chat1.ID())
	require.NoError(t, err)
	msgs, err := chat2.GetMessages()
	require.NoError(t, err)
	t.Logf("list of messages \n %v", msgs[0].Text)

}
