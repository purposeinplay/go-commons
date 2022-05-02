package amount

import (
	"errors"
)

// ErrInvalidValue is returned when an unexpected value
// is given to a NewMoney constructor.
var ErrInvalidValue = errors.New("invalid value")
