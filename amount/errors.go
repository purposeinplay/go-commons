package amount

import (
	"errors"
)

// ErrInvalidValue is returned when an unexpected value
// is given to an Amount constructor.
var ErrInvalidValue = errors.New("invalid value")
