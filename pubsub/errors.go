package pubsub

import (
	"errors"
)

// ErrExactlyOneChannelAllowed is returned a pubsub implementation supports only one channel.
var ErrExactlyOneChannelAllowed = errors.New("exactly one channel allowed")
