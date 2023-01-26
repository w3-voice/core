package core_test

import (
	"fmt"
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
	// err = logging.SetLogLevel("*", "DEBUG")
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

	chat1, err := mr1.ChatAPI().New(core.NewPrivateChat(*user2.ToContact()))
	require.NoError(t, err)
	chat1, err = mr1.ChatAPI().ChatInfo(chat1.ID)
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

func getMessengers(t *testing.T, n int) []core.MessengerAPI {
	messengers := make([]core.MessengerAPI, 0)
	for i := 0; i < n; i++ {
		name := "h" + fmt.Sprint(i)
		opt := core.DefaultOption()
		mr := core.NewMessengerAPI(t.TempDir()+"/"+name, opt, core.DefaultRoutedHost{})
		_, err := mr.IdentityAPI().SignUp(name)
		if err != nil {
			panic("cant create host")
		}
		require.NoError(t, err)
		mr.Start()
		messengers = append(messengers, mr)
	}
	return messengers
}

func TestGroup(t *testing.T) {
	err := logging.SetLogLevel("msgr-core", "DEBUG")
	require.NoError(t, err)
	// err = logging.SetLogLevel("*", "DEBUG")
	require.NoError(t, err)
	gpName := "something"
	msgrs := getMessengers(t, 3)
	members := make([]entity.Contact, 0)
	for _, v := range msgrs {
		self, err := v.IdentityAPI().Get()
		require.NoError(t, err)
		members = append(members, *self.ToContact())
	}
	chat, err := msgrs[0].ChatAPI().New(core.NewChatOpt{Name: gpName, Members: members, Type: entity.Group})
	require.NoError(t, err)
	err = msgrs[0].ChatAPI().Invite(chat.ID,chat.Members)
	require.NoError(t, err)
	time.Sleep(30 * time.Second)
	msg, err := msgrs[0].ChatAPI().Send(chat.ID, "helllooooooo")
	require.NoError(t, err)
	// go func() {
	// 	for {
	// 		for _,v := range msgrs {
	// 			_, err := v.ChatAPI().Send(chat.ID, "helllooooooo")
	// 			require.NoError(t, err)
	// 			time.Sleep(1 * time.Second)
	// 		}

	// 	}

	// }()
	time.Sleep(5 * time.Second)
	for _, v := range msgrs[1:] {
		_, err := v.ChatAPI().Message(msg.ID)
		require.NoError(t, err)
	}

}
