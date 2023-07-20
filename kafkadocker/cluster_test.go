package kafkadocker_test

import (
	"context"
	"testing"

	"fmt"
	"github.com/IBM/sarama"
	"github.com/avast/retry-go"
	"github.com/purposeinplay/go-commons/kafkadocker"
	"github.com/stretchr/testify/require"
	"time"
)

func TestBroker(t *testing.T) {
	t.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")

	req := require.New(t)
	ctx := context.TODO()

	cluster := &kafkadocker.Cluster{
		Brokers: 2,
	}

	t.Cleanup(func() {
		err := cluster.Stop(ctx)
		req.NoError(err)
	})

	err := cluster.Start(ctx)
	req.NoError(err)

	brokerAddresses := cluster.BrokerAddresses()

	t.Logf("broker addresses: %s", brokerAddresses)

	t.Run("TestBrokers", func(t *testing.T) {
		req := require.New(t)

		// time.Sleep(10 * time.Second)

		for _, addr := range brokerAddresses {
			ctx, cancel := context.WithDeadline(ctx, time.Now().Add(2*time.Minute))

			err = retry.Do(func() error {
				brk := sarama.NewBroker(addr)

				if err = brk.Open(nil); err != nil {
					return fmt.Errorf("open: %w", err)
				}

				defer brk.Close()

				conn, err := brk.Connected()
				if err != nil {
					return fmt.Errorf("connected: %w", err)
				}

				if !conn {
					return fmt.Errorf("not connected")
				}

				if _, err = brk.Heartbeat(&sarama.HeartbeatRequest{}); err != nil {
					return fmt.Errorf("heartbeat: %w", err)
				}

				return nil
			}, retry.Context(ctx))
			req.NoError(err)

			cancel()
		}
	})

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

	partConsumer, err := consumer.ConsumePartition(topic, partition, cfg.Consumer.Offsets.Initial)
	req.NoError(err)

	select {
	case mes := <-partConsumer.Messages():
		t.Logf("message: %s", mes.Value)

	default:
		req.Fail("no message")
	}

	topics, err := client.Topics()
	req.NoError(err)

	req.Equal([]string{topic}, topics)
}
