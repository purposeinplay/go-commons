package router

import (
	"context"
	"net/http"

	"github.com/purposeinplay/go-commons/http/httperr"

	"github.com/go-chi/chi"
)

func NewRouter() *Router {
	return &Router{chi.NewRouter()}
}

type Router struct {
	Chi chi.Router
}

func (r *Router) Route(pattern string, fn func(*Router)) {
	r.Chi.Route(pattern, func(c chi.Router) {
		fn(&Router{c})
	})
}

func (r *Router) Group(fn func(*Router)) {
	r.Chi.Group(func(c chi.Router) {
		fn(&Router{c})
	})
}

func (r *Router) Get(pattern string, fn handlerFunc) {
	r.Chi.Get(pattern, handler(fn))
}
func (r *Router) Post(pattern string, fn handlerFunc) {
	r.Chi.Post(pattern, handler(fn))
}
func (r *Router) Put(pattern string, fn handlerFunc) {
	r.Chi.Put(pattern, handler(fn))
}
func (r *Router) Delete(pattern string, fn handlerFunc) {
	r.Chi.Delete(pattern, handler(fn))
}

func (r *Router) With(fn MiddlewareHandler) *Router {
	c := r.Chi.With(middleware(fn))
	return &Router{c}
}

func (r *Router) Use(fn MiddlewareHandler) {
	r.Chi.Use(middleware(fn))
}
func (r *Router) UseBypass(fn func(next http.Handler) http.Handler) {
	r.Chi.Use(fn)
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.Chi.ServeHTTP(w, req)
}

type handlerFunc func(w http.ResponseWriter, r *http.Request) error

func handler(fn handlerFunc) http.HandlerFunc {
	return fn.serve
}

func (h handlerFunc) serve(w http.ResponseWriter, r *http.Request) {
	if err := h(w, r); err != nil {
		httperr.HandleError(err, w, r)
	}
}

type MiddlewareHandler func(w http.ResponseWriter, r *http.Request) (context.Context, error)

func (m MiddlewareHandler) handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.serve(next, w, r)
	})
}

func (m MiddlewareHandler) serve(next http.Handler, w http.ResponseWriter, r *http.Request) {
	ctx, err := m(w, r)
	if err != nil {
		httperr.HandleError(err, w, r)
		return
	}
	if ctx != nil {
		r = r.WithContext(ctx)
	}
	next.ServeHTTP(w, r)
}

func middleware(fn MiddlewareHandler) func(http.Handler) http.Handler {
	return fn.handler
}
