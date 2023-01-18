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
	mr1 := core.NewMessengerAPI(t.TempDir()+"/h1", opt1, core.DefaultRoutedHost{})
	t.Log("somthing wrong")
	_, err = mr1.IdentityAPI().SignUp("h1")
	require.NoError(t, err)
	mr1.Start()
	t.Log("messenger 1 created")
	_, err = mr1.IdentityAPI().Get()
	require.NoError(t, err)
	mr2 := core.NewMessengerAPI(t.TempDir()+"/h2", opt2, core.DefaultRoutedHost{})
	_, err = mr2.IdentityAPI().SignUp("h2")
	mr2.Start()
	require.NoError(t, err)
	user2, err := mr2.IdentityAPI().Get()
	require.NoError(t, err)
	t.Log("messenger 2 created")
	err = mr1.ContactBookAPI().Put(*user2.ToContact())
	require.NoError(t, err)

	time.Sleep(5 * time.Second)
	t.Log("contact added")

	chat1, err := mr1.ChatAPI().New(core.ForPrivateChat(user2.ID))
	require.NoError(t, err)
	t.Log("Chat created")
	_, err = mr1.ChatAPI().Send(chat1.ID, "hello")
	require.NoError(t, err)
	_, err = mr1.ChatAPI().Send(chat1.ID, "hello")
	require.NoError(t, err)
	t.Log("message sent")
	time.Sleep(30 * time.Second)

	chat2, err := mr2.ChatAPI().ChatInfo(chat1.ID)
	require.NoError(t, err)
	require.Equal(t, chat1.ID, chat2.ID)

	msgs, err := mr2.ChatAPI().Messages(chat1.ID, 0, 20)
	require.NoError(t, err)
	t.Logf("list of messages \n %v", msgs)
	
	// Test Event hand event handler
	time.Sleep(10 * time.Second)
	msgs, err = mr1.ChatAPI().Messages(chat1.ID, 0, 20)
	require.Equal(t, 2, len(msgs))
	require.Equal(t, int(chat2.Unread), len(msgs))
	require.NoError(t, err)
	for _, val := range msgs {
		require.Equal(t, val.Status, entity.Sent)
	}

	// Test Seen
	err = mr2.ChatAPI().Seen(chat1.ID)
	require.NoError(t, err)
	chat2, err = mr2.ChatAPI().ChatInfo(chat1.ID)
	require.NoError(t, err)
	require.Equal(t, 0, int(chat2.Unread))


	mr1.Stop()
	mr2.Stop()

}
