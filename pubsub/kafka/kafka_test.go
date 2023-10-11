package kafka_test

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/matryer/is"
	"github.com/purposeinplay/go-commons/kafkadocker"
	"github.com/purposeinplay/go-commons/pubsub"
	"github.com/purposeinplay/go-commons/pubsub/kafka"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestPubSub(t *testing.T) {
	logger := zap.NewExample()

	// nolint: gocritic, revive
	is := is.New(t)

	var (
		username  = os.Getenv("KAFKA_USERNAME")
		password  = os.Getenv("KAFKA_PASSWORD")
		brokerURL = os.Getenv("KAFKA_BROKER_URL")
		topic     = username + ".test"
	)

	suber1, err := kafka.NewSubscriber(
		logger,
		kafka.NewSASLSubscriberConfig(
			username,
			password,
		),
		[]string{brokerURL},
		"",
	)
	is.NoErr(err)

	t.Cleanup(func() { is.NoErr(suber1.Close()) })

	suber2, err := kafka.NewSubscriber(
		logger,
		kafka.NewSASLSubscriberConfig(
			username,
			password,
		),
		[]string{brokerURL},
		"",
	)
	is.NoErr(err)

	t.Cleanup(func() { is.NoErr(suber2.Close()) })

	pub, err := kafka.NewPublisher(
		logger,
		kafka.NewSASLPublisherConfig(
			username,
			password,
		),
		[]string{brokerURL},
	)
	is.NoErr(err)

	t.Cleanup(func() { is.NoErr(pub.Close()) })

	sub1, err := suber1.Subscribe(topic)
	is.NoErr(err)

	t.Cleanup(func() { is.NoErr(sub1.Close()) })

	sub2, err := suber1.Subscribe(topic)
	is.NoErr(err)

	t.Cleanup(func() { is.NoErr(sub2.Close()) })

	mes := pubsub.Event[[]byte]{
		Type:    "test",
		Payload: []byte("test"),
	}

	err = pub.Publish(mes, topic)
	is.NoErr(err)

	var wg sync.WaitGroup

	wg.Add(2)

	now := time.Now()

	go func() {
		defer wg.Done()

		receivedMes := <-sub1.C()
		is.Equal(receivedMes, mes)

		t.Logf("sub1 received the message in %s", time.Since(now))
	}()

	go func() {
		defer wg.Done()

		receivedMes := <-sub2.C()
		is.Equal(receivedMes, mes)

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
	t.Skip("consumer group is failing at the moment")

	logger := zap.NewExample()

	// nolint: gocritic, revive
	is := is.New(t)

	var (
		username  = os.Getenv("KAFKA_USERNAME")
		password  = os.Getenv("KAFKA_PASSWORD")
		brokerURL = os.Getenv("KAFKA_BROKER_URL")
		topic     = username + ".consumer"
	)

	ctx, cancel := context.WithCancel(context.Background())

	consumerGroup := username + "-consumer"

	suber1, err := kafka.NewSubscriber(
		logger,
		kafka.NewSASLSubscriberConfig(
			username,
			password,
		),
		[]string{brokerURL},
		consumerGroup,
	)
	is.NoErr(err)

	t.Cleanup(func() { is.NoErr(suber1.Close()) })

	sub1, err := suber1.Subscribe(topic)
	is.NoErr(err)

	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()

		for {
			select {
			case mes := <-sub1.C():
				t.Logf("sub 1: %s", mes.Payload)
			case <-ctx.Done():
				return
			}
		}
	}()

	suber2, err := kafka.NewSubscriber(
		logger,
		kafka.NewSASLSubscriberConfig(
			username,
			password,
		),
		[]string{brokerURL},
		consumerGroup,
	)
	is.NoErr(err)

	t.Cleanup(func() { is.NoErr(suber2.Close()) })

	sub2, err := suber2.Subscribe(topic)
	is.NoErr(err)

	go func() {
		defer wg.Done()

		for {
			select {
			case mes := <-sub2.C():
				t.Logf("sub 2: %s", mes.Payload)
			case <-ctx.Done():
				return
			}
		}
	}()

	pub, err := kafka.NewPublisher(
		logger,
		kafka.NewSASLPublisherConfig(
			username,
			password,
		),
		[]string{brokerURL},
	)
	is.NoErr(err)

	t.Cleanup(func() { is.NoErr(pub.Close()) })

	err = pub.Publish(pubsub.Event[[]byte]{
		Type:    "",
		Payload: []byte("brad"),
	}, topic)
	is.NoErr(err)

	time.Sleep(5 * time.Second)

	cancel()

	wg.Wait()
}

func TestKafkaServerCloseError(t *testing.T) {
	req := require.New(t)

	ctx := context.Background()

	logger := zap.NewExample()

	cluster := &kafkadocker.Cluster{
		Brokers:     2,
		HealthProbe: true,
		Topics:      []string{"test"},
	}

	err := cluster.Start(ctx)
	req.NoError(err)

	suber, err := kafka.NewSubscriber(logger, nil, cluster.BrokerAddresses(), "")
	req.NoError(err)

	sub, err := suber.Subscribe("test")
	req.NoError(err)

	err = cluster.Stop(ctx)
	req.NoError(err)

	t.Log("stopped cluster")

	select {
	case m := <-sub.C():
		t.Logf("message: %+v", m)
	case <-time.After(10 * time.Second):
		t.Fatal("timeout")
	}
}
