package auth

import (
	"net/http"
	"context"
)

const (
	HeaderUserID = "X-USER-ID"
	HeaderAppID  = "X-APP-ID"
)

type contextKey int

const (
	_ contextKey = iota

	userIDCtxKey
	appIDCtxKey
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

func UserIDFromContext(ctx context.Context) string {
	userID, _ := ctx.Value(userIDCtxKey).(string)

	return userID
}

func AppIDFromContext(ctx context.Context) string {
	appID, _ := ctx.Value(appIDCtxKey).(string)

	return appID
}
