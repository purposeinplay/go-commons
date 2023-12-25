package kafkasarama

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/IBM/sarama"
	"github.com/purposeinplay/go-commons/pubsub"
)

var _ pubsub.Subscriber[[]byte] = (*Subscriber)(nil)

// Subscriber represents a kafka subscriber.
type Subscriber struct {
	logger        *slog.Logger
	cfg           *sarama.Config
	brokers       []string
	consumerGroup string
}

// NewSubscriber creates a new kafka subscriber.
// nolint: revive // allow unused consumerGroup parameter for future proofing.
func NewSubscriber(
	slogHandler slog.Handler,
	saramaConfig *sarama.Config,
	brokers []string,
	consumerGroup string,
) (*Subscriber, error) {
	cfg := saramaConfig

	if cfg == nil {
		cfg = sarama.NewConfig()

		cfg.Consumer.Return.Errors = true
	}

	return &Subscriber{
		logger:        slog.New(slogHandler),
		cfg:           cfg,
		brokers:       brokers,
		consumerGroup: consumerGroup,
	}, nil
}

// Subscribe creates a new subscription that runs in the background.
func (s Subscriber) Subscribe(channels ...string) (pubsub.Subscription[[]byte], error) {
	if len(channels) != 1 {
		return nil, pubsub.ErrExactlyOneChannelAllowed
	}

	sarama.NewConsumerGroup(s.brokers, s.consumerGroup, s.cfg)

	consumer, err := sarama.NewConsumer(s.brokers, s.cfg)
	if err != nil {
		return nil, fmt.Errorf("new sarama consumer: %w", err)
	}

	topic := channels[0]

	return newSubscription(s.logger, consumer, topic)
}

var _ pubsub.Subscription[[]byte] = (*Subscription)(nil)

// Subscription represents a stream of events published to a kafka topic.
type Subscription struct {
	eventCh    chan pubsub.Event[[]byte]
	cancelFunc context.CancelFunc
	wg         *sync.WaitGroup
	consumer   sarama.Consumer
}

// newSubscription creates a new subscription.
// nolint: gocognit
func newSubscription(
	logger *slog.Logger,
	consumer sarama.Consumer,
	topic string,
) (*Subscription, error) {
	partitions, err := consumer.Partitions(topic)
	if err != nil {
		return nil, fmt.Errorf("get topic %q partitions: %w", topic, err)
	}

	eventCh := make(chan pubsub.Event[[]byte])

	ctx, cancel := context.WithCancel(context.Background())

	wg := new(sync.WaitGroup)

	wg.Add(len(partitions))

	for _, partition := range partitions {
		partitionConsumer, err := consumer.ConsumePartition(topic, partition, sarama.OffsetNewest)
		if err != nil {
			cancel()

			return nil, fmt.Errorf("consume partition %d for topic %q: %w", partition, topic, err)
		}

		go func() {
			defer wg.Done()

			// consume partition in the background, stop when the context is
			// cancelled.
			consumePartition(ctx, logger, partitionConsumer, eventCh)
		}()
	}

	return &Subscription{
		eventCh:    eventCh,
		cancelFunc: cancel,
		wg:         wg,
		consumer:   consumer,
	}, nil
}

// nolint: gocognit // allow high cog complexity
func consumePartition(
	ctx context.Context,
	logger *slog.Logger,
	partitionConsumer sarama.PartitionConsumer,
	eventCh chan<- pubsub.Event[[]byte],
) {
	for {
		select {
		case m := <-partitionConsumer.Messages():
			var typ string

			for _, h := range m.Headers {
				if bytes.Equal(h.Key, []byte("type")) {
					typ = string(h.Value)

					break
				}
			}

			eventCh <- pubsub.Event[[]byte]{
				Type:    typ,
				Payload: m.Value,
			}

		case err := <-partitionConsumer.Errors():
			eventCh <- pubsub.Event[[]byte]{
				Type:  pubsub.EventTypeError,
				Error: err,
			}

		case <-ctx.Done():
			if err := partitionConsumer.Close(); err != nil {
				logger.Error(
					"close partition consumer error",
					slog.String("error", err.Error()),
				)
			}

			return
		}
	}
}

// C returns a receive-only go channel of events published.
func (s Subscription) C() <-chan pubsub.Event[[]byte] {
	return s.eventCh
}

// Close closes the subscription.
func (s Subscription) Close() error {
	s.cancelFunc()

	s.wg.Wait()

	close(s.eventCh)

	return s.consumer.Close()
}
