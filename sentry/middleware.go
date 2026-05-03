package sentry

import (
	"log/slog"
	"net/http"

	sentrygo "github.com/getsentry/sentry-go"
)

// Middleware returns a chi-compatible HTTP middleware that recovers from
// panics inside the request handler chain, reports them to Sentry via
// the supplied Client, and flushes the buffered events when the request
// ends.
//
// Usage:
//
//	r := chi.NewRouter()
//	client, _ := sentry.NewClient(dsn, env, release, sampleRate)
//	r.Use(sentry.Middleware(client))
//
// On panic, the middleware:
//  1. Calls sentry.CurrentHub().Recover(rvr) to capture the event.
//  2. Calls client.Close() to flush buffered events before the
//     response is finalised.
//
// The recovered panic is NOT re-raised by this middleware; pair with
// chi's middleware.Recoverer (or an equivalent) if you also need a 500
// response written for the client.
func Middleware(client *Client) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rvr := recover(); rvr != nil {
					sentrygo.CurrentHub().Recover(rvr)
				}

				if err := client.Close(); err != nil {
					slog.Default().Error(
						"closing sentry on request end",
						slog.Any("error", err),
					)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
