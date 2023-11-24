package kafkashopifysarama_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/matryer/is"
	"github.com/purposeinplay/go-commons/pubsub/kafkashopifysarama"
	"go.uber.org/zap"
	"go.uber.org/zap/exp/zapslog"
)

func TestConsumerGroups(t *testing.T) {
	// nolint: gocritic, revive
	is := is.New(t)

	logger := zap.NewExample()
	slogHandler := zapslog.NewHandler(logger.Core(), nil)
	topic := "test"

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tlsCfg, err := kafkashopifysarama.LoadTLSConfig(
		"./config/test.crt",
		"./config/test.key",
		"./config/ca_test.crt",
	)
	is.NoErr(err)

	var (
		clientID               = "test_client_id"
		autoCommitIntervalSecs = 10
		sessionTimeoutMS       = 45000
		heartbeatIntervalMS    = 15000
		brokers                = []string{"localhost:9092"}
		groupID                = "test_groupd_id"
	)

	consumerGroup1, err := kafkashopifysarama.NewConsumerGroup(
		tlsCfg,
		clientID,
		autoCommitIntervalSecs,
		sessionTimeoutMS,
		heartbeatIntervalMS,
		brokers,
		groupID,
	)
	is.NoErr(err)

	suber1, err := kafkashopifysarama.NewSubscriber(
		slogHandler,
		consumerGroup1,
	)
	is.NoErr(err)

	consumerGroup2, err := kafkashopifysarama.NewConsumerGroup(
		tlsCfg,
		clientID,
		autoCommitIntervalSecs,
		sessionTimeoutMS,
		heartbeatIntervalMS,
		brokers,
		groupID,
	)
	is.NoErr(err)

	suber2, err := kafkashopifysarama.NewSubscriber(
		slogHandler,
		consumerGroup2,
	)
	is.NoErr(err)

	var wg sync.WaitGroup

	wg.Add(2)

	startSubscriber := func(t *testing.T, subscriber *kafkashopifysarama.Subscriber) {
		t.Helper()

		defer wg.Done()

		//nolint: contextcheck // The subscription is closed in the test
		// cleanup.
		sub, err := subscriber.Subscribe(topic)
		is.NoErr(err)

		for {
			select {
			case mes := <-sub.C():
				is.Equal(mes.Type, "test")
				is.Equal(string(mes.Payload), "test message")
				mes.Ack()

				return
			case <-ctx.Done():
				is.NoErr(ctx.Err())
				return
			}
		}
	}

	// Start both subscribers in separate goroutines
	go startSubscriber(t, suber1)
	go startSubscriber(t, suber2)

	// Initialize Kafka server and send message
	kafkaServer := initialize(tlsCfg, brokers)

	time.Sleep(25 * time.Second)
	kafkaServer.SendMessage(t, "test", "test message")
	kafkaServer.SendMessage(t, "test", "test message")

	t.Cleanup(func() {
		is.NoErr(consumerGroup1.Close())
		is.NoErr(consumerGroup2.Close())
		is.NoErr(kafkaServer.Close())
	})

	// Wait for all goroutines to finish
	wg.Wait()
}
