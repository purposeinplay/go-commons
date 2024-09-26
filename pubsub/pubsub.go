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

// Event represents an event that occurs in the system.
type Event[T, P any] struct {
	// Specifies the type of event that is occurring.
	Type T `json:"type"`

	// The actual data from the event.
	Payload P `json:"payload"`

	// Carries an error produced by the underlying subscriber.
	Error error
}
