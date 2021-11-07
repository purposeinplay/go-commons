package amqp

import "github.com/streadway/amqp"

type ExchangeConfig struct {
	name       string
	kind       string
	durable    bool
	autoDelete bool
	internal   bool
	noWait     bool
	args       amqp.Table
}

type QueueNameGenerator func(topic string) string

// GenerateQueueNameTopicName generates queueName equal to the topic.
func GenerateQueueNameTopicName(topic string) string {
	return topic
}

// GenerateQueueNameTopicNameWithPrefix generates queue name equal to:
// 	prefix + "." + topic
func GenerateQueueNameTopicNameWithPrefix(prefix string) QueueNameGenerator {
	return func(topic string) string {
		return prefix + "." + topic
	}
}

type QueueConfig struct {
	generateName QueueNameGenerator
	durable      bool
	autoDelete   bool
	exclusive    bool
	noWait       bool
	arguments    amqp.Table
}

type QueueBindNameGenerator func(topic string) string

type QueueBindConfig struct {
	generateRoutingKey QueueBindNameGenerator
	noWait             bool
	args               amqp.Table
}

type ConsumeConfig struct {
	consumer  string
	autoAck   bool
	exclusive bool
	noLocal   bool
	noWait    bool
	args      amqp.Table
}

type PublishConfig struct {
	mandatory bool
	immediate bool
}

type AmqpConfig struct {
	exchange  ExchangeConfig
	queue     QueueConfig
	queueBind QueueBindConfig
	consume   ConsumeConfig
	publish   PublishConfig
}

type AmqpConfigOption func(config *AmqpConfig)

func WithExchangeName(name string) AmqpConfigOption {
	return func(c *AmqpConfig) {
		c.exchange.name = name
	}
}

func WithQueueNameGenerator(f QueueNameGenerator) AmqpConfigOption {
	return func(c *AmqpConfig) {
		c.queue.generateName = f
	}
}

func NewDefaultConfig() AmqpConfig {
	return AmqpConfig{
		queue: QueueConfig{
			generateName: GenerateQueueNameTopicName,
			durable:      true,
			autoDelete:   false,
			exclusive:    false,
			noWait:       false,
			arguments:    nil,
		},
		exchange: ExchangeConfig{
			"",
			"direct",
			true,
			false,
			false,
			false,
			nil,
		},
		queueBind: QueueBindConfig{
			generateRoutingKey: func(topic string) string {
				return topic
			},
			noWait: false,
			args:   nil,
		},
		consume: ConsumeConfig{
			"",
			false, // auto-ack
			false, // exclusive
			false, // no-local
			false, // no-wait
			nil,   // args
		},
		publish: PublishConfig{
			mandatory: true,
			immediate: false,
		},
	}
}
