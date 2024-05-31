package inmem

import (
	"testing"

	"github.com/matryer/is"
	"github.com/purposeinplay/go-commons/pubsub"
)

func TestPubSub_SubscribeSuccess(t *testing.T) {
	i := is.New(t)

	const (
		eventBufferSize = 1

		channelA = "a"
		channelB = "b"
		channelC = "c"
	)

	ps := NewPubSub[string, string](eventBufferSize)

	subA, err := ps.Subscribe(channelA)
	i.NoErr(err)

	subB, err := ps.Subscribe(channelA, channelB)
	i.NoErr(err)

	subC, err := ps.Subscribe(channelC)
	i.NoErr(err)

	// Publish event for first 2 subscriptions.
	_ = ps.Publish(pubsub.Event[string, string]{Type: "test", Payload: "test"}, channelA)

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
	i := is.New(t)

	const (
		eventBufferSize = 1
		channelA        = "a"
	)

	ps := NewPubSub[string, string](eventBufferSize)

	subscription, err := ps.Subscribe(channelA)
	i.NoErr(err)

	err = ps.Publish(pubsub.Event[string, string]{Type: "test", Payload: "test"}, channelA)
	i.NoErr(err)

	err = subscription.Close()
	i.NoErr(err)

	// Verify event is still received.
	select {
	case <-subscription.C():
	default:
		t.Error("expected event")
	}

	// Ensure channel is closed.
	_, open := <-subscription.C()
	i.True(!open)

	// Ensure unsubscribing twice is ok.
	err = subscription.Close()
	i.NoErr(err)
}
