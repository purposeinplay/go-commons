package leaderelection

import (
	"errors"
)

// ErrEmptyDSN is returned when the database connection string is empty.
var ErrEmptyDSN = errors.New(
	"database connection string is required for advisory lock operations",
)
