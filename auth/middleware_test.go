package auth_test

import (
	"net/http"
	"testing"

	"github.com/purposeinplay/go-commons/auth"
	"github.com/stretchr/testify/require"
)

func TestMiddlewareUserID(t *testing.T) {
	tests := map[string]struct {
		requestUserID string
	}{
		"NoUserID": {
			requestUserID: "",
		},
		"UserID": {
			requestUserID: "123",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var userID string

			auth.UserIDMiddlewareFunc(
				http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
					userID = auth.UserIDFromContext(r.Context())
				}),
			).ServeHTTP(
				nil,
				&http.Request{
					Header: http.Header{
						auth.HeaderUserID: []string{test.requestUserID},
					},
				},
			)

			t.Log("UserID:", userID)

			require.Equal(t, test.requestUserID, userID)
		})
	}
}
