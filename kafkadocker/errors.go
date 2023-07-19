package kafkadocker

import (
	"errors"
)

var (
	ErrBrokerAlreadyStarted = errors.New("broker already started")
	ErrBrokerWasNotStarted  = errors.New("broker was not started")
)
