package amqpw

import (
	"context"
	"fmt"

	"github.com/purposeinplay/go-commons/logs"
	"github.com/purposeinplay/go-commons/worker"
	"go.uber.org/zap"

	"errors"
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

	// MaxConcurrency restricts the amount of workers in parallel.
	MaxConcurrency int
}

// ErrInvalidConnection is returned when the Connection opt is not defined.
var ErrInvalidConnection = errors.New("invalid connection")

// Ensures Adapter implements the buffalo.Worker interface.
var _ worker.Events = &Adapter{}

// New creates a new AMQP adapter
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
	ctx            context.Context
	maxConcurrency int
}

func (q *Adapter) exchangeDeclare(exchanges []string) error {
	for _, e := range exchanges {
		err := q.Channel.ExchangeDeclare(
			e,        // name
			"direct", // type
			true,     // durable
			false,    // auto-deleted
			false,    // internal
			false,    // no-wait
			nil,      // arguments
		)

		if err != nil {
			return fmt.Errorf("unable to declare exchange: %w", err)
		}
	}

	return nil
}

// Start connects to the broker.
func (q *Adapter) Start(ctx context.Context) error {
	q.ctx = ctx
	go func() {
		select {
		case <-ctx.Done():
			q.Stop()
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

	err = q.exchangeDeclare([]string{"win.users", "win.payments", "win.cashout"})
	if err != nil {
		return fmt.Errorf("could not perform exchangeDeclare: %w", err)
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

// Emit enqueues a new job.
func (q Adapter) Emit(job worker.Job) error {
	q.Logger.Info("enqueuing job", zap.Any("job", job))

	err := q.Channel.Publish(
		job.Exchange, // exchange
		job.Handler,  // routing key
		false,        // mandatory
		false,        // immediate
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
