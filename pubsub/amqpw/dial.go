package amqpw

import (
	"github.com/cenkalti/backoff/v4"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

func Dial(cfg *Config) (*amqp.Connection, error) {
	var rabbit *amqp.Connection

	operation := func() error {
		conn, err := amqp.Dial(cfg.URL)
		rabbit = conn

		if err != nil {
			return errors.Wrap(err, "opening rabbitmq connection")
		}

		return nil
	}

	err := backoff.Retry(operation, backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 5))

	if err != nil {
		return nil, err
	}

	return rabbit, nil
}
