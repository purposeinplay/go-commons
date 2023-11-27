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
type Publisher[T any] interface {
	// Publish publishes an event to specified channels.
	Publish(event Event[T], channels ...string) error
}

// Subscriber is the interface that wraps the Subscribe method.
type Subscriber[T any] interface {
	// Subscribe creates a new subscription for the events published
	// in the specified channels.
	Subscribe(channels ...string) (Subscription[T], error)
}

// PublishSubscriber is the interface that groups the basic
// Publish and Subscribe methods.
type PublishSubscriber[T any] interface {
	Publisher[T]
	Subscriber[T]
}

// Subscription represents a stream of events for a single user.
type Subscription[T any] interface {
	// C represents an even stream for all events that are published
	// in the channels that of this Subscription.
	C() <-chan Event[T]

	// Close disconnects the subscription, from the PubSub service.
	// It also closes the event stream channel, thus the subscription
	// will stop receiving any type of Event.
	Close() error
}

// EventTypeError is used as type for an event that carries an error.
var EventTypeError = "error"

// Event represents an event that occurs in the system.
type Event[T any] struct {
	// Specifies the type of event that is occurring.
	Type string `json:"type"`

	// The actual data from the event.
	Payload T `json:"payload"`

	// Carries an error produced by the underlying subscriber.
	Error error

	// Ack should be called to acknowledge the event if the processing
	// of the event was successful.
	Ack func()
}
