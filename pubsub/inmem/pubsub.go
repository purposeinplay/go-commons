// Package inmem defines implementations for the PublisherSubscriber
// interface defined at ./go-commons/pubsub/pubsub.go
// using an in memory storage.
package inmem

import (
	"errors"
	"sync"

	"github.com/purposeinplay/go-commons/pubsub"
)

// Ensure type inmem.PubSub implements interface pubsub.PublishSubscriber.
var _ pubsub.PublishSubscriber[any] = (*PubSub[any])(nil)

// PubSub represents a PubSub backed my an in memory storage.
type PubSub[T any] struct {
	mu sync.Mutex

	// map having channels as keys and subscriptions as value
	channelsSubs map[string]map[*Subscription[T]]struct{}

	// eventBufferSize is the buffer size of the channel for each subscription.
	eventBufferSize int
}

// NewPubSub returns a new instance of PubSub backed
// by an in memory storage.
func NewPubSub[T any](eventBufferSize int) *PubSub[T] {
	return &PubSub[T]{
		channelsSubs:    make(map[string]map[*Subscription[T]]struct{}),
		eventBufferSize: eventBufferSize,
	}
}

// Publish publishes event to all the subscriptions of the channels provided.
func (ps *PubSub[T]) Publish(event pubsub.Event[T], channels ...string) error {
	// Ensure at least one channel is provided.
	if len(channels) == 0 {
		return ErrNoChannel
	}

	ps.mu.Lock()
	defer ps.mu.Unlock()

	// Iterate over the provided channels.
	for _, channel := range channels {
		// Retrieve the subs map for the channel.
		// If there are no subs for the current channel
		// proceed to the next channel.
		subs := ps.channelsSubs[channel]
		if len(subs) == 0 {
			continue
		}

		// Iterate over the subscriptions for the current channel.
		for sub := range subs {
			select {
			// Send the event to the subscriptions go channel.
			case sub.c <- event:

			// In case no one listens to the subscriptions channel
			// remove the subscription.
			default:
				ps.removeSubscription(sub)
			}
		}
	}

	return nil
}

// ErrNoChannel is returned when no channels are passed
// to the Subscribe method.
var ErrNoChannel = errors.New("no channel given")

// Subscribe creates a new subscription for the provided channels.
func (ps *PubSub[T]) Subscribe(channels ...string) (pubsub.Subscription[T], error) {
	// Ensure at least one channel is provided.
	if len(channels) == 0 {
		return nil, ErrNoChannel
	}

	// Create a new subscription.
	sub := &Subscription[T]{
		channels: channels,
		c:        make(chan pubsub.Event[T], ps.eventBufferSize),
		pubsub:   ps,
	}

	ps.mu.Lock()
	defer ps.mu.Unlock()

	// Iterate over the provided channels.
	for _, c := range channels {
		// Retrieve the subs map of the channel.
		subs, ok := ps.channelsSubs[c]
		if !ok {
			// Create the subs map if it does not exist.
			subs = make(map[*Subscription[T]]struct{})
			ps.channelsSubs[c] = subs
		}

		// Add the sub to the subs map of the channel.
		subs[sub] = struct{}{}
	}

	return sub, nil
}

// Unsubscribe removes a sub from the service
// The purpose of this method is to provide a way
// for a Subscription to remove itself from the system.
//
// This method wraps the removeSubscription method
// with the mutexes. So it's safe to be from external
// entities.
func (ps *PubSub[T]) Unsubscribe(sub *Subscription[T]) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ps.removeSubscription(sub)
}

// removeSubscription closes the subscriptions go channel and
// removes it from the pubsubs storage.
func (ps *PubSub[T]) removeSubscription(sub *Subscription[T]) {
	// Only close the underlying channel once.
	sub.once.Do(func() {
		close(sub.c)
	})

	// iterate over the subscriptions channels
	for _, channel := range sub.channels {
		// Retrieve the channels subscriptions.
		// If there are no subscriptions
		// proceed to the next channel.
		subs, ok := ps.channelsSubs[channel]
		if !ok {
			continue
		}

		// Remove the subscription from the channels
		// subscriptions map.
		delete(subs, sub)

		// Remove the channel if there are no
		// subscriptions left.
		if len(subs) == 0 {
			delete(ps.channelsSubs, channel)
		}
	}
}
