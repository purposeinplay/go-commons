package amqp

import (
	"fmt"

	"github.com/cenkalti/backoff/v4"
	"github.com/streadway/amqp"
)

type Config struct {
	URL string
}

func NewConnection(cfg *Config) (*amqp.Connection, error) {
	var rabbit *amqp.Connection

	operation := func() error {
		conn, err := amqp.Dial(cfg.URL)
		rabbit = conn

		if err != nil {
			return fmt.Errorf("could not establish rabbitmq connection: %w", err)
		}

		return nil
	}

	err := backoff.Retry(operation, backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 5))
	if err != nil {
		return nil, err
	}

	return rabbit, nil
}
