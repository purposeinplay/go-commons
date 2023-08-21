package kafka

import (
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/purposeinplay/go-commons/pubsub"
)

var _ pubsub.Subscription[[]byte] = (*Subscription)(nil)

// Subscription represents a stream of events published to a kafka topic.
type Subscription struct {
	eventCh chan pubsub.Event[[]byte]
	closeCh chan struct{}
}

// newSubscription creates a new subscription.
func newSubscription(mesCh <-chan *message.Message) *Subscription {
	eventCh := make(chan pubsub.Event[[]byte])
	closeCh := make(chan struct{})

	go func() {
		for {
			select {
			case <-closeCh:
				return
			case mes, ok := <-mesCh:
				if !ok {
					return
				}

				eventCh <- pubsub.Event[[]byte]{
					Type:    mes.Metadata.Get("type"),
					Payload: mes.Payload,
				}
			}
		}
	}()

	return &Subscription{
		eventCh: eventCh,
		closeCh: closeCh,
	}
}

// C returns a receive-only go channel of events published.
func (s Subscription) C() <-chan pubsub.Event[[]byte] {
	return s.eventCh
}

// Close closes the subscription.
func (s Subscription) Close() error {
	close(s.eventCh)
	close(s.closeCh)

	return nil
}
