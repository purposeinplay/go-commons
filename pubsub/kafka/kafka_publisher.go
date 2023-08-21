package kafka

import (
	"fmt"

	"github.com/Shopify/sarama"
	"github.com/ThreeDotsLabs/watermill-kafka/v2/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/purposeinplay/go-commons/pubsub"
	"go.uber.org/zap"
)

var _ pubsub.Publisher[[]byte] = (*Publisher)(nil)

// Publisher represents a kafka publisher.
type Publisher struct {
	kafkaPublisher *kafka.Publisher
}

// NewPublisher creates a new kafka publisher.
func NewPublisher(
		logger *zap.Logger,
		saramaConfig *sarama.Config,
		brokers []string,
) (*Publisher, error) {
	pub, err := kafka.NewPublisher(
		kafka.PublisherConfig{
			Brokers:               brokers,
			Marshaler:             kafka.DefaultMarshaler{},
			OverwriteSaramaConfig: saramaConfig,
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
func (p Publisher) Publish(event pubsub.Event[[]byte], channels ...string) error {
	if len(channels) != 1 {
		return pubsub.ErrExactlyOneChannelAllowed
	}

	mes := message.NewMessage(uuid.New().String(), event.Payload)

	mes.Metadata.Set("type", event.Type)

	if err := p.kafkaPublisher.Publish(
		channels[0],
		mes,
	); err != nil {
		return fmt.Errorf("publish: %w", err)
	}

	return nil
}

// Close closes the kafka publisher.
func (p Publisher) Close() error {
	return p.kafkaPublisher.Close()
}
