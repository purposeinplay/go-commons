package kafka

import (
	"fmt"

	"github.com/ThreeDotsLabs/watermill-kafka/v2/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/purposeinplay/go-commons/pubsub"
	"go.uber.org/zap"
)

var _ pubsub.Publisher = (*Publisher)(nil)

// Publisher represents a kafka publisher.
type Publisher struct {
	kafkaPublisher *kafka.Publisher
}

// NewPublisher creates a new kafka publisher.
func NewPublisher(logger *zap.Logger, brokers []string) (*Publisher, error) {
	pub, err := kafka.NewPublisher(
		kafka.PublisherConfig{
			Brokers:   brokers,
			Marshaler: kafka.DefaultMarshaler{},
		},
		newLoggerAdapter(logger),
	)
	if err != nil {
		return nil, fmt.Errorf("new kafka publisher: %w", err)
	}

	return &Publisher{
		kafkaPublisher: pub,
	}, nil
}

// Publish publishes an event to a kafka topic.
func (p Publisher) Publish(event pubsub.Event, channels ...string) error {
	if len(channels) != 1 {
		return pubsub.ErrExactlyOneChannelAllowed
	}

	payload, ok := event.Payload.([]byte)
	if !ok {
		return pubsub.ErrEventPayloadMustBeByteSlice
	}

	if err := p.kafkaPublisher.Publish(
		channels[0],
		message.NewMessage(uuid.New().String(), payload),
	); err != nil {
		return fmt.Errorf("publish: %w", err)
	}

	return nil
}

// Close closes the kafka publisher.
func (p Publisher) Close() error {
	return p.kafkaPublisher.Close()
}
