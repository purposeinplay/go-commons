package middleware

import (
	"context"
	"fmt"
	"github.com/purposeinplay/go-commons/http/httperr"
	"github.com/purposeinplay/go-commons/logs"
	"net/http"
	"os"
	"runtime/debug"
)

// Recoverer is a middleware that recovers from panics, logs the panic (and a
// backtrace), and returns a HTTP 500 (Internal Server Error) status if
// possible. Recoverer prints a request ID if one is provided.
func Recoverer(w http.ResponseWriter, r *http.Request) (context.Context, error) {
	defer func() {
		if rvr := recover(); rvr != nil {
			logEntry := logs.GetLogEntry(r)

			if logEntry != nil {
				logEntry.Sugar().Panic(rvr, debug.Stack())
			} else {
				fmt.Fprintf(os.Stderr, "Panic: %+v\n", rvr)
				debug.PrintStack()
			}

			err := &httperr.HTTPError{
				Code:    http.StatusInternalServerError,
				Message: http.StatusText(http.StatusInternalServerError),
			}
			httperr.HandleError(err, w, r)
		}
	}()

	return nil, nil
}
