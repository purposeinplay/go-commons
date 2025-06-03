package kafkasarama

import (
	"fmt"
	"github.com/dnwe/otelsarama"
	"log/slog"
	"time"

	"github.com/IBM/sarama"
	"github.com/purposeinplay/go-commons/pubsub"
)

var _ pubsub.Publisher[string, []byte] = (*Publisher)(nil)

// Publisher represents a kafka publisher.
type Publisher struct {
	logger       *slog.Logger
	syncProducer sarama.SyncProducer
}

// NewPublisher creates a new kafka publisher.
func NewPublisher(
	logger *slog.Logger,
	saramaConfig *sarama.Config,
	brokers []string,
) (*Publisher, error) {
	cfg := saramaConfig

	if cfg == nil {
		cfg = sarama.NewConfig()

		cfg.Producer.Retry.Max = 10
		cfg.Producer.Return.Successes = true
		cfg.Metadata.Retry.Backoff = time.Second * 2
	}

	producer, err := sarama.NewSyncProducer(brokers, cfg)
	if err != nil {
		return nil, fmt.Errorf("new kafka publisher: %w", err)
	}

	p := otelsarama.WrapSyncProducer(cfg, producer)

	return &Publisher{
		logger:       logger.With(slog.String("component", "kafkasarama")),
		syncProducer: p,
	}, nil
}

// Publish publishes an event to a kafka topic.
func (p Publisher) Publish(event pubsub.Event[string, []byte], channels ...string) error {
	if len(channels) != 1 {
		return pubsub.ErrExactlyOneChannelAllowed
	}

	topic := channels[0]

	mes := &sarama.ProducerMessage{
		Topic: topic,
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte("type"),
				Value: []byte(event.Type),
			},
		},
		Value: sarama.ByteEncoder(event.Payload),
	}

	if _, _, err := p.syncProducer.SendMessage(mes); err != nil {
		return fmt.Errorf("publish: %w", err)
	}

	p.logger.Debug("published message", slog.String("topic", topic), slog.String("type", event.Type))

	return nil
}

// Close closes the kafka publisher.
func (p Publisher) Close() error {
	return p.syncProducer.Close()
}
