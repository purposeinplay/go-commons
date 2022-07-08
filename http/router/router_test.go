package router_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	commonhttp "github.com/purposeinplay/go-commons/http"
	"github.com/purposeinplay/go-commons/http/router"
	"go.uber.org/zap"
)

func TestHandlerErrorFunc(t *testing.T) {
	mux := router.New(router.WithLogger(zap.NewExample()))

	mux.Get("/test", router.HandlerErrorFunc(func(w http.ResponseWriter, r *http.Request) error {
		if r.Header.Get("test") != "yes" {
			return commonhttp.BadRequestError("invalid test header")
		}

		return nil
	}).ServeHTTP)

	t.Parallel()

	t.Run("Error", func(t *testing.T) {
		t.Parallel()

		r := httptest.NewRequest(http.MethodGet, "/test", nil)
		rr := httptest.NewRecorder()

		mux.ServeHTTP(rr, r)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("invalid status code, expected: 400, received: %d", rr.Code)
		}
	})

	t.Run("NoError", func(t *testing.T) {
		t.Parallel()

		r := httptest.NewRequest(http.MethodGet, "/test", nil)

		r.Header.Set("test", "yes")

		rr := httptest.NewRecorder()

		mux.ServeHTTP(rr, r)

		if rr.Code != http.StatusOK {
			t.Errorf("invalid status code, expected: 200, received: %d", rr.Code)
		}
	})
}
