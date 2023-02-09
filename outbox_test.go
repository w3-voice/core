package core

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

type mockExpiry struct {
	ID string
	C  time.Time
}

func (m *mockExpiry) id() string {
	return m.ID
}

func (m *mockExpiry) createdAt() time.Time {
	return m.C
}

func TestOutboxPutAndPop(t *testing.T) {
	// Test values
	key := peer.ID("testKey")
	val1 := &mockExpiry{"val1", time.Now()}
	val2 := &mockExpiry{"val2", time.Now()}
	val3 := &mockExpiry{"val3", time.Now()}

	// Create outbox with a timeout of 1 second and interval of 1 second
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	outbox := NewOutBox(ctx, Config{Keep: false, Timeout: 2 * time.Second, Interval: 1 * time.Second})

	// Put values into outbox
	outbox.Put(key, val1)
	outbox.Put(key, val2)

	// Test if values were put into outbox
	msgs := outbox.Pop(key)
	if len(msgs) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(msgs))
	}
	for _, msg := range msgs {
		switch msg.id() {
		case val1.id():
		case val2.id():
		default:
			t.Errorf("Unexpected message: %v", msg)
		}
	}

	// Put another value and test if it's still empty
	outbox.Put(key, val3)
	failedMsgs := outbox.C()
	<-failedMsgs
	time.Sleep(3 * time.Second)
	msgs = outbox.Pop(key)
	if len(msgs) != 1 {
		t.Errorf("Expected 1 messages, got %d, %s", len(msgs), msgs)
	}

}

func TestOutboxTimeOut(t *testing.T) {
	key := peer.ID("testKey")
	val1 := &mockExpiry{"val1", time.Now()}
	val2 := &mockExpiry{"val2", time.Now()}
	val3 := &mockExpiry{"val3", time.Now()}

	// Create outbox with a timeout of 1 second and interval of 1 second
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	outbox := NewOutBox(ctx, Config{Keep: true, Timeout: 2 * time.Second, Interval: 1 * time.Second})

	// Put values into outbox
	outbox.Put(key, val1)
	outbox.Put(key, val2)

	// Test if values were put into outbox
	msgs := outbox.Pop(key)
	if len(msgs) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(msgs))
	}
	for _, msg := range msgs {
		switch msg.id() {
		case val1.id():
		case val2.id():
		default:
			t.Errorf("Unexpected message: %v", msg)
		}
	}

	// Put another value and test if it's still empty
	outbox.Put(key, val3)
	failedMsgs := outbox.C()
	<-failedMsgs
	time.Sleep(3 * time.Second)
	msgs = outbox.Pop(key)
	if len(msgs) != 1 {
		t.Errorf("Expected 1 messages, got %d, %s", len(msgs), msgs)
	}
}
