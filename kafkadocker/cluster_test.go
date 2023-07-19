package kafkadocker_test

import (
	"context"
	"testing"

	"github.com/IBM/sarama"
	"github.com/purposeinplay/go-commons/kafkadocker"
	"github.com/stretchr/testify/require"
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

		for _, addr := range brokerAddresses {
			brk := sarama.NewBroker(addr)

			err = brk.Open(nil)
			req.NoError(err)

			t.Cleanup(func() {
				err := brk.Close()
				req.NoError(err)
			})

			conn, err := brk.Connected()
			req.NoError(err)
			req.True(conn)

			_, err = brk.Heartbeat(&sarama.HeartbeatRequest{})
			req.NoError(err)
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

	_, _, err = producer.SendMessage(&sarama.ProducerMessage{
		Topic: "test",
		Value: sarama.StringEncoder("test"),
	})
	req.NoError(err)

	topics, err := client.Topics()
	req.NoError(err)

	t.Log(topics)
}
