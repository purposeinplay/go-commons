package kafka_test

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/matryer/is"
	"github.com/purposeinplay/go-commons/pubsub"
	"github.com/purposeinplay/go-commons/pubsub/kafka"
	"go.uber.org/zap"
)

func TestPubSub(t *testing.T) {
	logger := zap.NewExample()

	// nolint: gocritic, revive
	is := is.New(t)

	sarama.DebugLogger = log.New(os.Stdout, "[Sarama] ", log.LstdFlags)

	var (
		username  = os.Getenv("KAFKA_USERNAME")
		password  = os.Getenv("KAFKA_PASSWORD")
		brokerURL = os.Getenv("KAFKA_BROKER_URL")
		topic     = username + ".test"
	)

	suber, err := kafka.NewSubscriber(
		logger,
		kafka.NewSASLSubscriberConfig(
			username,
			password,
		),
		[]string{brokerURL},
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

	mes := pubsub.Event{Payload: []byte("test")}

	err = pub.Publish(mes, topic)
	is.NoErr(err)

	select {
	case receivedMes := <-sub.C():
		is.Equal(receivedMes, mes)

	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}
