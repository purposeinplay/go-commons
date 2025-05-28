package kafkasarama

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/IBM/sarama"
	"github.com/purposeinplay/go-commons/pubsub"
	"log/slog"
	"sync"
)

var _ pubsub.Subscriber[string, []byte] = (*Subscriber)(nil)

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
	logger *slog.Logger,
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
		logger:        logger.With(slog.String("component", "kafkasarama")),
		cfg:           cfg,
		brokers:       brokers,
		consumerGroup: consumerGroup,
	}, nil
}

// Subscribe creates a new subscription that runs in the background.
func (s Subscriber) Subscribe(channels ...string) (pubsub.Subscription[string, []byte], error) {
	if len(channels) != 1 {
		return nil, pubsub.ErrExactlyOneChannelAllowed
	}

	topic := channels[0]

	logger := s.logger.With(slog.String("topic", topic))

	switch s.consumerGroup {
	case "":
		consumer, err := sarama.NewConsumer(s.brokers, s.cfg)
		if err != nil {
			return nil, fmt.Errorf("new sarama consumer: %w", err)
		}

		return newConsumerSubscription(logger, consumer, topic)

	default:
		consumerGroup, err := sarama.NewConsumerGroup(s.brokers, s.consumerGroup, s.cfg)
		if err != nil {
			return nil, fmt.Errorf("new sarama consumer group: %w", err)
		}

		return newConsumerGroupSubscription(
			logger.With(slog.String("consumer_group", s.consumerGroup)),
			consumerGroup,
			topic,
		)
	}
}

var _ pubsub.Subscription[string, []byte] = (*Subscription)(nil)

// Subscription represents a stream of events published to a kafka topic.
type Subscription struct {
	eventCh       chan pubsub.Event[string, []byte]
	cancelFunc    context.CancelFunc
	wg            *sync.WaitGroup
	consumer      sarama.Consumer
	consumerGroup sarama.ConsumerGroup
}

// newConsumerSubscription creates a new subscription.
// nolint: gocognit
func newConsumerSubscription(
	logger *slog.Logger,
	consumer sarama.Consumer,
	topic string,
) (*Subscription, error) {
	partitions, err := consumer.Partitions(topic)
	if err != nil {
		return nil, fmt.Errorf("get topic %q partitions: %w", topic, err)
	}

	eventCh := make(chan pubsub.Event[string, []byte])

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
	eventCh chan<- pubsub.Event[string, []byte],
) {
	for {
		select {
		case m := <-partitionConsumer.Messages():
			processMessage(m, eventCh)

		case err := <-partitionConsumer.Errors():
			eventCh <- pubsub.Event[string, []byte]{
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

func processMessage(
	m *sarama.ConsumerMessage,
	eventCh chan<- pubsub.Event[string, []byte],
) {
	var typ string

	for _, h := range m.Headers {
		if bytes.Equal(h.Key, []byte("type")) {
			typ = string(h.Value)

			break
		}
	}

	eventCh <- pubsub.Event[string, []byte]{
		Type:    typ,
		Payload: m.Value,
	}
}

// C returns a receive-only go channel of events published.
func (s Subscription) C() <-chan pubsub.Event[string, []byte] {
	return s.eventCh
}

// Close closes the subscription.
func (s Subscription) Close() error {
	s.cancelFunc()

	s.wg.Wait()

	close(s.eventCh)

	if s.consumer != nil {
		return s.consumer.Close()
	}

	if s.consumerGroup != nil {
		return s.consumerGroup.Close()
	}

	return nil
}

func newConsumerGroupSubscription(
	logger *slog.Logger,
	consumerGroup sarama.ConsumerGroup,
	topic string,
) (*Subscription, error) {
	eventCh := make(chan pubsub.Event[string, []byte])

	ctx, cancel := context.WithCancel(context.Background())

	wg := new(sync.WaitGroup)

	wg.Add(1)

	consumer := &consumerGroupHandler{
		logger:  logger.With(slog.String("component", "kafkasarama.consumer_group_handler")),
		eventCh: eventCh,
		ready:   make(chan struct{}),
	}

	go func() {
		defer logger.Debug("sarama consumer group closed")
		defer wg.Done()

		// `Consume` should be called inside an infinite loop, when a
		// server-side rebalance happens, the consumer session will need to be
		// recreated to get the new claims
		for {
			if err := consumerGroup.Consume(ctx, []string{topic}, consumer); err != nil {
				if errors.Is(err, sarama.ErrClosedConsumerGroup) {
					return
				}

				logger.Error("unexpected consume error", slog.String("error", err.Error()))
			}

			// check if context was cancelled, signaling that the consumer should stop
			if ctx.Err() != nil {
				return
			}

			consumer.ready = make(chan struct{})
		}
	}()

	<-consumer.ready

	logger.Debug("sarama consumer up and running")

	return &Subscription{
		eventCh:       eventCh,
		cancelFunc:    cancel,
		wg:            wg,
		consumer:      nil,
		consumerGroup: consumerGroup,
	}, nil
}

type consumerGroupHandler struct {
	logger  *slog.Logger
	eventCh chan<- pubsub.Event[string, []byte]
	ready   chan struct{}
}

func (h consumerGroupHandler) Setup(_ sarama.ConsumerGroupSession) error {
	close(h.ready)

	return nil
}

func (consumerGroupHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }

func (h consumerGroupHandler) ConsumeClaim(
	session sarama.ConsumerGroupSession,
	claim sarama.ConsumerGroupClaim,
) error {
	// NOTE:
	// Do not move the code below to a goroutine.
	// The `ConsumeClaim` itself is called within a goroutine, see:
	// https://github.com/IBM/sarama/blob/main/consumer_group.go#L27-L29
	for {
		select {
		case message, ok := <-claim.Messages():
			if !ok {
				h.logger.Debug("consumer group message channel closed")
				return nil
			}

			processMessage(message, h.eventCh)

			h.logger.Debug(
				"message claimed",
				slog.String("value", string(message.Value)),
				slog.Time("timestamp", message.Timestamp),
				slog.String("topic", message.Topic),
			)

			session.MarkMessage(message, "")
		// Should return when `session.Context()` is done.
		// If not, will raise `ErrRebalanceInProgress` or `read tcp <ip>:<port>: i/o timeout` when kafka rebalance. see:
		// https://github.com/IBM/sarama/issues/1192
		case <-session.Context().Done():
			return nil
		}
	}
}
