package pubsub_test

import (
	"sync"
	"testing"

	"github.com/purposeinplay/go-commons/pubsub"
)

type countingAcker struct {
	mu    sync.Mutex
	acks  int
	nacks int
}

func (c *countingAcker) Ack() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.acks++
}

func (c *countingAcker) Nack() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.nacks++
}

func TestEventAckNackNoOpWhenNoAcker(t *testing.T) {
	t.Parallel()

	evt := pubsub.Event[string, []byte]{Type: "x", Payload: []byte("y")}

	evt.Ack()
	evt.Nack()
}

func TestEventDelegatesToAcker(t *testing.T) {
	t.Parallel()

	a := &countingAcker{}
	evt := pubsub.Event[string, []byte]{Type: "x", Acker: a}

	evt.Ack()
	evt.Ack()
	evt.Nack()

	if a.acks != 2 {
		t.Fatalf("expected 2 acks, got %d", a.acks)
	}

	if a.nacks != 1 {
		t.Fatalf("expected 1 nack, got %d", a.nacks)
	}
}
