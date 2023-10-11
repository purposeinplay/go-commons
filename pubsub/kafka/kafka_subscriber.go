package kafka

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Shopify/sarama"
	"github.com/ThreeDotsLabs/watermill-kafka/v2/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/purposeinplay/go-commons/pubsub"
	"go.uber.org/zap"
)

var _ pubsub.Subscriber[[]byte] = (*Subscriber)(nil)

// Subscriber represents a kafka subscriber.
type Subscriber struct {
	kafkaSubscriber *kafka.Subscriber
}

// NewSubscriber creates a new kafka subscriber.
func NewSubscriber(
	logger *zap.Logger,
	saramaConfig *sarama.Config,
	brokers []string,
	consumerGroup string,
) (*Subscriber, error) {
	sub, err := kafka.NewSubscriber(
		kafka.SubscriberConfig{
			Brokers:               brokers,
			Unmarshaler:           kafka.DefaultMarshaler{},
			OverwriteSaramaConfig: saramaConfig,
			ConsumerGroup:         consumerGroup,
		},
		newLoggerAdapter(logger),
	)
	if err != nil {
		return nil, fmt.Errorf("new kafka subscriber: %w", err)
	}

	return &Subscriber{
		kafkaSubscriber: sub,
	}, nil
}

// Subscribe subscribes to a kafka topic.
func (s Subscriber) Subscribe(channels ...string) (pubsub.Subscription[[]byte], error) {
	if len(channels) != 1 {
		return nil, pubsub.ErrExactlyOneChannelAllowed
	}

	mes, err := s.kafkaSubscriber.Subscribe(context.Background(), channels[0])
	if err != nil {
		return nil, fmt.Errorf("subscribe: %w", err)
	}

	return newSubscription(mes), nil
}

// Close closes the kafka subscriber.
func (s Subscriber) Close() error {
	return s.kafkaSubscriber.Close()
}

var _ pubsub.Subscription[[]byte] = (*Subscription)(nil)

// Subscription represents a stream of events published to a kafka topic.
type Subscription struct {
	eventCh chan pubsub.Event[[]byte]
	closeCh chan struct{}
}

// newSubscription creates a new subscription.
// nolint: gocognit
func newSubscription(
	mesCh <-chan *message.Message,
) *Subscription {
	eventCh := make(chan pubsub.Event[[]byte])
	closeCh := make(chan struct{})

	go func() {
		for {
			select {
			case <-closeCh:
				return
			case mes, ok := <-mesCh:
				if !ok {
					slog.Info("sub closed")
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
