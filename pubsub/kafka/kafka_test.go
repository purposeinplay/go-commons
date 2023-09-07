package kafka_test

import (
	"log"
	"os"
	"testing"
	"time"

	"context"
	"github.com/Shopify/sarama"
	"github.com/matryer/is"
	"github.com/purposeinplay/go-commons/pubsub"
	"github.com/purposeinplay/go-commons/pubsub/kafka"
	"go.uber.org/zap"
	"sync"
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

	sarama.DebugLogger = log.New(os.Stdout, "[Sarama] ", log.LstdFlags)

	suber, err := kafka.NewSubscriber(
		logger,
		kafka.NewSASLSubscriberConfig(
			username,
			password,
		),
		[]string{brokerURL},
		"",
	)
	is.NoErr(err)

	t.Cleanup(func() { is.NoErr(suber.Close()) })

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

	sub, err := suber.Subscribe(topic)
	is.NoErr(err)

	t.Cleanup(func() { is.NoErr(sub.Close()) })

	mes := pubsub.Event[[]byte]{
		Type:    "test",
		Payload: []byte("test"),
	}

	err = pub.Publish(mes, topic)
	is.NoErr(err)

	select {
	case receivedMes := <-sub.C():
		is.Equal(receivedMes, mes)

	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

func TestConsumerGroups(t *testing.T) {
	logger := zap.NewExample()

	// nolint: gocritic, revive
	is := is.New(t)

	var (
		username  = os.Getenv("KAFKA_USERNAME")
		password  = os.Getenv("KAFKA_PASSWORD")
		brokerURL = os.Getenv("KAFKA_BROKER_URL")
		topic     = username + ".consumer"
	)

	// sarama.DebugLogger = log.New(os.Stdout, "[Sarama] ", log.LstdFlags)

	ctx, cancel := context.WithCancel(context.Background())

	var consumerGroup = username + "-consumer"
	// var consumerGroup = ""

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
