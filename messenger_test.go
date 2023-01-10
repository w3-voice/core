package core_test

import (
	"testing"
	"time"

	"github.com/hood-chat/core"
	"github.com/hood-chat/core/entity"
	logging "github.com/ipfs/go-log"
	"github.com/stretchr/testify/require"
)

func TestMessenger(t *testing.T) {
	t.Log("start test")
	err := logging.SetLogLevel("msgr-core", "DEBUG")
	require.NoError(t, err)
	err = logging.SetLogLevel("*", "DEBUG")
	require.NoError(t, err)
	opt1 := core.DefaultOption()
	opt2 := core.DefaultOption()
	mr1 := core.MessengerBuilder(t.TempDir()+"/h1", opt1, core.DefaultRoutedHost{})
	t.Log("somthing wrong")
	_, err = mr1.SignUp("h1")
	require.NoError(t, err)
	t.Log("messenger 1 created")
	_, err = mr1.GetIdentity()
	require.NoError(t, err)
	mr2 := core.MessengerBuilder(t.TempDir()+"/h2", opt2, core.DefaultRoutedHost{})
	_, err = mr2.SignUp("h2")
	require.NoError(t, err)
	user2, err := mr2.GetIdentity()
	require.NoError(t, err)
	t.Log("messenger 2 created")
	err = mr1.AddContact(*user2.Me())
	require.NoError(t, err)

	time.Sleep(5 * time.Second)
	t.Log("contact added")

	chat1, err := mr1.CreatePMChat(user2.ID)
	require.NoError(t, err)
	t.Log("Chat created")
	_, err = mr1.SendPM(chat1.ID, "hello")
	require.NoError(t, err)
	_, err = mr1.SendPM(chat1.ID, "hello")
	require.NoError(t, err)
	t.Log("message sent")
	time.Sleep(30 * time.Second)

	chat2, err := mr2.GetChat(chat1.ID)
	require.NoError(t, err)
	require.Equal(t, chat1.ID, chat2.ID)

	msgs, err := mr2.GetMessages(chat1.ID, 0, 20)
	require.NoError(t, err)
	t.Logf("list of messages \n %v", msgs)
	// Test Event hand event handler
	time.Sleep(10 * time.Second)
	msgs, err = mr1.GetMessages(chat1.ID, 0, 20)
	require.Equal(t, 2, len(msgs))
	require.NoError(t, err)

	for _, val := range msgs {
		require.Equal(t, val.Status, entity.Sent)
	}

	mr1.Stop()
	mr2.Stop()

}
