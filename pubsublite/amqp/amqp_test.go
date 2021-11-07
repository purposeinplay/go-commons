package amqp

import (
	"context"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/purposeinplay/go-commons/pubsublite"

	"github.com/stretchr/testify/require"
)

func TestAmqp_PublishWithDefaultExchange(t *testing.T) {
	conn, err := NewConnection(WithConnection(
		ConnectionConfig{url: "amqp://guest:guest@localhost:5672"},
	))
	require.NoError(t, err)

	ctx := context.Background()

	publisher, err := NewPublisher(conn)
	require.NoError(t, err)

	publisher.Publish("user_deleted", &pubsublite.Event{
		Payload: nil,
	})

	conn2, err := NewConnection(WithConnection(
		ConnectionConfig{url: "amqp://guest:guest@localhost:5672"},
	))
	require.NoError(t, err)

	var hit bool
	wg := &sync.WaitGroup{}
	wg.Add(1)

	subscriber, err := NewSubscription(conn2)
	require.NoError(t, err)

	messages, err := subscriber.Subscribe(ctx, "user_deleted")
	require.NoError(t, err)

	select {
	case <-time.After(2 * time.Second):
		t.Fatal("Timed out waiting for message")
	case msg := <-messages:
		hit = true
		wg.Done()
		msg.Ack()
	}

	require.True(t, hit)
	require.NoError(t, conn.Close())
}

func TestAmqp_PublishWithCustomExchange(t *testing.T) {
	conn, err := NewConnection(WithConnection(
		ConnectionConfig{url: "amqp://guest:guest@localhost:5672"},
	))
	require.NoError(t, err)

	ctx := context.Background()

	publisher, err := NewPublisher(conn, WithExchangeName("custom_exchange"))
	require.NoError(t, err)

	publisher.Publish("payment_created", &pubsublite.Event{
		Payload: nil,
	})

	conn2, err := NewConnection(WithConnection(
		ConnectionConfig{url: "amqp://guest:guest@localhost:5672"},
	))
	require.NoError(t, err)

	var hit bool
	wg := &sync.WaitGroup{}
	wg.Add(1)

	subscriber, err := NewSubscription(conn2, WithExchangeName("custom_exchange"))
	require.NoError(t, err)

	messages, err := subscriber.Subscribe(ctx, "payment_created")
	require.NoError(t, err)

	select {
	case <-time.After(2 * time.Second):
		t.Fatal("Timed out waiting for message")
	case msg := <-messages:
		hit = true
		wg.Done()
		msg.Ack()
	}

	wg.Wait()
	require.True(t, hit)
	require.NoError(t, conn.Close())
}

func TestAmqp_PublishWithExchangeWithQueuePrefix(t *testing.T) {
	conn, err := NewConnection(WithConnection(
		ConnectionConfig{url: "amqp://guest:guest@localhost:5672"},
	))
	require.NoError(t, err)

	ctx := context.Background()

	publisher, err := NewPublisher(conn, WithExchangeName("custom_exchange"))
	require.NoError(t, err)

	publisher.Publish("payment_created", &pubsublite.Event{
		Payload: nil,
	})

	conn2, err := NewConnection(WithConnection(
		ConnectionConfig{url: "amqp://guest:guest@localhost:5672"},
	))
	require.NoError(t, err)

	var hit bool
	wg := &sync.WaitGroup{}
	wg.Add(1)

	subscriber, err := NewSubscription(conn2,
		WithExchangeName("custom_exchange"),
		WithQueueNameGenerator(GenerateQueueNameTopicNameWithPrefix("mailman.users")),
	)
	require.NoError(t, err)

	messages, err := subscriber.Subscribe(ctx, "payment_created")
	require.NoError(t, err)

	select {
	case <-time.After(2 * time.Second):
		t.Fatal("Timed out waiting for message")
	case msg := <-messages:
		hit = true
		wg.Done()
		msg.Ack()
	}

	wg.Wait()
	require.True(t, hit)
	require.NoError(t, conn.Close())
}

func TestAmqp_PublishWithExchangeMultipleSubscriptions(t *testing.T) {
	conn, err := NewConnection(WithConnection(
		ConnectionConfig{url: "amqp://guest:guest@localhost:5672"},
	))
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	publisher, err := NewPublisher(conn, WithExchangeName("custom_exchange"))
	require.NoError(t, err)

	publisher.Publish("payment_created", &pubsublite.Event{
		Payload: nil,
	})

	conn2, err := NewConnection(WithConnection(
		ConnectionConfig{url: "amqp://guest:guest@localhost:5672"},
	))
	require.NoError(t, err)

	conn3, err := NewConnection(WithConnection(
		ConnectionConfig{url: "amqp://guest:guest@localhost:5672"},
	))
	require.NoError(t, err)

	var hit bool
	var hit2 bool
	wg := &sync.WaitGroup{}
	wg.Add(2)

	subscriber, err := NewSubscription(conn2, WithExchangeName("custom_exchange"))
	require.NoError(t, err)

	messages, err := subscriber.Subscribe(ctx, "payment_created")
	require.NoError(t, err)

	subscriber2, err := NewSubscription(conn3,
		WithExchangeName("custom_exchange"),
		WithQueueNameGenerator(GenerateQueueNameTopicNameWithPrefix("mailman.users")),
	)
	require.NoError(t, err)

	messages2, err := subscriber2.Subscribe(ctx, "payment_created")
	require.NoError(t, err)

	go func() {
		select {
		case <-time.After(2 * time.Second):
			t.Fatal("Timed out waiting for message")
		case msg := <-messages:
			hit = true
			wg.Done()
			msg.Ack()
		}
	}()

	go func() {
		select {
		case <-time.After(2 * time.Second):
			t.Fatal("Timed out waiting for message")
		case msg := <-messages2:
			hit2 = true
			wg.Done()
			msg.Ack()
		}
	}()

	wg.Wait()
	require.True(t, hit)
	require.True(t, hit2)
	require.NoError(t, conn.Close())
}
