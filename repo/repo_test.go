package repo_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/hood-chat/core/entity"
	"github.com/hood-chat/core/repo"
	"github.com/hood-chat/core/store"
	"github.com/stretchr/testify/require"
)

func TestContact(t *testing.T) {
	s, err := store.NewStore(t.TempDir())
	require.NoError(t, err)
	rc := repo.NewContactRepo(s)

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
		err := rc.Add(val)
		require.NoError(t, err)
	}
	opt := repo.NewOption(0, 50)
	res2, err := rc.GetAll(opt)
	require.NoError(t, err)
	if !reflect.DeepEqual(res2, test_contact) {
		t.Error("in and out are not equal")
	}
}

func TestChat(t *testing.T) {
	s, err := store.NewStore(t.TempDir())
	require.NoError(t, err)

	chrepo := repo.NewChatRepo(s)
	rc := repo.NewContactRepo(s)

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
		err := rc.Add(val)
		require.NoError(t, err)
	}
	chatinfo0 := entity.ChatInfo{
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
		Admins: []entity.Contact{},	
		Type:       entity.Private,
		Unread:     1,
		LatestText: "123 123 345",
	}
	chatinfo1 := entity.ChatInfo{
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
		Type:       entity.Private,
		Admins: []entity.Contact{},	
		Unread:     0,
		LatestText: "234vbxvb cvdfrg ",
	}

	chat0 := []entity.Message{
		{
			ChatID: chatinfo0.ID,
			ID:     "1",
			Author: entity.Contact{
				ID:   "1",
				Name: "blue",
			},
			CreatedAt: time.Now().UTC().Unix(),
			Text:      "asdf cbdgf",
			Status:    entity.Pending,
		},
		{
			ID:     "2",
			ChatID: chatinfo0.ID,
			Author: entity.Contact{
				ID:   "2",
				Name: "red",
			},
			CreatedAt: time.Now().UTC().Unix() + 20,
			Text:      "123 123 345",
			Status:    entity.Received,
		},
	}

	chat1 := []entity.Message{
		{
			ID:     "3",
			ChatID: chatinfo1.ID,
			Author: entity.Contact{
				ID:   "3",
				Name: "blue",
			},
			CreatedAt: time.Now().UTC().Unix() + 20,
			Text:      "234vbxvb cvdfrg ",
			Status:    entity.Pending,
		},
		{
			ID:     "4",
			ChatID: chatinfo1.ID,
			Author: entity.Contact{
				ID:   "1",
				Name: "blue",
			},
			CreatedAt: time.Now().UTC().Unix(),
			Text:      "dsgfcvbr56 etrtert",
			Status:    entity.Pending,
		},
	}

	err = chrepo.Add(chatinfo0)
	require.NoError(t, err)
	rmsg := repo.NewMessageRepo(s)
	rmsg.Add(chat0[0])
	rmsg.Add(chat0[1])

	err = chrepo.Add(chatinfo1)
	require.NoError(t, err)

	rmsg = repo.NewMessageRepo(s)
	rmsg.Add(chat1[0])
	rmsg.Add(chat1[1])
	b := repo.NewOption(0, 50)
	res, err := chrepo.GetAll(b)
	require.NoError(t, err)

	if !reflect.DeepEqual(res, []entity.ChatInfo{chatinfo0, chatinfo1}) {
		t.Error("in and out are not equal")
	}

	t.Logf("result %v", res)

	res2, err := chrepo.GetByID("1")
	require.NoError(t, err)
	if !reflect.DeepEqual(res2, chatinfo0) {
		t.Error("in and out are not equal")
	}
	b.AddFilter("ChatID", string(res2.ID))
	res_msg, err := rmsg.GetAll(b)
	require.NoError(t, err)
	require.Equal(t, len(res_msg), len(chat1))
	res_msg, err = rmsg.GetAll(b)
	require.NoError(t, err)
	require.Equal(t, len(res_msg), len(chat0))

	// Test Status filter
	opt := repo.NewOption(0, 50)
	opt.AddFilter("ChatID", "1")
	opt.AddFilter("Status",[]entity.Status{entity.Received})
	msgs, err := rmsg.GetAll(opt)
	require.NoError(t, err)
	require.Equal(t, 1, len(msgs))

}

func TestIdentity(t *testing.T) {
	s, err := store.NewStore(t.TempDir())
	require.NoError(t, err)

	chrepo := repo.NewIdentityRepo(s)
	require.NoError(t, err)

	_, err = chrepo.Get()
	require.Error(t, err)

	err = chrepo.Put(entity.Identity{ID: "123", Name: "farhoud", PrivKey: "mykey"})
	require.NoError(t, err)

	id, err := chrepo.Get()
	require.NoError(t, err)

	t.Log(id)

}
