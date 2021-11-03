package worker

import "context"

// Handler function that will be run by the worker and given
// a slice of arguments
type Handler func(Args) error

type Worker interface {
	// Start the worker with the given context
	Start(context.Context) error
	// Stop the worker
	Stop() error
	// Perform a job as soon as possibly
	Perform(job Job) error
	// Register a Handler
	Register(string, Handler) error
}
