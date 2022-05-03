package money

import (
	"errors"
)

// ErrInvalidValue is returned when an unexpected value
// is given to a NewAmount constructor.
var ErrInvalidValue = errors.New("invalid value")
