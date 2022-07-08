package amqp

import (
	"fmt"

	"github.com/purposeinplay/go-commons/pubsublite"
	"github.com/streadway/amqp"
)

var _ pubsublite.Publisher = &Publisher{}

type Publisher struct {
	exchange string

	connection  *Connection
	amqpChannel *amqp.Channel

	config AmqpConfig
}

func NewPublisher(conn *Connection, opts ...AmqpConfigOption) (*Publisher, error) {
	amqpChannel, err := conn.amqpConnection.Channel()
	if err != nil {
		return nil, fmt.Errorf("could not start a new broker channel: %w", err)
	}

	cfg := NewDefaultConfig()

	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.exchange.name != "" {
		err = amqpChannel.ExchangeDeclare(
			cfg.exchange.name,       // Name
			cfg.exchange.kind,       // Type
			cfg.exchange.durable,    // Durable
			cfg.exchange.autoDelete, // Auto-deleted
			cfg.exchange.internal,   // Internal
			cfg.exchange.noWait,     // No wait
			cfg.exchange.args,       // Args
		)

		if err != nil {
			return nil, fmt.Errorf("unable to declare exchange: %w", err)
		}
	}

	return &Publisher{
		connection:  conn,
		amqpChannel: amqpChannel,
		config:      cfg,
	}, nil
}

func (p *Publisher) Publish(topic string, event *pubsublite.Event) error {
	err := p.amqpChannel.Publish(
		p.config.exchange.name,     // exchange
		topic,                      // routing key
		p.config.publish.mandatory, // mandatory
		p.config.publish.immediate, // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         []byte(event.Payload.String()),
		},
	)
	if err != nil {
		return fmt.Errorf("error enqueuing event: %w", err)
	}

	return nil
}

func (p *Publisher) Close() error {
	p.connection.Close()
	// if err := p.amqpChannel.Close(); err != nil {
	//	return err
	//}

	return nil
}
