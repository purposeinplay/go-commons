package inmem

import (
	"testing"

	"github.com/purposeinplay/go-commons/pubsub"
	"github.com/stretchr/testify/require"
)

func TestPubSub_SubscribeSuccess(t *testing.T) {
	ps := NewPubSub(1)

	subA, err := ps.Subscribe("a")
	require.NoError(t, err)

	subB, err := ps.Subscribe("a", "b")
	require.NoError(t, err)

	subC, err := ps.Subscribe("c")
	require.NoError(t, err)

	// Publish event for first 2 subscriptions.
	ps.Publish(pubsub.Event{Type: "test"}, "a")

	select {
	case <-subA.C():
	default:
		t.Error("expected event on subA")
	}

	select {
	case <-subB.C():
	default:
		t.Error("expected event on subB")
	}

	// Ensure third subscription did not receive event.
	select {
	case <-subC.C():
		t.Error("expected no event on subC")
	default:
	}
}

func TestPubSub_UnsubscribeSuccess(t *testing.T) {
	ps := NewPubSub(1)

	s, err := ps.Subscribe("a")
	require.NoError(t, err)

	err = ps.Publish(pubsub.Event{Type: "test"}, "a")
	require.NoError(t, err)

	err = s.Close()
	require.NoError(t, err)

	// Verify event is still received.
	select {
	case <-s.C():
	default:
		t.Error("expected event")
	}

	// Ensure channel is closed.
	_, open := <-s.C()
	require.False(t, open)

	// Ensure unsubscribing twice is ok.
	err = s.Close()
	require.NoError(t, err)
}
