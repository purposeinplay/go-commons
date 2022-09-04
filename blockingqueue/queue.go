package blockingqueue

import (
	"context"
	"sync"
)

// Queue is a Blocking queue, it supports operations that wait
// for the queue to have available elements before providing them.
type Queue[T any] struct {
	elements []T

	elementsChan  chan T
	resetMutex    sync.Mutex
	elementsIndex int
}

// New creates a new Blocking Queue.
func New[T any](elements []T) *Queue[T] {
	c := make(chan T, len(elements))
	for i := range elements {
		c <- elements[i]
	}

	return &Queue[T]{
		elements:      elements,
		elementsChan:  c,
		resetMutex:    sync.Mutex{},
		elementsIndex: 0,
	}
}

// Take retrieves and removes the head of the queue.
func (q *Queue[T]) Take(
	ctx context.Context,
) (v T) {
	select {
	case v = <-q.elementsChan:
	case <-ctx.Done():
	}

	return v
}

// Refill sends elements into the elements chan until the channel is full.
func (q *Queue[T]) Refill() {
	q.resetMutex.Lock()
	defer q.resetMutex.Unlock()

	// execute the loop until the elements channel is full.
	for i := q.elementsIndex; len(q.elementsChan) <= cap(q.elementsChan); i++ {
		// if the elements slice is consumed, reset the index and consume
		// it again from the start.
		if i == len(q.elements) {
			i = 0
		}

		select {
		case q.elementsChan <- q.elements[i]:
			// successfully sent an element to the elements channel.
		default:
			// channel is full, store the elements index and return.
			q.elementsIndex = i
			return
		}
	}
}
