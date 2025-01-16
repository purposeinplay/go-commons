package kafkashopifysarama_test

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/matryer/is"
	"github.com/purposeinplay/go-commons/pubsub"
	"github.com/purposeinplay/go-commons/pubsub/kafkashopifysarama"
	"go.uber.org/zap"
	"go.uber.org/zap/exp/zapslog"
)

func TestPubSub(t *testing.T) {
	logger := zap.NewExample()

	i := is.New(t)

	slogHandler := zapslog.NewHandler(logger.Core(), nil)

	var (
		clientID            = "test_client_id"
		initialOffset       = sarama.OffsetNewest
		autoCommit          = true
		sessionTimeoutMS    = 45000
		heartbeatIntervalMS = 15000
		brokers             = []string{os.Getenv("KAFKA_BROKER_URL")}
		groupID             = "test_groupd_id"
		topic               = "test_topic"
	)

	cfg1, err := kafkashopifysarama.NewTLSSubscriberConfig(
		"./config/test.crt",
		"./config/test.key",
		"./config/ca_test.crt",
	)
	i.NoErr(err)

	suber1, err := kafkashopifysarama.NewSubscriber(
		slogHandler,
		cfg1,
		clientID,
		initialOffset,
		autoCommit,
		sessionTimeoutMS,
		heartbeatIntervalMS,
		brokers,
		groupID,
	)
	i.NoErr(err)

	pubCfg, err := kafkashopifysarama.NewTLSPublisherConfig(
		"./config/test.crt",
		"./config/test.key",
		"./config/ca_test.crt",
	)
	i.NoErr(err)

	pub, err := kafkashopifysarama.NewPublisher(
		slogHandler,
		pubCfg,
		brokers,
	)
	i.NoErr(err)

	t.Cleanup(func() { i.NoErr(pub.Close()) })

	sub1, err := suber1.Subscribe(topic)
	i.NoErr(err)

	t.Cleanup(func() { i.NoErr(sub1.Close()) })

	sub2, err := suber1.Subscribe(topic)
	i.NoErr(err)

	t.Cleanup(func() { i.NoErr(sub2.Close()) })

	mes := pubsub.Event[string, []byte]{
		Type:    "test",
		Payload: []byte("test_payload"),
	}

	err = pub.Publish(mes, topic)
	i.NoErr(err)

	var wg sync.WaitGroup

	wg.Add(2)

	now := time.Now()

	go func() {
		defer wg.Done()

		receivedMes := <-sub1.C()
		i.Equal(receivedMes, mes)

		t.Logf("sub1 received the message in %s", time.Since(now))
	}()

	go func() {
		defer wg.Done()

		receivedMes := <-sub2.C()
		i.Equal(receivedMes, mes)

		t.Logf("sub2 received the message in %s", time.Since(now))
	}()

	done := make(chan struct{})

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

func TestConsumerGroups(t *testing.T) {
	logger := zap.NewExample()

	i := is.New(t)

	slogHandler := zapslog.NewHandler(logger.Core(), nil)

	var (
		clientID            = "test_client_id"
		initialOffset       = sarama.OffsetNewest
		autoCommit          = true
		sessionTimeoutMS    = 45000
		heartbeatIntervalMS = 15000
		brokers             = []string{os.Getenv("KAFKA_BROKER_URL")}
		groupID             = "test_groupd_id"
		topic               = "test_topic"
	)

	ctx, cancel := context.WithCancel(context.Background())

	cfg1, err := kafkashopifysarama.NewTLSSubscriberConfig(
		"./config/test.crt",
		"./config/test.key",
		"./config/ca_test.crt",
	)
	i.NoErr(err)

	suber1, err := kafkashopifysarama.NewSubscriber(
		slogHandler,
		cfg1,
		clientID,
		initialOffset,
		autoCommit,
		sessionTimeoutMS,
		heartbeatIntervalMS,
		brokers,
		groupID,
	)
	i.NoErr(err)

	mes := pubsub.Event[string, []byte]{
		Type:    "test",
		Payload: []byte("test_payload"),
	}

	sub1, err := suber1.Subscribe(topic)
	i.NoErr(err)

	t.Cleanup(func() { i.NoErr(sub1.Close()) })

	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()

		for {
			select {
			case receivedMes := <-sub1.C():
				i.Equal(receivedMes, mes)

				t.Logf("sub 1: %s", mes.Payload)
			case <-ctx.Done():
				return
			}
		}
	}()

	cfg2, err := kafkashopifysarama.NewTLSSubscriberConfig(
		"./config/test.crt",
		"./config/test.key",
		"./config/ca_test.crt",
	)
	i.NoErr(err)

	suber2, err := kafkashopifysarama.NewSubscriber(
		slogHandler,
		cfg2,
		clientID,
		initialOffset,
		autoCommit,
		sessionTimeoutMS,
		heartbeatIntervalMS,
		brokers,
		groupID,
	)
	i.NoErr(err)

	sub2, err := suber2.Subscribe(topic)
	i.NoErr(err)

	t.Cleanup(func() { i.NoErr(sub2.Close()) })

	go func() {
		defer wg.Done()

		for {
			select {
			case receivedMes := <-sub2.C():
				i.Equal(receivedMes, mes)

				t.Logf("sub 2: %s", mes.Payload)
			case <-ctx.Done():
				return
			}
		}
	}()

	pubCfg, err := kafkashopifysarama.NewTLSPublisherConfig(
		"./config/test.crt",
		"./config/test.key",
		"./config/ca_test.crt",
	)
	i.NoErr(err)

	pub, err := kafkashopifysarama.NewPublisher(
		slogHandler,
		pubCfg,
		brokers,
	)
	i.NoErr(err)

	t.Cleanup(func() { i.NoErr(pub.Close()) })

	err = pub.Publish(mes, topic)
	i.NoErr(err)

	time.Sleep(5 * time.Second)

	cancel()

	wg.Wait()
}
