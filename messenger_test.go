package core_test

import (
	"testing"
	"time"

	"github.com/hood-chat/core"
	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
)

func TestMessenger(t *testing.T) {
	err := logging.SetLogLevel("msgr-core", "DEBUG")
	require.NoError(t, err)
	opt1 := core.DefaultOption()
	opt2 := core.DefaultOption()
	mr1 := core.MessengerBuilder(t.TempDir()+"/h1", opt1, core.DefaultRoutedHost{})
	_, err = mr1.SignUp("h1")
	require.NoError(t, err)
	_, err = mr1.GetIdentity()
	require.NoError(t, err)
	mr2 := core.MessengerBuilder(t.TempDir()+"/h2", opt2, core.DefaultRoutedHost{})
	_, err = mr2.SignUp("h2")
	require.NoError(t, err)
	user2, err := mr2.GetIdentity()
	require.NoError(t, err)

	err = mr1.AddContact(*user2.Me())
	require.NoError(t, err)

	time.Sleep(30 * time.Second)
	// h1.Peerstore().AddAddr(h2.ID(), h2.Addrs()[0], peerstore.PermanentAddrTTL)
	// _, err = h1.NewStream(ctx, h2.ID(), core.ID)
	// require.NoError(t, err)

	chat1, err := mr1.CreatePMChat(user2.ID)
	require.NoError(t, err)
	env, err := mr1.NewMessage(chat1.ID, "hello")
	require.NoError(t, err)
	_, err = peer.Decode(env.To)
	require.NoError(t, err)

	time.Sleep(5 * time.Second)

	chat2, err := mr2.GetChat(chat1.ID)
	require.NoError(t, err)
	require.Equal(t, chat1.ID, chat2.ID)

	msgs, err := mr2.GetMessages(chat1.ID)
	require.NoError(t, err)
	t.Logf("list of messages \n %v", msgs[0].Text)

}
