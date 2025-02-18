package kafkasarama_test

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/matryer/is"
	"github.com/purposeinplay/go-commons/pubsub"
	"github.com/purposeinplay/go-commons/pubsub/kafka"
	"github.com/purposeinplay/go-commons/pubsub/kafkasarama"
	"go.uber.org/zap"
	"go.uber.org/zap/exp/zapslog"
)

func TestPubSub(t *testing.T) {
	logger := zap.NewExample()

	// nolint: gocritic, revive
	is := is.New(t)

	slogHandler := zapslog.NewHandler(logger.Core(), nil)

	os.Setenv("KAFKA_USERNAME", "win-kafka")
	os.Setenv("KAFKA_PASSWORD", "WigCHHFjpAOllDj")
	os.Setenv("KAFKA_BROKER_URL", "b-1-public.winkafkacluster.5y0luk.c3.kafka.eu-central-1.amazonaws.com:9196")
	os.Setenv("KAFKA_TEST_TOPIC", "openmatch_complete_tournament")

	var (
		username  = os.Getenv("KAFKA_USERNAME")
		password  = os.Getenv("KAFKA_PASSWORD")
		brokerURL = os.Getenv("KAFKA_BROKER_URL")
		topic     = os.Getenv("KAFKA_TEST_TOPIC")
	)

	suber1, err := kafkasarama.NewSubscriber(
		slogHandler,
		kafkasarama.NewSASLSubscriberConfig(
			username,
			password,
		),
		[]string{brokerURL},
		"",
	)
	is.NoErr(err)

	pub, err := kafkasarama.NewPublisher(
		slogHandler,
		kafkasarama.NewSASLPublisherConfig(
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

	mes := pubsub.Event[string, []byte]{
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

		timeout := time.After(3 * time.Second)

		var count int

	loop:
		for {
			select {
			case receivedMes := <-sub1.C():
				if count > 0 {
					t.Errorf("more than one message received: %+v", mes)
					return
				}

				t.Logf("sub1 received the message: %+v in %s", mes, time.Since(now))

				is.Equal(receivedMes, mes)

				count++

			case <-timeout:
				break loop
			}
		}
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

	err = pub.Publish(pubsub.Event[string, []byte]{
		Type:    "",
		Payload: []byte("brad"),
	}, topic)
	is.NoErr(err)

	time.Sleep(5 * time.Second)

	cancel()

	wg.Wait()
}
