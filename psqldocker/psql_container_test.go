package psqldocker_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/lib/pq"
	"github.com/matryer/is"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
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

		p, err := dockertest.NewPool("")
		i.NoErr(err)

		cont := psqldocker.NewContainer(
			user,
			password,
			dbName,
			psqldocker.WithContainerName(containerNameFromTest(t)),
			psqldocker.WithDBPort("5432"),
			psqldocker.WithPool(p),
			psqldocker.WithImageTag("alpine"),
			psqldocker.WithPoolEndpoint(""),
			psqldocker.WithSQL(
				"CREATE TABLE users(user_id UUID PRIMARY KEY);",
			),
			psqldocker.WithPingRetryTimeout(20),
			psqldocker.WithExpiration(20),
		)

		err = cont.Start()
		i.NoErr(err)

		t.Logf("container started on hostPort: %s", cont.Port())

		err = cont.Close()
		i.NoErr(err)
	})

	t.Run("NoOptions", func(t *testing.T) {
		t.Parallel()

		i := is.New(t)

		c := psqldocker.NewContainer(
			user,
			password,
			dbName,
			psqldocker.WithContainerName(containerNameFromTest(t)),
		)

		err := c.Start()
		i.NoErr(err)

		err = c.Close()
		i.NoErr(err)
	})

	t.Run("InvalidTagFormat", func(t *testing.T) {
		t.Parallel()

		i := is.New(t)

		c := psqldocker.NewContainer(
			user,
			password,
			dbName,
			psqldocker.WithImageTag("error:latest"),
		)

		err := c.Start()

		var dockerErr *docker.Error

		i.True(errors.As(err, &dockerErr))
		i.Equal(
			"invalid tag format",
			dockerErr.Message,
		)
	})

	t.Run("InvalidSQL", func(t *testing.T) {
		t.Parallel()

		i := is.New(t)

		c := psqldocker.NewContainer(
			user,
			password,
			dbName,
			psqldocker.WithContainerName(containerNameFromTest(t)),
			psqldocker.WithSQL("error"),
		)

		err := c.Start()

		var pqErr *pq.Error

		i.True(errors.As(err, &pqErr))
		i.Equal(
			"syntax error at or near \"error\"",
			pqErr.Message,
		)
	})

	t.Run("ProvideWithPoolAndWithPoolEndpoint", func(t *testing.T) {
		t.Parallel()

		i := is.New(t)

		c := psqldocker.NewContainer(
			user,
			password,
			dbName,
			psqldocker.WithPool(new(dockertest.Pool)),
			psqldocker.WithPoolEndpoint("endpoint"),
		)

		err := c.Start()

		i.True(errors.Is(
			err,
			psqldocker.ErrWithPoolAndWithPoolEndpoint,
		))
	})

	t.Run("InvalidPoolEndpointURL", func(t *testing.T) {
		t.Parallel()

		i := is.New(t)

		c := psqldocker.NewContainer(
			user,
			password,
			dbName,
			psqldocker.WithPoolEndpoint("://endpoint"),
		)

		err := c.Start()

		i.Equal(
			"new pool: : invalid endpoint",
			err.Error(),
		)
	})

	t.Run("PingFail", func(t *testing.T) {
		t.Parallel()

		i := is.New(t)

		c := psqldocker.NewContainer(
			user,
			password,
			dbName,
			psqldocker.WithContainerName(containerNameFromTest(t)),
			psqldocker.WithPingRetryTimeout(1),
		)

		err := c.Start()

		i.Equal(
			"ping db: reached retry deadline",
			err.Error(),
		)
	})
}

func containerNameFromTest(t *testing.T) string {
	t.Helper()

	containerName := strings.Split(t.Name(), "/")

	return containerName[len(containerName)-1]
}
