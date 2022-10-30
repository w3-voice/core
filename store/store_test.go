package store_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/hood-chat/core/store"
	"github.com/stretchr/testify/require"
)

func TestContact(t *testing.T) {
	s, err := store.NewStore(t.TempDir())
	require.NoError(t, err)
	data := []store.BHContact{
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
	for _, val := range data {
		err := s.InsertContact(val)
		require.NoError(t, err)
	}

	res, err := s.AllContacts()
	require.NoError(t, err)
	if !reflect.DeepEqual(res, data) {
		t.Error("in and out are not equal")
	}
	t.Log("result ", res)

	res, err = s.ContactByIDs([]string{"1", "2"})
	t.Log("result ", res)
	require.NoError(t, err)
	if !reflect.DeepEqual(res, data[:2]) {
		t.Error("in and out are not equal")
	}
	t.Log("result ", res)

	res2, err := s.ContactByID("2")
	require.NoError(t, err)
	if res2 != data[1] {
		t.Error("in and out are not equal")
	}
	t.Log("result ", res2)

}

func TestChat(t *testing.T) {
	s, err := store.NewStore(t.TempDir())
	require.NoError(t, err)
	test_contact := []store.BHContact{
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
		err := s.InsertContact(val)
		require.NoError(t, err)
	}
	test_chat := []store.BHChat{
		{
			ID:      "1",
			Name:    "blue",
			Members: []string{"1", "2"},
		},
		{
			ID:      "2",
			Name:    "blue",
			Members: []string{"1", "3"},
		},
	}
	for _, val := range test_chat {
		err := s.InsertChat(val)
		require.NoError(t, err)
	}
	test_msg := []store.BHTextMessage{
		{
			ID:     "1",
			ChatID: "1",
			Author: store.BHContact{
				ID:   "1",
				Name: "blue",
			},
			CreatedAt: time.Now().Round(0),
			Text:      "asdf cbdgf",
			Status:    store.Pending,
		},
		{
			ID:     "2",
			ChatID: "1",
			Author: store.BHContact{
				ID:   "2",
				Name: "red",
			},
			CreatedAt: time.Now().Round(0),
			Text:      "123 123 345",
			Status:    store.Pending,
		},
		{
			ID:     "3",
			ChatID: "2",
			Author: store.BHContact{
				ID:   "3",
				Name: "blue",
			},
			CreatedAt: time.Now().Round(0),
			Text:      "asdrytxcv 567567",
			Status:    store.Pending,
		},
		{
			ID:     "4",
			ChatID: "2",
			Author: store.BHContact{
				ID:   "1",
				Name: "blue",
			},
			CreatedAt: time.Now().Round(0),
			Text:      "x.zcvm,dlfkjgerotiu ",
			Status:    store.Pending,
		},
	}
	for _, val := range test_msg {
		err := s.InsertTextMessage(val)
		require.NoError(t, err)
	}

	res, err := s.ChatList()
	require.NoError(t, err)
	if !reflect.DeepEqual(res, test_chat) {
		t.Error("in and out are not equal")
	}

	res2, err := s.ChatByID("1")
	require.NoError(t, err)
	if !reflect.DeepEqual(res2, test_chat[0]) {
		t.Error("in and out are not equal")
	}

	res3, err := s.ChatMessages("1")
	require.NoError(t, err)
	if !reflect.DeepEqual(res3, test_msg[:2]) {
		t.Error("in and out are not equal")
	}
	t.Log("result ", res3, test_msg[:2])

}
