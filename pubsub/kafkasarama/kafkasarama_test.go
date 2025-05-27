package kafkasarama_test

import (
	"context"
	"log"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/matryer/is"
	"github.com/purposeinplay/go-commons/pubsub"
	"github.com/purposeinplay/go-commons/pubsub/kafkasarama"
	"go.uber.org/zap"
	"go.uber.org/zap/exp/zapslog"
)

func TestMain(m *testing.M) {
	if err := godotenv.Load(".env.test"); err != nil {
		log.Fatalf("failed to load .env.test file: %v", err)
	}

	m.Run()
}

func TestPubSub(t *testing.T) {
	logger := zap.NewExample()

	// nolint: gocritic, revive
	is := is.New(t)

	slogLogger := slog.New(zapslog.NewHandler(logger.Core(), nil))

	var (
		username  = os.Getenv("KAFKA_USERNAME")
		password  = os.Getenv("KAFKA_PASSWORD")
		brokerURL = os.Getenv("KAFKA_BROKER_URL")
		topic     = os.Getenv("KAFKA_TEST_TOPIC")
	)

	suber1, err := kafkasarama.NewSubscriber(
		slogLogger,
		kafkasarama.NewSASLSubscriberConfig(
			username,
			password,
		),
		[]string{brokerURL},
		"",
	)
	is.NoErr(err)

	pub, err := kafkasarama.NewPublisher(
		slogLogger,
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
	//t.Skip("consumer group is failing at the moment")

	slogLogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	slogLogger.Debug("starting test")

	// nolint: gocritic, revive
	is := is.New(t)

	var (
		username  = os.Getenv("KAFKA_USERNAME")
		password  = os.Getenv("KAFKA_PASSWORD")
		brokerURL = os.Getenv("KAFKA_BROKER_URL")
		topic     = os.Getenv("KAFKA_TEST_TOPIC")
	)

	ctx, cancel := context.WithCancel(context.Background())

	consumerGroup := username + "-consumer"

	suber1, err := kafkasarama.NewSubscriber(
		slogLogger,
		kafkasarama.NewSASLSubscriberConfig(
			username,
			password,
		),
		[]string{brokerURL},
		consumerGroup,
	)
	is.NoErr(err)

	sub1, err := suber1.Subscribe(topic)
	is.NoErr(err)

	t.Cleanup(func() { is.NoErr(sub1.Close()) })

	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()

		for {
			select {
			case mes := <-sub1.C():
				t.Logf("sub 1: %s", mes.Payload)
			case <-ctx.Done():
				t.Log("sub 1: context done")
				return
			}
		}
	}()

	suber2, err := kafkasarama.NewSubscriber(
		slogLogger,
		kafkasarama.NewSASLSubscriberConfig(
			username,
			password,
		),
		[]string{brokerURL},
		consumerGroup,
	)
	is.NoErr(err)

	sub2, err := suber2.Subscribe(topic)
	is.NoErr(err)

	t.Cleanup(func() { is.NoErr(sub2.Close()) })

	go func() {
		defer wg.Done()

		for {
			select {
			case mes := <-sub2.C():
				t.Logf("sub 2: %s", mes.Payload)
			case <-ctx.Done():
				t.Log("sub 2: context done")
				return
			}
		}
	}()

	pub, err := kafkasarama.NewPublisher(
		slogLogger,
		kafkasarama.NewSASLPublisherConfig(
			username,
			password,
		),
		[]string{brokerURL},
	)
	is.NoErr(err)

	t.Cleanup(func() { is.NoErr(pub.Close()) })

	err = pub.Publish(pubsub.Event[string, []byte]{
		Type:    "",
		Payload: []byte("test"),
	}, topic)
	is.NoErr(err)

	t.Logf("published message to topic %s", topic)

	time.Sleep(5 * time.Second)

	cancel()

	wg.Wait()
}
