package amqp

import (
	"fmt"

	"github.com/cenkalti/backoff/v4"
	"github.com/streadway/amqp"
)

type Connection struct {
	amqpConnection *amqp.Connection
}

func NewConnection(opts ...AmqpConfigOption) (*Connection, error) {
	var conn *amqp.Connection

	cfg := &AmqpConfig{
		connection: ConnectionConfig{},
	}

	for _, opt := range opts {
		opt(cfg)
	}

	operation := func() error {
		var err error

		if cfg.connection.amqpConfig != nil {
			conn, err = amqp.DialConfig(cfg.connection.url, *cfg.connection.amqpConfig)
		} else if cfg.connection.tlsConfig != nil {
			conn, err = amqp.DialTLS(cfg.connection.url, cfg.connection.tlsConfig)
		} else {
			conn, err = amqp.Dial(cfg.connection.url)
		}

		if err != nil {
			return fmt.Errorf("could not establish rabbitmq connection: %w", err)
		}

		return nil
	}

	err := backoff.Retry(operation, backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 5))
	if err != nil {
		return nil, err
	}

	return &Connection{
		amqpConnection: conn,
	}, nil
}

func (c *Connection) Connection() *amqp.Connection {
	return c.amqpConnection
}

func (c *Connection) Close() error {
	err := c.amqpConnection.Close()
	if err != nil {
		return err
	}

	return nil
}
