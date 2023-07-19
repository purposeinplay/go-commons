package kafkadocker

import (
	"errors"
)

var (
	// ErrBrokerAlreadyStarted is returned when a broker is started more than once.
	ErrBrokerAlreadyStarted = errors.New("broker already started")
	// ErrBrokerWasNotStarted is returned when a broker is stopped before it is started.
	ErrBrokerWasNotStarted = errors.New("broker was not started")
)
