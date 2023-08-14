package pubsub

import (
	"errors"
)

var (
	// ErrExactlyOneChannelAllowed is returned a pubsub implementation supports only one channel.
	ErrExactlyOneChannelAllowed = errors.New("exactly one channel allowed")
	// ErrEventPayloadMustBeByteSlice is returned when an event payload is expected to be
	// a byte slice and it's not.
	ErrEventPayloadMustBeByteSlice = errors.New("event payload must be byte slice")
)
