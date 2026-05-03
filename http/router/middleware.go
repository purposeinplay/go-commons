package router

import (
	"fmt"
	"net/http"
	"os"
	"runtime/debug"

	commonshttp "github.com/purposeinplay/go-commons/http"
)

type Middleware func(http.Handler) http.Handler

func MiddlewareFunc(f func(w http.ResponseWriter, r *http.Request, next http.Handler)) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			f(w, r, next)
		})
	}
}

// Recoverer is a middleware that recovers from panics, logs the panic (and a
// backtrace), and returns a HTTP 500 (Internal Server Error) status if
// possible. Recoverer prints a request ID if one is provided.
func Recoverer() Middleware {
	return MiddlewareFunc(func(w http.ResponseWriter, r *http.Request, next http.Handler) {
		defer func() {
			if rvr := recover(); rvr != nil {
				if entry := commonshttp.GetLogEntry(r); entry != nil {
					entry.Panic(rvr, debug.Stack())
					entry.Logger.Error("panic recovered", "panic", fmt.Sprintf("%+v", rvr))
				} else {
					fmt.Fprintf(os.Stderr, "Panic: %+v\n", rvr)
					debug.PrintStack()
				}

				commonshttp.HandleError(&commonshttp.HTTPError{
					Code:    http.StatusInternalServerError,
					Message: http.StatusText(http.StatusInternalServerError),
				}, w, r)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
