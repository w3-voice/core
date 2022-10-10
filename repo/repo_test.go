package repo_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/bee-messenger/core/entity"
	"github.com/bee-messenger/core/repo"
	"github.com/bee-messenger/core/store"
	"github.com/stretchr/testify/require"
)

func TestChat(t *testing.T) {
	s, err := store.NewStore(t.TempDir())
	require.NoError(t, err)

	chrepo, err := repo.NewChatRepo(*s)
	require.NoError(t, err)

	test_contact := []entity.Contact{
		{
			ID:   "1",
			Name: "blue",
		},
		{
			ID:   "2",
			Name: "red",
		},
		{
			ID:   "3",
			Name: "blue",
		},
		{
			ID:   "4",
			Name: "blue",
		},
	}
	for _, val := range test_contact {
		err := chrepo.AddContact(val)
		require.NoError(t, err)
	}
	test_chat := []entity.Chat{
		{
			Info: entity.ChatInfo{
				ID:   "1",
				Name: "blue",
				Members: []entity.Contact{
					{
						ID:   "1",
						Name: "blue",
					},
					{
						ID:   "2",
						Name: "red",
					}},
			},
			Messages: []entity.Message{
				{
					ID: "1",
					Author: entity.Contact{
						ID:   "1",
						Name: "blue",
					},
					CreatedAt: time.Now().Round(0),
					Text:      "asdf cbdgf",
					Status:    entity.Pending,
				},
				{
					ID: "2",
					Author: entity.Contact{
						ID:   "2",
						Name: "red",
					},
					CreatedAt: time.Now().Round(0),
					Text:      "123 123 345",
					Status:    entity.Pending,
				},
			},
		},
		{
			Info: entity.ChatInfo{
				ID:   "2",
				Name: "blue",
				Members: []entity.Contact{
					{
						ID:   "1",
						Name: "blue",
					},
					{
						ID:   "3",
						Name: "blue",
					}},
			},
			Messages: []entity.Message{
				{
					ID: "3",
					Author: entity.Contact{
						ID:   "3",
						Name: "blue",
					},
					CreatedAt: time.Now().Round(0),
					Text:      "234vbxvb cvdfrg ",
					Status:    entity.Pending,
				},
				{
					ID: "4",
					Author: entity.Contact{
						ID:   "1",
						Name: "blue",
					},
					CreatedAt: time.Now().Round(0),
					Text:      "dsgfcvbr56 etrtert",
					Status:    entity.Pending,
				},
			},
		},
	}
	chat0 := test_chat[0]
	err = chrepo.CreateChat(chat0.Info)
	require.NoError(t, err)

	chrepo.AddMessage(chat0.Info.ID, chat0.Messages[0])
	chrepo.AddMessage(chat0.Info.ID, chat0.Messages[1])

	chat1 := test_chat[1]
	err = chrepo.CreateChat(chat1.Info)
	require.NoError(t, err)

	chrepo.AddMessage(chat1.Info.ID, chat1.Messages[0])
	chrepo.AddMessage(chat1.Info.ID, chat1.Messages[1])

	res, err := chrepo.GetAllChat()
	require.NoError(t, err)

	if !reflect.DeepEqual(res, []entity.ChatInfo{test_chat[0].Info, test_chat[1].Info}) {
		t.Error("in and out are not equal")
	}

	t.Logf("result %v", res)
	t.Logf("expected %v", test_chat)

	res2, err := chrepo.GetByIDChat("1")
	require.NoError(t, err)
	if !reflect.DeepEqual(*res2, test_chat[0]) {
		t.Error("in and out are not equal")
	}
	t.Logf("result %v", res2)
	t.Logf("expected %v", test_chat[0])
}
