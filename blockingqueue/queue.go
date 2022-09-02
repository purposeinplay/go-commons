package blockingqueue

import (
	"context"

	"go.uber.org/atomic"
)

// Queue is a Blocking queue, it supports operations that wait
// for the queue to have available elements before providing them.
type Queue[T any] struct {
	elements []T

	c *atomic.Pointer[chan T]
}

// New creates a new Blocking Queue.
func New[T any](elements []T) *Queue[T] {
	c := make(chan T, len(elements))
	for i := range elements {
		c <- elements[i]
	}

	return &Queue[T]{
		elements: elements,
		c:        atomic.NewPointer(&c),
	}
}

// Take retrieves and removes the head of the queue.
func (q *Queue[T]) Take(
	ctx context.Context,
) (v T) {
	select {
	case e, ok := <-*q.c.Load():
		if !ok {
			return q.Take(ctx)
		}

		v = e
	case <-ctx.Done():
	}

	return v
}

// Reset refills the queue with the elements given at construction.
func (q *Queue[T]) Reset() {
	newC := make(chan T, len(q.elements))

	for i := range q.elements {
		newC <- q.elements[i]
	}

	oldC := q.c.Swap(&newC)

	close(*oldC)
}
