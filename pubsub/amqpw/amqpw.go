package amqpw

import (
	"context"
	fmt "fmt"

	"github.com/purposeinplay/go-commons/logs"
	"github.com/purposeinplay/go-commons/pubsub"
	"go.uber.org/zap"

	"github.com/pkg/errors"
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
var _ pubsub.Events = &Adapter{}

// New creates a new AMQP adapter
func New(opts Options) *Adapter {
	ctx := context.Background()

	if opts.Name == "" {
		opts.Name = "win"
	}

	if opts.MaxConcurrency == 0 {
		opts.MaxConcurrency = 25
	}

	if opts.Logger == nil {
		l := logs.NewLogger()
		opts.Logger = l
	}

	return &Adapter{
		Connection:     opts.Connection,
		Logger:         opts.Logger,
		consumerName:   opts.Name,
		maxConcurrency: opts.MaxConcurrency,
		ctx:            ctx,
	}
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
			return errors.WithMessage(err, "unable to declare exchange")
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
		return errors.WithStack(err)
	}

	q.Channel = c

	q.exchangeDeclare([]string{"win.users", "win.payments", "win.cashout"})

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
func (q Adapter) Emit(job pubsub.Job) error {
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

		return errors.WithStack(err)
	}
	return nil
}
