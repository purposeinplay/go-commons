package auth

import (
	"context"
	"net/http"
)

// Header keys.
const (
	HeaderUserID   = "X-User-Id"
	HeaderAppID    = "X-App-Id"
	HeaderUsername = "X-Username"
)

type contextKey int

const (
	_ contextKey = iota

	userIDCtxKey
	appIDCtxKey
	usernameCtxKey
)

// UserIDMiddlewareFunc is a middleware that extracts the user ID from the request.
func UserIDMiddlewareFunc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get(HeaderUserID)
		userIDCtx := NewUserIDContext(r.Context(), userID)

		next.ServeHTTP(w, r.WithContext(userIDCtx))
	})
}

// AppIDMiddlewareFunc is a middleware that extracts the app ID from the request.
func AppIDMiddlewareFunc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appID := r.Header.Get(HeaderAppID)
		appIDCtx := NewAppIDContext(r.Context(), appID)

		next.ServeHTTP(w, r.WithContext(appIDCtx))
	})
}

// UsernameMiddlewareFunc is a middleware that extracts the username from the request.
func UsernameMiddlewareFunc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username := r.Header.Get(HeaderUsername)
		usernameCtx := NewUsernameContext(r.Context(), username)

		next.ServeHTTP(w, r.WithContext(usernameCtx))
	})
}

// NewUserIDContext returns a new context with the user ID.
func NewUserIDContext(ctx context.Context, userID string) context.Context {
	if userID == "" {
		return ctx
	}

	return context.WithValue(ctx, userIDCtxKey, userID)
}

// NewAppIDContext returns a new context with the app ID.
func NewAppIDContext(ctx context.Context, appID string) context.Context {
	if appID == "" {
		return ctx
	}

	return context.WithValue(ctx, appIDCtxKey, appID)
}

// NewUsernameContext returns a new context with the username.
func NewUsernameContext(ctx context.Context, username string) context.Context {
	if username == "" {
		return ctx
	}

	return context.WithValue(ctx, usernameCtxKey, usernameCtxKey)
}

// UserIDFromContext returns the user ID from the context.
func UserIDFromContext(ctx context.Context) string {
	// nolint: revive
	userID, _ := ctx.Value(userIDCtxKey).(string)

	return userID
}

// AppIDFromContext returns the app ID from the context.
func AppIDFromContext(ctx context.Context) string {
	// nolint: revive
	appID, _ := ctx.Value(appIDCtxKey).(string)

	return appID
}

// UsernameFromContext returns the username from the context.
func UsernameFromContext(ctx context.Context) string {
	// nolint: revive
	username, _ := ctx.Value(usernameCtxKey).(string)

	return username
}
