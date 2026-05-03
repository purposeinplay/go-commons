package router

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	commonshttp "github.com/purposeinplay/go-commons/http"
)

type Option func(r *chiRouter)

var defaultCorsOptions = cors.Options{
	AllowedMethods:   []string{"GET", "POST", "PATCH", "PUT", "DELETE"},
	AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
	ExposedHeaders:   []string{"Link", "X-Total-Count"},
	AllowCredentials: true,
}

func WithCors(options cors.Options) Option {
	return func(r *chiRouter) {
		corsMiddleware := cors.New(options)

		r.Use(corsMiddleware.Handler)
	}
}

func WithHealthcheck(path string, handler http.HandlerFunc) Option {
	return func(r *chiRouter) {
		if handler == nil {
			handler = Healthcheck()
		}

		r.Get(path, handler)
	}
}

func WithRecoverer() Option {
	return func(r *chiRouter) {
		r.Use(Recoverer())
	}
}

func WithRequestID() Option {
	return func(r *chiRouter) {
		r.Use(middleware.RequestID)
	}
}

func WithRealIP() Option {
	return func(r *chiRouter) {
		r.Use(middleware.RealIP)
	}
}

func WithLogger(logger *slog.Logger) Option {
	return func(r *chiRouter) {
		r.Use(commonshttp.NewStructuredLogger(logger))
	}
}
