package amqpw

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/purposeinplay/go-commons/rand"

	"github.com/purposeinplay/go-commons/logs"
	"github.com/purposeinplay/go-commons/worker"
	"go.uber.org/zap"

	"github.com/streadway/amqp"
)

// Options are used to configure the AMQP worker adapter.
type Options struct {
	// Connection is the AMQP connection to use.
	Connection *amqp.Connection

	// Logger is a logger interface to write the worker logs.
	Logger *zap.Logger

	// Name is used to identify the app as a consumer. Defaults to "win".
	Name string

	// Exchange is used to customize the AMQP exchange name. Defaults to "".
	Exchange string

	// MaxConcurrency restricts the amount of workers in parallel.
	MaxConcurrency int
}

// ErrInvalidConnection is returned when the Connection opt is not defined.
var ErrInvalidConnection = errors.New("invalid connection")

// Ensures Adapter implements the Worker interface.
var _ worker.Worker = &Adapter{}

// New creates a new AMQP adapter.
func New(opts Options) (*Adapter, error) {
	ctx := context.Background()

	if opts.Name == "" {
		opts.Name = "win"
	}

	if opts.MaxConcurrency == 0 {
		opts.MaxConcurrency = 25
	}

	if opts.Logger == nil {
		l, err := logs.NewLogger()
		if err != nil {
			return nil, err
		}
		opts.Logger = l
	}

	return &Adapter{
		Connection:     opts.Connection,
		Logger:         opts.Logger,
		consumerName:   opts.Name,
		maxConcurrency: opts.MaxConcurrency,
		ctx:            ctx,
	}, nil
}

// Adapter implements the buffalo.Worker interface.
type Adapter struct {
	Connection     *amqp.Connection
	Channel        *amqp.Channel
	Logger         *zap.Logger
	consumerName   string
	exchange       string
	ctx            context.Context
	maxConcurrency int
}

// Start connects to the broker.
func (q *Adapter) Start(ctx context.Context) error {
	q.ctx = ctx

	go func() {
		select {
		case <-ctx.Done():
			_ = q.Stop()
		}
	}()

	// Ensure Connection is defined
	if q.Connection == nil {
		return ErrInvalidConnection
	}

	// Start new broker channel
	c, err := q.Connection.Channel()
	if err != nil {
		fmt.Println(err)
		return fmt.Errorf("could not start a new broker channel: %w", err)
	}

	q.Channel = c

	// Declare exchange
	if q.exchange != "" {
		err = c.ExchangeDeclare(
			q.exchange, // Name
			"direct",   // Type
			true,       // Durable
			false,      // Auto-deleted
			false,      // Internal
			false,      // No wait
			nil,        // Args
		)

		if err != nil {
			return fmt.Errorf("unable to declare exchange: %w", err)
		}
	}

	return nil
}

// Stop closes the connection to the broker.
func (q *Adapter) Stop() error {
	q.Logger.Info("stopping AMQP worker")

	if q.Channel == nil {
		return nil
	}

	if err := q.Channel.Close(); err != nil {
		return err
	}

	return q.Connection.Close()
}

// Perform enqueues a new job.
func (q Adapter) Perform(job worker.Job) error {
	q.Logger.Info("enqueuing job", zap.Any("job", job))

	err := q.Channel.Publish(
		q.exchange,  // exchange
		job.Handler, // routing key
		true,        // mandatory
		false,       // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         []byte(job.Args.String()),
		},
	)
	if err != nil {
		q.Logger.Error("error enqueuing job", zap.Any("job", job))

		return fmt.Errorf("error enqueuing job: %w", err)
	}

	return nil
}

// Register consumes a task, using the declared worker.Handler.
func (q *Adapter) Register(name string, h worker.Handler) error {
	q.Logger.Info("register job", zap.Any("job", name))

	_, err := q.Channel.QueueDeclare(
		name,
		true,
		false,
		false,
		false,
		amqp.Table{},
	)
	if err != nil {
		return fmt.Errorf("unable to create queue: %w", err)
	}

	msgs, err := q.Channel.Consume(
		name,
		fmt.Sprintf("%s_%s_%s", q.consumerName, name, rand.String(20)),
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("could not consume queue: %w", err)
	}

	// Process jobs with maxConcurrency workers
	sem := make(chan bool, q.maxConcurrency)
	go func() {
		for d := range msgs {
			sem <- true
			q.Logger.Info("received job", zap.Any("job", name), zap.Any("body", d.Body))

			args := worker.Args{}

			err := json.Unmarshal(d.Body, &args)
			if err != nil {
				q.Logger.Info("unable to retrieve job", zap.Any("job", name))
				continue
			}

			if err := h(args); err != nil {
				q.Logger.Info("unable to process job", zap.Any("job", name))
				continue
			}

			if err := d.Ack(false); err != nil {
				q.Logger.Info("unable to ack job", zap.Any("job", name))
			}
		}

		for i := 0; i < cap(sem); i++ {
			sem <- true
		}
	}()

	return nil
}

// PerformIn performs a job delayed by the given duration.
func (q Adapter) PerformIn(job worker.Job, t time.Duration) error {
	q.Logger.Info("enqueuing job", zap.Any("job", job))
	d := int64(t / time.Second)

	// Trick broker using x-dead-letter feature:
	// the message will be pushed in a temp queue with the given duration as TTL.
	// When the TTL expires, the message is forwarded to the original queue.
	dq, err := q.Channel.QueueDeclare(
		fmt.Sprintf("%s_delayed_%d", job.Handler, d),
		true, // Save on disk
		true, // Auto-deletion
		false,
		true,
		amqp.Table{
			"x-message-ttl":             d,
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": job.Handler,
		},
	)
	if err != nil {
		q.Logger.Info("error creating delayed temp queue for job %w", zap.Any("job", job.Handler))
		return err
	}

	err = q.Channel.Publish(
		q.exchange, // exchange
		dq.Name,    // publish to temp delayed queue
		true,       // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         []byte(job.Args.String()),
		},
	)

	if err != nil {
		q.Logger.Info("error enqueuing job %w", zap.Any("job", job.Handler))
		return err
	}
	return nil
}

// PerformAt performs a job at the given time.
func (q Adapter) PerformAt(job worker.Job, t time.Time) error {
	return q.PerformIn(job, time.Until(t))
}
