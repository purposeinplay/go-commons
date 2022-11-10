package rabbitmq

import (
	"fmt"

	"github.com/ThreeDotsLabs/watermill-amqp/pkg/amqp"
)

// NewSubscriberConfig creates a new Watermill subscriber config.
func NewSubscriberConfig(
	amqpURI string,
	queueSuffix string,
	exchange string,
	routeKey string,
) (amqp.Config, error) {
	cfg := amqp.Config{
		Connection: amqp.ConnectionConfig{
			AmqpURI: amqpURI,
		},

		Marshaler: amqp.DefaultMarshaler{},

		Exchange: amqp.ExchangeConfig{
			GenerateName: func(topic string) string {
				return exchange
			},
			Type:    "direct",
			Durable: true,
		},
		Queue: amqp.QueueConfig{
			GenerateName: amqp.GenerateQueueNameTopicNameWithSuffix(
				queueSuffix,
			),
			Durable: true,
		},
		QueueBind: amqp.QueueBindConfig{
			GenerateRoutingKey: func(topic string) string {
				return routeKey
			},
		},
		Publish: amqp.PublishConfig{
			GenerateRoutingKey: func(topic string) string {
				return routeKey
			},
		},
		Consume: amqp.ConsumeConfig{
			Qos: amqp.QosConfig{
				PrefetchCount: 1,
			},
		},
		TopologyBuilder: &amqp.DefaultTopologyBuilder{},
	}

	err := cfg.ValidateSubscriber()
	if err != nil {
		return amqp.Config{}, fmt.Errorf("validate subscriber config: %w", err)
	}

	return cfg, nil
}

// NewPublisherConfig creates a new Watermill publisher config.
func NewPublisherConfig(
	amqpURI string,
	queueSuffix string,
	exchange string,
	routeKey string,
) (amqp.Config, error) {
	config := amqp.Config{
		Connection: amqp.ConnectionConfig{
			AmqpURI: amqpURI,
		},

		Marshaler: amqp.DefaultMarshaler{},

		Exchange: amqp.ExchangeConfig{
			GenerateName: func(topic string) string {
				return exchange
			},
			Type:    "direct",
			Durable: true,
		},
		Queue: amqp.QueueConfig{
			GenerateName: amqp.GenerateQueueNameTopicNameWithSuffix(
				queueSuffix,
			),
			Durable: true,
		},
		QueueBind: amqp.QueueBindConfig{
			GenerateRoutingKey: func(topic string) string {
				return routeKey
			},
		},
		Publish: amqp.PublishConfig{
			GenerateRoutingKey: func(topic string) string {
				return routeKey
			},
		},
		Consume: amqp.ConsumeConfig{
			Qos: amqp.QosConfig{
				PrefetchCount: 1,
			},
		},
		TopologyBuilder: &amqp.DefaultTopologyBuilder{},
	}

	err := config.ValidatePublisher()
	if err != nil {
		return amqp.Config{}, err
	}

	return config, nil
}
