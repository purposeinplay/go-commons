// Package uuid provides small helpers for working with canonical UUID strings.
//
// NewString generates a UUIDv7 and returns its string form.
// ParseString validates an input UUID string and returns its canonical form.
package uuid

import "github.com/gofrs/uuid/v5"

// NewString returns a new UUIDv7 encoded as a canonical string.
func NewString() string {
	return uuid.Must(uuid.NewV7()).String()
}

// ParseString validates s as a UUID and returns its canonical string form.
func ParseString(s string) (string, error) {
	parsed, err := uuid.FromString(s)
	if err != nil {
		return "", err
	}

	return parsed.String(), nil
}
