package kafkashopifysarama_test

import (
	"crypto/tls"
	"fmt"
	"testing"

	"github.com/Shopify/sarama"
	"github.com/matryer/is"
	"github.com/purposeinplay/go-commons/pubsub"
)

type publisher struct {
	asyncProducer sarama.AsyncProducer
}

type kafkaServer struct {
	publisher *publisher
}

func initialize(tlsCfg *tls.Config, brokers []string) *kafkaServer {
	kafkaCfg := sarama.NewConfig()

	kafkaCfg.Producer.Return.Successes = true
	kafkaCfg.Net.TLS.Enable = true
	kafkaCfg.Net.TLS.Config = tlsCfg

	publisher, err := newPublisher(kafkaCfg, brokers)
	if err != nil {
		panic(err)
	}

	return &kafkaServer{
		publisher: publisher,
	}
}

func newPublisher(
	saramaConfig *sarama.Config,
	brokers []string,
) (*publisher, error) {
	producer, err := sarama.NewAsyncProducer(brokers, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("new kafka publisher: %w", err)
	}

	return &publisher{
		asyncProducer: producer,
	}, nil
}

// Publish publishes an event to a kafka topic.
func (p *publisher) Publish(event pubsub.Event[[]byte], channels ...string) error {
	if len(channels) != 1 {
		return pubsub.ErrExactlyOneChannelAllowed
	}

	topic := channels[0]

	mes := &sarama.ProducerMessage{
		Topic: topic,
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte("type"),
				Value: []byte(event.Type),
			},
		},
		Value: sarama.ByteEncoder(event.Payload),
	}

	p.asyncProducer.Input() <- mes

	return nil
}

// SendMessage sends a message to the Kafka consumer on the given topic.
func (ks *kafkaServer) SendMessage(t *testing.T, topic, msg string) {
	t.Helper()

	i := is.New(t)
	i.Helper()

	err := ks.publisher.Publish(pubsub.Event[[]byte]{
		Type:    "test",
		Payload: []byte(msg),
	}, topic)

	i.NoErr(err)

	t.Logf("Sent message to topic %s: %s", topic, msg)
}

// Close closes the kafka publisher.
func (ks *kafkaServer) Close() error {
	return ks.publisher.asyncProducer.Close()
}
