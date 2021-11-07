package amqp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/purposeinplay/go-commons/pubsublite"
	"github.com/streadway/amqp"
)

var _ pubsublite.Subscriber = &Subscriber{}

type Subscriber struct {
	amqpConnection *amqp.Connection
	// subscription name
	name string
	// name of the exchange
	topic string
	// Consumer channel
	amqpChannel *amqp.Channel
	// ConsumerTag is a unique tag of the amqp consumer we want to cancel.
	consumerTag string
	cfg         AmqpConfig

	closing chan struct{}
}

func NewSubscription(conn *amqp.Connection, opts ...AmqpConfigOption) (*Subscriber, error) {
	amqpChannel, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("could not start a new broker channel: %w", err)
	}

	closing := make(chan struct{})

	cfg := NewDefaultConfig()

	for _, opt := range opts {
		// Call the option giving the instantiated
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

	return &Subscriber{
		amqpConnection: conn,
		amqpChannel:    amqpChannel,
		cfg:            cfg,
		closing:        closing,
	}, nil
}

func (s *Subscriber) Subscribe(ctx context.Context, topic string) (<-chan *pubsublite.Event, error) {
	out := make(chan *pubsublite.Event)

	queueName := s.cfg.queue.generateName(topic)
	routingKey := s.cfg.queueBind.generateRoutingKey(topic)
	exchangeName := s.cfg.exchange.name

	err := s.setupQueue(routingKey, queueName, exchangeName)
	if err != nil {
		return nil, err
	}

	msgs, err := s.createConsumer(queueName)
	if err != nil {
		return nil, err
	}

	go func() {
	ConsumingLoop:
		for msg := range msgs {
			evt := pubsublite.NewEvent(msg.RoutingKey)
			_ = json.Unmarshal(msg.Body, &evt.Payload)

			select {
			case <-s.closing:
				break ConsumingLoop
			case out <- evt:
				// log event sent to consumer
			}

			<-evt.Acked()
			msg.Ack(false)
		}
	}()

	return out, nil
}

func (s *Subscriber) setupQueue(routingKey string, queueName string, exchangeName string) error {
	_, err := s.amqpChannel.QueueDeclare(
		queueName,
		s.cfg.queue.durable,
		s.cfg.queue.autoDelete,
		s.cfg.queue.exclusive,
		s.cfg.queue.noWait,
		s.cfg.queue.arguments,
	)
	if err != nil {
		return fmt.Errorf("unable to declare queue: %w", err)
	}

	if exchangeName != "" {
		err = s.amqpChannel.QueueBind(
			queueName,              // queue name
			routingKey,             // routing key
			exchangeName,           // exchange
			s.cfg.queueBind.noWait, // no wait
			s.cfg.queueBind.args,   // args
		)

		if err != nil {
			return fmt.Errorf("unable to bind queue: %w", err)
		}
	}

	return nil
}

func (s *Subscriber) createConsumer(queueName string) (<-chan amqp.Delivery, error) {
	msgs, err := s.amqpChannel.Consume(
		queueName,
		s.cfg.consume.consumer,
		s.cfg.consume.autoAck,
		s.cfg.consume.exclusive,
		s.cfg.consume.noLocal,
		s.cfg.consume.noWait,
		s.cfg.consume.args,
	)
	if err != nil {
		return nil, fmt.Errorf("could not consume channel: %w", err)
	}

	return msgs, nil
}

func (s *Subscriber) Close() error {
	if err := s.amqpChannel.Cancel(s.consumerTag, true); err != nil {
		return err
	}

	if err := s.amqpChannel.Close(); err != nil {
		return err
	}

	close(s.closing)

	return nil
}
