package worker

import (
	"context"
	"time"
)

// Handler function that will be run by the worker and given
// a slice of arguments.
type Handler func(Args) error

// Worker describes how a worker should be implemented.
type Worker interface {
	// Start the worker with the given context
	Start(context.Context) error

	// Stop the worker
	Stop() error

	// Perform a job as soon as possibly
	Perform(job Job) error

	// PerformAt performs a job at a particular time
	PerformAt(Job, time.Time) error

	// PerformIn performs a job after waiting for a specified amount of time
	PerformIn(Job, time.Duration) error

	// Register a Handler
	Register(string, Handler) error
}
