package psqldocker_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/lib/pq"
	"github.com/matryer/is"
	"github.com/purposeinplay/go-commons/psqldocker"
)

func TestNewContainer(t *testing.T) {
	t.Parallel()

	const (
		user     = "user"
		password = "pass"
		dbName   = "test"
	)

	t.Run("AllOptions", func(t *testing.T) {
		t.Parallel()

		i := is.New(t)
		ctx := context.Background()

		cont := psqldocker.NewContainer(
			user,
			password,
			dbName,
			psqldocker.WithContainerName(containerNameFromTest(t)),
			psqldocker.WithDBPort("5432"),
			psqldocker.WithImageTag("alpine"),
			psqldocker.WithSQL(
				"CREATE TABLE users(user_id UUID PRIMARY KEY);",
			),
			psqldocker.WithStartupTimeout(20),
		)

		err := cont.Start(ctx)
		i.NoErr(err)

		t.Logf("container started on hostPort: %s", cont.Port())

		err = cont.Close(ctx)
		i.NoErr(err)
	})

	t.Run("NoOptions", func(t *testing.T) {
		t.Parallel()

		i := is.New(t)
		ctx := context.Background()

		c := psqldocker.NewContainer(
			user,
			password,
			dbName,
			psqldocker.WithContainerName(containerNameFromTest(t)),
		)

		err := c.Start(ctx)
		i.NoErr(err)

		err = c.Close(ctx)
		i.NoErr(err)
	})

	t.Run("CustomDBPort", func(t *testing.T) {
		t.Parallel()

		i := is.New(t)
		ctx := context.Background()

		c := psqldocker.NewContainer(
			user,
			password,
			dbName,
			psqldocker.WithContainerName(containerNameFromTest(t)),
			psqldocker.WithDBPort("5433"),
		)

		err := c.Start(ctx)
		i.NoErr(err)

		t.Logf("container started on hostPort: %s", c.Port())

		err = c.Close(ctx)
		i.NoErr(err)
	})

	t.Run("CloseBeforeStart", func(t *testing.T) {
		t.Parallel()

		i := is.New(t)
		ctx := context.Background()

		c := psqldocker.NewContainer(user, password, dbName)

		// Must not panic; should be a no-op.
		err := c.Close(ctx)
		i.NoErr(err)
	})

	t.Run("InvalidTagFormat", func(t *testing.T) {
		t.Parallel()

		i := is.New(t)
		ctx := context.Background()

		c := psqldocker.NewContainer(
			user,
			password,
			dbName,
			psqldocker.WithImageTag("error:latest"),
		)

		err := c.Start(ctx)
		i.True(err != nil)
	})

	t.Run("InvalidSQL", func(t *testing.T) {
		t.Parallel()

		i := is.New(t)
		ctx := context.Background()

		c := psqldocker.NewContainer(
			user,
			password,
			dbName,
			psqldocker.WithContainerName(containerNameFromTest(t)),
			psqldocker.WithSQL("error"),
		)

		err := c.Start(ctx)

		var pqErr *pq.Error

		i.True(errors.As(err, &pqErr))
		i.Equal(
			"syntax error at or near \"error\"",
			pqErr.Message,
		)
	})

	t.Run("PingFail", func(t *testing.T) {
		t.Parallel()

		i := is.New(t)
		ctx := context.Background()

		c := psqldocker.NewContainer(
			user,
			password,
			dbName,
			psqldocker.WithContainerName(containerNameFromTest(t)),
			psqldocker.WithStartupTimeout(1),
		)

		err := c.Start(ctx)

		i.True(err != nil)
		i.True(strings.Contains(err.Error(), "start container"))
	})
}

func containerNameFromTest(t *testing.T) string {
	t.Helper()

	containerName := strings.Split(t.Name(), "/")

	return containerName[len(containerName)-1]
}
