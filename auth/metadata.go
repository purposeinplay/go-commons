package auth

import (
	"errors"
	"strings"

	"google.golang.org/grpc/metadata"
)

// GRPC Metadata auth errors.
var (
	ErrAuthTokenMissing = errors.New("auth token is missing")
	ErrBadAuthString    = errors.New("bad authorization string")
)

// ExtractTokenFromMetadata extracts the token from the grpc metadata.
func ExtractTokenFromMetadata(md metadata.MD) (string, error) {
	meta, ok := md["authorization"]
	if !ok {
		return "", ErrAuthTokenMissing
	}

	auth := meta[0]

	const prefix = "Bearer "

	if !strings.HasPrefix(auth, prefix) {
		return "", ErrBadAuthString
	}

	return auth[len(prefix):], nil
}
