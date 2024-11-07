package kafkadocker_test

import (
	"context"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/purposeinplay/go-commons/kafkadocker"
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
