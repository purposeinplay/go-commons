package auth

import (
	"context"
	"net/http"
)

const (
	HeaderUserID   = "X-USER-ID"
	HeaderAppID    = "X-APP-ID"
	HeaderUsername = "X-USERNAME"
)

type contextKey int

const (
	_ contextKey = iota

	userIDCtxKey
	appIDCtxKey
	usernameCtxKey
)

func UserIDMiddlewareFunc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get(HeaderUserID)
		userIDCtx := NewUserIDContext(r.Context(), userID)

		next.ServeHTTP(w, r.WithContext(userIDCtx))
	})
}

func AppIDMiddlewareFunc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appID := r.Header.Get(HeaderAppID)
		appIDCtx := NewAppIDContext(r.Context(), appID)

		next.ServeHTTP(w, r.WithContext(appIDCtx))
	})
}

func UsernameMiddlewareFunc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username := r.Header.Get(HeaderUsername)
		usernameCtx := NewUsernameContext(r.Context(), username)

		next.ServeHTTP(w, r.WithContext(usernameCtx))
	})
}

func NewUserIDContext(ctx context.Context, userID string) context.Context {
	if userID == "" {
		return ctx
	}

	return context.WithValue(ctx, userIDCtxKey, userID)
}

func NewAppIDContext(ctx context.Context, appID string) context.Context {
	if appID == "" {
		return ctx
	}

	return context.WithValue(ctx, appIDCtxKey, appID)
}

func NewUsernameContext(ctx context.Context, username string) context.Context {
	if username == "" {
		return ctx
	}

	return context.WithValue(ctx, usernameCtxKey, usernameCtxKey)
}

func UserIDFromContext(ctx context.Context) string {
	userID, _ := ctx.Value(userIDCtxKey).(string)

	return userID
}

func AppIDFromContext(ctx context.Context) string {
	appID, _ := ctx.Value(appIDCtxKey).(string)

	return appID
}

func UsernameFromContext(ctx context.Context) string {
	username, _ := ctx.Value(usernameCtxKey).(string)

	return username
}
