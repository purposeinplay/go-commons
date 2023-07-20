package kafkadocker_test

import (
	"context"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/purposeinplay/go-commons/kafkadocker"
	"github.com/stretchr/testify/require"
)

func TestBroker(t *testing.T) {
	t.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")

	req := require.New(t)
	ctx := context.TODO()

	cluster := &kafkadocker.Cluster{
		Brokers:     2,
		HealthProbe: true,
	}

	t.Cleanup(func() {
		err := cluster.Stop(ctx)
		req.NoError(err)
	})

	err := cluster.Start(ctx)
	req.NoError(err)

	brokerAddresses := cluster.BrokerAddresses()

	t.Logf("broker addresses: %s", brokerAddresses)

	cfg := sarama.NewConfig()

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
}
