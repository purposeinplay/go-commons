package kafkadocker_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/purposeinplay/go-commons/kafkadocker"
	"github.com/purposeinplay/go-commons/pubsub"
	"github.com/purposeinplay/go-commons/pubsub/kafkasarama"
	"github.com/stretchr/testify/require"
)

func TestKafka(t *testing.T) {
	ctx := context.Background()

	tests := map[string]struct {
		cluster *kafkadocker.Cluster
	}{
		"Zookeeper": {
			cluster: &kafkadocker.Cluster{
				Brokers:     2,
				HealthProbe: true,
				Kraft:       false,
			},
		},
		"Kraft": {
			cluster: &kafkadocker.Cluster{
				Brokers:     1,
				HealthProbe: true,
				Kraft:       true,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req := require.New(t)

			cluster := test.cluster

			t.Logf("starting cluster")

			err := cluster.Start(ctx)
			req.NoError(err)

			t.Logf("cluster started successfully")

			t.Cleanup(func() {
				cluster.Stop(ctx)
			})

			brokerAddresses := cluster.BrokerAddresses()

			t.Logf("broker addresses: %s", brokerAddresses)

			cfg := sarama.NewConfig()

			if test.cluster.Kraft {
				cfg.Net.SASL.Enable = true
				cfg.Net.SASL.Handshake = true
				cfg.Net.SASL.Mechanism = sarama.SASLTypePlaintext
				cfg.Net.SASL.User = "admin"
				cfg.Net.SASL.Version = sarama.SASLHandshakeV1
				cfg.Net.SASL.Password = "admin-secret"
			}

			cfg.Producer.Return.Successes = true

			client, err := sarama.NewClient(brokerAddresses, cfg)
			req.NoError(err)

			t.Cleanup(func() {
				err := client.Close()
				req.NoError(err)
			})

			producer, err := sarama.NewSyncProducerFromClient(client)
			req.NoError(err)

			const topic = "test"

			partition, offset, err := producer.SendMessage(&sarama.ProducerMessage{
				Topic: topic,
				Value: sarama.StringEncoder(topic),
			})
			req.NoError(err)

			t.Logf("message partition: %d, message offset: %d", partition, offset)

			consumer, err := sarama.NewConsumerFromClient(client)
			req.NoError(err)

			partConsumer, err := consumer.ConsumePartition(topic, partition, sarama.OffsetOldest)
			req.NoError(err)

			select {
			case mes := <-partConsumer.Messages():
				t.Logf("message: %+v", mes)
				req.Equal(topic, string(mes.Value))

			case <-time.After(20 * time.Second):
				req.Fail("timeout")
			}

			topics, err := client.Topics()
			req.NoError(err)

			req.Equal([]string{topic}, topics)
		})
	}
}

func TestTopicIsReceivedByGroupedAndStandaloneConsumers(t *testing.T) {
	ctx := context.Background()
	req := require.New(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	cluster := &kafkadocker.Cluster{
		Brokers:     1,
		HealthProbe: true,
		Kraft:       true,
	}

	err := cluster.Start(ctx)
	req.NoError(err)

	t.Cleanup(func() {
		cluster.Stop(ctx)
	})

	brokers := cluster.BrokerAddresses()
	saramaCfg := kafkasarama.NewSASLPlainSubscriberConfig("admin", "admin-secret")

	client, err := sarama.NewClient(brokers, saramaCfg)
	req.NoError(err)

	t.Cleanup(func() {
		err := client.Close()
		if err != nil && !errors.Is(err, sarama.ErrClosedClient) {
			req.NoError(err)
		}
	})

	admin, err := sarama.NewClusterAdminFromClient(client)
	req.NoError(err)

	t.Cleanup(func() {
		err := admin.Close()
		req.NoError(err)
	})

	topic := fmt.Sprintf("test-fanout-%d", time.Now().UnixNano())
	groupID := fmt.Sprintf("test-group-%d", time.Now().UnixNano())

	err = admin.CreateTopic(topic, &sarama.TopicDetail{
		NumPartitions:     1,
		ReplicationFactor: 1,
	}, false)
	req.NoError(err)

	req.Eventually(func() bool {
		if err := client.RefreshMetadata(topic); err != nil {
			return false
		}

		partitions, err := client.Partitions(topic)
		if err != nil {
			return false
		}

		return len(partitions) > 0
	}, 20*time.Second, 200*time.Millisecond)

	standaloneSubscriber, err := kafkasarama.NewSubscriber(
		logger,
		kafkasarama.NewSASLPlainSubscriberConfig("admin", "admin-secret"),
		brokers,
		"",
	)
	req.NoError(err)

	standaloneSub, err := standaloneSubscriber.Subscribe(topic)
	req.NoError(err)

	t.Cleanup(func() {
		err := standaloneSub.Close()
		req.NoError(err)
	})

	groupedSubscriber, err := kafkasarama.NewSubscriber(
		logger,
		kafkasarama.NewSASLPlainSubscriberConfig("admin", "admin-secret"),
		brokers,
		groupID,
	)
	req.NoError(err)

	groupedSub, err := groupedSubscriber.Subscribe(topic)
	req.NoError(err)

	t.Cleanup(func() {
		err := groupedSub.Close()
		req.NoError(err)
	})

	publisher, err := kafkasarama.NewPublisher(
		logger,
		kafkasarama.NewSASLPlainPublisherConfig("admin", "admin-secret"),
		brokers,
	)
	req.NoError(err)

	t.Cleanup(func() {
		err := publisher.Close()
		req.NoError(err)
	})

	event := pubsub.Event[string, []byte]{
		Type:    "fanout-test",
		Payload: []byte(fmt.Sprintf("fanout-message-%d", time.Now().UnixNano())),
	}

	err = publisher.Publish(event, topic)
	req.NoError(err)

	t.Logf("published message to topic %s", topic)

	select {
	case received := <-standaloneSub.C():
		req.NoError(received.Error)
		req.Equal(event.Type, received.Type)
		req.Equal(event.Payload, received.Payload)

		t.Logf("received message in standalone subscriber: %s", received.Payload)

	case <-time.After(20 * time.Second):
		req.FailNow("timeout waiting standalone consumer message")
	}

	select {
	case received := <-groupedSub.C():
		req.NoError(received.Error)
		req.Equal(event.Type, received.Type)
		req.Equal(event.Payload, received.Payload)

		t.Logf("received message in grouped subscriber: %s", received.Payload)

	case <-time.After(20 * time.Second):
		req.FailNow("timeout waiting consumer group message")
	}
}
