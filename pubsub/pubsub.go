// Package pubsub provides basic interfaces that make up a PubSub system.
// It also provides an Event type that is used to pass the actual information.
//
// The Event is sent to one or more channels using the Publish method.
// From where the underlying implementation should dispatch the event
// to all the subscribers of those channels.
//
// Its primary job is to wrap implementations of such PubSub systems,
package pubsub

// Publisher is the interface that wraps the basic Publish method.
type Publisher[T, P any] interface {
	// Publish publishes an event to specified channels.
	Publish(event Event[T, P], channels ...string) error
}

// Subscriber is the interface that wraps the Subscribe method.
type Subscriber[T, P any] interface {
	// Subscribe creates a new subscription for the events published
	// in the specified channels.
	Subscribe(channels ...string) (Subscription[T, P], error)
}

// PublishSubscriber is the interface that groups the basic
// Publish and Subscribe methods.
type PublishSubscriber[T, P any] interface {
	Publisher[T, P]
	Subscriber[T, P]
}

// Subscription represents a stream of events for a single user.
type Subscription[T, P any] interface {
	// C represents an even stream for all events that are published
	// in the channels that of this Subscription.
	C() <-chan Event[T, P]

	// Close disconnects the subscription, from the PubSub service.
	// It also closes the event stream channel, thus the subscription
	// will stop receiving any type of Event.
	Close() error
}

// EventTypeError is used as type for an event that carries an error.
var EventTypeError = "error"

// Acker is implemented by backends that support per-message acknowledgement.
// Implementations must make Ack and Nack idempotent — only the first call
// (of either) has an effect.
type Acker interface {
	Ack()
	Nack()
}

// Event represents an event that occurs in the system.
type Event[T, P any] struct {
	// Specifies the type of event that is occurring.
	Type T `json:"type"`

	// The actual data from the event.
	Payload P `json:"payload"`

	// Carries an error produced by the underlying subscriber.
	Error error `json:"-"`

	// Acker is set by ack-aware backends. Nil for backends that don't
	// support ack/nack — Event.Ack and Event.Nack are no-ops in that case.
	Acker Acker `json:"-"`
}

// Ack acknowledges the event. No-op when the backend does not support acks.
func (e Event[T, P]) Ack() {
	if e.Acker != nil {
		e.Acker.Ack()
	}
}

// Nack signals that the event was not processed successfully. No-op when
// the backend does not support nacks.
func (e Event[T, P]) Nack() {
	if e.Acker != nil {
		e.Acker.Nack()
	}
}
