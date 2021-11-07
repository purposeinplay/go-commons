package pubsublite

import (
	"context"
	"encoding/json"
	"sync"
)

// Package pubsub provides basic interfaces that make up a PubSub system.
// It also provides an Event type that is used to pass the actual information.
//
// The Event is sent to one or more channels using the Publish method. From where the underlying
// implementation should dispatch the event to all the subscribers of those channels.
//
// Its primary job is to wrap implementations of such PubSub systems,

// Publisher is the interface that wraps the basic Publish method.
type Publisher interface {
	// Publish publishes an event to specified channels.
	Publish(topic string, event *Event) error
	Close() error
}

type Subscriber interface {
	Subscribe(ctx context.Context, topic string) (<-chan *Event, error)
	Close() error
}

// Subscriber is the interface that wraps the Subscribe method.
//type Subscriber interface {
//	// Subscribe creates a new subscription for the events published
//	// in the specified channels.
//	Subscribe(f func(Payload) error) error
//	Close() error
//}

// PublishSubscriber is the interface that groups the basic
// Publish and Subscribe methods.
type PublishSubscriber interface {
	Publisher
	Subscriber
}

type Payload map[string]interface{}

func (p Payload) String() string {
	b, _ := json.Marshal(p)
	return string(b)
}

var closedchan = make(chan struct{})

func init() {
	close(closedchan)
}

// Event represents an event that occurs in the system.
type Event struct {
	// Specifies the type of event that is occurring.
	Type string `json:"type"`

	// ack is closed, when acknowledge is received.
	ack chan struct{}

	// noACk is closed, when negative acknowledge is received.
	noAck chan struct{}

	ackMutex sync.Mutex // era aici, folosit pt close

	ackSentType ackType

	// The actual data from the event.
	Payload Payload `json:"payload"`
}

func (e Event) String() string {
	b, _ := json.Marshal(e)
	return string(b)
}

type ackType int

const (
	noAckSent ackType = iota
	ack
	nack
)

// Ack sends message's acknowledgement.
//
// Ack is not blocking.
// Ack is idempotent.
// False is returned, if Nack is already sent.
func (e *Event) Ack() bool {
	e.ackMutex.Lock()
	defer e.ackMutex.Unlock()

	if e.ackSentType == nack {
		return false
	}
	if e.ackSentType != noAckSent {
		return true
	}

	e.ackSentType = ack

	if e.ack == nil {
		e.ack = closedchan
	} else {
		close(e.ack)
	}

	return true
}

func (e *Event) Acked() <-chan struct{} {
	return e.ack
}

func NewEvent(eventType string) *Event {
	return &Event{
		Type: eventType,
		// UUID:     uuid,
		// Metadata: make(map[string]string),
		// Payload:  payload,
		ack:   make(chan struct{}),
		noAck: make(chan struct{}),
	}
}
