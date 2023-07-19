package kafkadocker_test

import (
	"context"
	"fmt"
	"github.com/IBM/sarama"
	"github.com/purposeinplay/go-commons/kafkadocker"
	"github.com/stretchr/testify/require"
	"testing"
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

	for i, a := range brokerAddresses {
		msg := fmt.Sprintf("index: %d, address: %s", i, a)

		broker := sarama.NewBroker(a)

		err = broker.Open(nil)
		req.NoError(err, msg)

		conn, err := broker.Connected()
		req.NoError(err, msg)
		req.True(conn)

		_, err = broker.Heartbeat(&sarama.HeartbeatRequest{})
		req.NoError(err, msg)
	}
}
