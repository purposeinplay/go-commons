package amqpw

import (
	"context"
	"log"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/purposeinplay/go-commons/rand"
	"github.com/purposeinplay/go-commons/worker"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/require"
)

var q *Adapter

// Setup the adapter.
func TestMain(m *testing.M) {
	l := slog.Default()

	// Setup AMQP connection
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672")
	if err != nil {
		log.Fatal(err)
	}

	q, err = New(Options{
		Connection: conn,
		Name:       rand.String(20),
	})

	if err != nil {
		l.Error("new adapter", slog.Any("error", err))
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())

	ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	go func() {
		select {
		case <-ctx.Done():
			cancel()
			l.Error(ctx.Err().Error(), slog.Any("error", ctx.Err()))
		}
	}()

	err = q.Start(ctx)
	if err != nil {
		cancel()
		l.Error(err.Error(), slog.Any("error", err))
		os.Exit(1)
	}

	code := m.Run()

	err = q.Stop()
	if err != nil {
		l.Error(err.Error(), slog.Any("error", err))
		os.Exit(1)
	}

	l.Info("Test stopped")

	os.Exit(code)
}

func Test_Perform(t *testing.T) {
	r := require.New(t)

	var hit bool

	wg := &sync.WaitGroup{}

	wg.Add(1)

	q.Register("perform", func(worker.Args) error {
		hit = true
		wg.Done()

		return nil
	})

	q.Perform(worker.Job{
		Handler: "perform",
	})

	wg.Wait()

	r.True(hit)
}

func Test_PerformMultiple(t *testing.T) {
	r := require.New(t)

	var hitPerform1, hitPerform2 bool

	wg := &sync.WaitGroup{}
	wg.Add(2)

	q.Register("perform1", func(worker.Args) error {
		hitPerform1 = true
		wg.Done()
		return nil
	})

	q.Register("perform2", func(worker.Args) error {
		hitPerform2 = true
		wg.Done()
		return nil
	})

	q.Perform(worker.Job{
		Handler: "perform1",
	})

	q.Perform(worker.Job{
		Handler: "perform2",
	})

	wg.Wait()

	r.True(hitPerform1)
	r.True(hitPerform2)
}

func Test_PerformAt(t *testing.T) {
	r := require.New(t)

	var hit bool

	wg := &sync.WaitGroup{}

	wg.Add(1)

	q.Register("perform_at", func(_ worker.Args) error {
		hit = true
		wg.Done()
		return nil
	})

	q.PerformAt(worker.Job{
		Handler: "perform_at",
	}, time.Now().Add(5*time.Nanosecond))

	wg.Wait()

	r.True(hit)
}

func Test_PerformIn(t *testing.T) {
	r := require.New(t)

	var hit bool
	wg := &sync.WaitGroup{}
	wg.Add(1)
	q.Register("perform_in", func(worker.Args) error {
		hit = true
		wg.Done()
		return nil
	})
	q.PerformIn(worker.Job{
		Handler: "perform_in",
	}, 5*time.Nanosecond)
	wg.Wait()
	r.True(hit)
}
