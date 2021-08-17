package router

import (
	commonshttp "github.com/purposeinplay/go-commons/http"
	"go.uber.org/zap"
	"net/http"

	"github.com/go-chi/chi"
)

type chiRouter struct {
	chi chi.Router
}

// Router wraps the chi router to make it slightly more accessible
type Router interface {
	// Use appends one middleware onto the Router stack.
	Use(fn Middleware)

	// With adds an inline middleware for an endpoint handler.
	With(fn Middleware) Router

	// Route mounts a sub-Router along a `pattern`` string.
	Route(pattern string, fn func(r Router))

	// Method adds a routes for a `pattern` that matches the `method` HTTP method.
	Method(method, pattern string, h HandlerErrorFunc)

	// HTTP-method routing along `pattern`
	Delete(pattern string, h HandlerErrorFunc)
	Get(pattern string, h HandlerErrorFunc)
	Post(pattern string, h HandlerErrorFunc)
	Put(pattern string, h HandlerErrorFunc)

	// Mount attaches another http.Handler along ./pattern/*
	Mount(pattern string, h http.Handler)

	ServeHTTP(http.ResponseWriter, *http.Request)
}

// New creates a router with sensible defaults (xff, request id, cors)
func New(options ...Option) Router {
	r := &chiRouter{
		chi:        chi.NewRouter(),
	}

	for _, opt := range options {
		opt(r)
	}

	return r
}

func NewDefaultRouter(logger *zap.Logger) Router {
	r := New(
		WithRequestID(),
		WithRealIP(),
		WithRecoverer(),
		WithLogger(logger),
		WithCors(defaultCorsOptions),
		WithHealthcheck("/health", nil),
	)

	return r
}

// Route allows creating a generic route
func (r *chiRouter) Route(pattern string, fn func(Router)) {
	r.chi.Route(pattern, func(c chi.Router) {
		wrapper := new(chiRouter)
		*wrapper = *r
		wrapper.chi = c
		fn(wrapper)
	})
}

// Method adds a routes for a `pattern` that matches the `method` HTTP method.
func (r *chiRouter) Method(method, pattern string, h HandlerErrorFunc) {
	f := HandlerFunc(h)
	r.chi.Method(method, pattern, f)
}

// Get adds a GET route
func (r *chiRouter) Get(pattern string, fn HandlerErrorFunc) {
	r.chi.Get(pattern, HandlerFunc(fn))
}

// Post adds a POST route
func (r *chiRouter) Post(pattern string, fn HandlerErrorFunc) {
	r.chi.Post(pattern, HandlerFunc(fn))
}

// Put adds a PUT route
func (r *chiRouter) Put(pattern string, fn HandlerErrorFunc) {
	r.chi.Put(pattern, HandlerFunc(fn))
}

// Delete adds a DELETE route
func (r *chiRouter) Delete(pattern string, fn HandlerErrorFunc) {
	r.chi.Delete(pattern, HandlerFunc(fn))
}

// With adds an inline chi middleware for an endpoint handler
func (r *chiRouter) With(fn Middleware) Router {
	r.chi = r.chi.With(fn)
	return r
}

// Use appends one chi middleware onto the Router stack
func (r *chiRouter) Use(fn Middleware) {
	r.chi.Use(fn)
}

// ServeHTTP will serve a request
func (r *chiRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.chi.ServeHTTP(w, req)
}

// Mount attaches another http.Handler along ./pattern/*
func (r *chiRouter) Mount(pattern string, h http.Handler) {
	r.chi.Mount(pattern, h)
}

// =======================================
// HTTP handler with custom error payload
// =======================================

type HandlerErrorFunc func(w http.ResponseWriter, r *http.Request) error

func HandlerFunc(fn HandlerErrorFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := fn(w, r); err != nil {
			commonshttp.HandleError(err, w, r)
		}
	}
}

func Healthcheck() HandlerErrorFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		w.WriteHeader(http.StatusOK)
		return nil
	}
}
