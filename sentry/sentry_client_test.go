package sentry_test

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/matryer/is"
	"github.com/purposeinplay/go-commons/sentry"
)

func TestClient(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skipf("skipping sentry tests in ci")
	}

	i := is.New(t)

	ctx := context.Background()

	f, err := os.ReadFile(".sentry-dsn")
	i.NoErr(err)

	sentryDSN := string(f)

	rep, err := sentry.NewClient(
		sentryDSN,
		"testing",
		"testing",
		1,
	)
	i.NoErr(err)

	t.Cleanup(func() {
		err := rep.Close()
		if err != nil {
			t.Logf("close sentry err: %s", err)
		}
	})

	t.Run("ReportEvent", func(t *testing.T) {
		t.Parallel()

		i := is.New(t)

		err = rep.ReportEvent(
			ctx,
			"test",
		)
		i.NoErr(err)
	})

	t.Run("MonitorOp", func(t *testing.T) {
		t.Parallel()

		rep.MonitorOperation(
			ctx,
			"test_operation",
			"test/monitor_op",
			uuid.NewString(),
			func(context.Context) {
			},
		)
	})

	err = rep.Close()
	i.NoErr(err)
}
