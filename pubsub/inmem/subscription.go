package inmem

import (
	"sync"

	"github.com/purposeinplay/go-commons/pubsub"
)

// Ensure type inmem.Subscription implements interface pubsub.Subscription.
var _ pubsub.Subscription[any] = (*Subscription[any])(nil)

// Subscription represents a stream of events published to the channels
// of this subscription.
type Subscription[T any] struct {
	// Channels this subscription is subscribed to.
	channels []string

	// Ensures c only closed once
	once sync.Once
	// Channel of events
	c chan pubsub.Event[T]

	pubsub *PubSub[T]
}

// Close disconnects the subscription from the service it was created from.
func (s *Subscription[T]) Close() error {
	s.pubsub.Unsubscribe(s)
	return nil
}

// C returns a receive-only go channel of events published
// on the channels this subscription is subscribed to.
func (s *Subscription[T]) C() <-chan pubsub.Event[T] {
	return s.c
}
