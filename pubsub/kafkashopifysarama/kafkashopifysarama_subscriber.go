package kafkashopifysarama

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/Shopify/sarama"
	"github.com/purposeinplay/go-commons/pubsub"
)

var _ pubsub.Subscriber[[]byte] = (*Subscriber)(nil)

// NewConsumerGroup generates a new kafka consumer to be used by the subscriber,
// allowing for dependency injection for testing with a Sarama mock.
func NewConsumerGroup(
	tlsConfig *tls.Config,
	clientID string,
	sessionTimeoutMS int,
	heartbeatIntervalMS int,
	brokers []string,
	groupID string,
) (sarama.ConsumerGroup, error) {
	kafkaCfg := NewTLSSubscriberConfig(tlsConfig)

	kafkaCfg.ClientID = clientID
	kafkaCfg.Consumer.Offsets.Initial = sarama.OffsetNewest
	kafkaCfg.Consumer.Offsets.AutoCommit.Enable = false
	kafkaCfg.Consumer.Group.Session.Timeout = time.Duration(sessionTimeoutMS) * time.Millisecond
	kafkaCfg.Consumer.Group.Heartbeat.Interval = time.Duration(
		heartbeatIntervalMS,
	) * time.Millisecond

	consumerGroup, err := sarama.NewConsumerGroup(brokers, groupID, kafkaCfg)
	if err != nil {
		return nil, fmt.Errorf("new sarama consumer group: %w", err)
	}

	return consumerGroup, nil
}

// Subscriber represents a kafka subscriber.
type Subscriber struct {
	logger        *slog.Logger
	consumerGroup sarama.ConsumerGroup
}

// NewSubscriber creates a new kafka subscriber.
func NewSubscriber(
	slogHandler slog.Handler,
	consumerGroup sarama.ConsumerGroup,
) (*Subscriber, error) {
	return &Subscriber{
		logger:        slog.New(slogHandler),
		consumerGroup: consumerGroup,
	}, nil
}

// Subscribe creates a new subscription that runs in the background.
func (s Subscriber) Subscribe(channels ...string) (pubsub.Subscription[[]byte], error) {
	return newSubscription(s.logger, s.consumerGroup, channels)
}

var _ pubsub.Subscription[[]byte] = (*Subscription)(nil)

// Subscription represents a stream of events published to a kafka topic.
type Subscription struct {
	logger        *slog.Logger
	consumerGroup sarama.ConsumerGroup
	eventCh       chan pubsub.Event[[]byte]
	cancelFunc    context.CancelFunc
	wg            *sync.WaitGroup
	ready         chan bool
}

func newSubscription(
	logger *slog.Logger,
	consumerGroup sarama.ConsumerGroup,
	topics []string,
) (*Subscription, error) {
	eventCh := make(chan pubsub.Event[[]byte])

	ctx, cancel := context.WithCancel(context.Background())

	wg := new(sync.WaitGroup)

	wg.Add(1)

	sub := &Subscription{
		logger:        logger,
		consumerGroup: consumerGroup,
		eventCh:       eventCh,
		cancelFunc:    cancel,
		wg:            wg,
		ready:         make(chan bool),
	}

	go func() {
		defer wg.Done()

		for {
			if err := consumerGroup.Consume(ctx, topics, sub); err != nil {
				// When setup fails, error will be returned here
				logger.Error("Error from consumer group:", slog.String("error", err.Error()))
				return
			}
			// check if context was cancelled, signaling that the consumer should stop
			if ctx.Err() != nil {
				logger.Info("context cancelled:", slog.String("error", ctx.Err().Error()))
				return
			}

			sub.ready = make(chan bool)
		}
	}()
	<-sub.ready
	logger.Info("Sarama consumer up and running!...")

	return sub, nil
}

// C returns a receive-only go channel of events published.
func (s *Subscription) C() <-chan pubsub.Event[[]byte] {
	return s.eventCh
}

// Close closes the subscription.
func (s *Subscription) Close() error {
	s.cancelFunc()

	s.wg.Wait()

	close(s.eventCh)

	return s.consumerGroup.Close()
}

// Setup is run at the beginning of a new session, before ConsumeClaim.
func (s *Subscription) Setup(session sarama.ConsumerGroupSession) error {
	s.logger.Info("setup")
	s.logger.Info("setup claims:", slog.Any("claims", session.Claims()))
	// Mark the consumer as Ready
	close(s.ready)

	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have
// exited.
func (s *Subscription) Cleanup(sarama.ConsumerGroupSession) error {
	s.logger.Info("cleanup")
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (s *Subscription) ConsumeClaim(
	session sarama.ConsumerGroupSession,
	claim sarama.ConsumerGroupClaim,
) error {
	// NOTE:
	// Do not move the code below to a goroutine.
	// The `ConsumeClaim` itself is called within a goroutine, see:
	// <https://github.com/Shopify/sarama/blob/master/consumer_group.go#L27-L29>
	// Specific consumption news
	for msg := range claim.Messages() {
		var typ string

		for _, h := range msg.Headers {
			if bytes.Equal(h.Key, []byte("type")) {
				typ = string(h.Value)

				break
			}
		}

		// Create a closure for committing the message
		markFunc := func() {
			session.MarkMessage(msg, "")
			session.Commit()
		}

		s.eventCh <- pubsub.Event[[]byte]{
			Type:    typ,
			Payload: msg.Value,
			Ack:     markFunc,
		}
	}

	return nil
}
