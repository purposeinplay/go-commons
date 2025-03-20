package clickhousedocker_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/matryer/is"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/purposeinplay/go-commons/clickhousedocker"
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

		cont := clickhousedocker.NewContainer(
			user,
			password,
			dbName,
			clickhousedocker.WithContainerName(containerNameFromTest(t)),
			clickhousedocker.WithDBPort("9000"),
			clickhousedocker.WithPool(p),
			clickhousedocker.WithImageTag("latest"),
			clickhousedocker.WithPoolEndpoint(""),
			clickhousedocker.WithSQL(
				"CREATE TABLE users(id UInt32, name String) ENGINE = Memory",
			),
			clickhousedocker.WithPingRetryTimeout(20),
			clickhousedocker.WithExpiration(20),
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

		c := clickhousedocker.NewContainer(
			user,
			password,
			dbName,
			clickhousedocker.WithContainerName(containerNameFromTest(t)),
		)

		err := c.Start()
		i.NoErr(err)

		err = c.Close()
		i.NoErr(err)
	})

	t.Run("InvalidTagFormat", func(t *testing.T) {
		t.Parallel()

		i := is.New(t)

		c := clickhousedocker.NewContainer(
			user,
			password,
			dbName,
			clickhousedocker.WithImageTag("error:latest"),
		)

		err := c.Start()

		var dockerErr *docker.Error

		i.True(errors.As(err, &dockerErr))
		i.Equal(
			"invalid tag format",
			dockerErr.Message,
		)
	})

	t.Run("ProvideWithPoolAndWithPoolEndpoint", func(t *testing.T) {
		t.Parallel()

		i := is.New(t)

		c := clickhousedocker.NewContainer(
			user,
			password,
			dbName,
			clickhousedocker.WithPool(new(dockertest.Pool)),
			clickhousedocker.WithPoolEndpoint("endpoint"),
		)

		err := c.Start()

		i.True(errors.Is(
			err,
			clickhousedocker.ErrWithPoolAndWithPoolEndpoint,
		))
	})

	t.Run("InvalidPoolEndpointURL", func(t *testing.T) {
		t.Parallel()

		i := is.New(t)

		c := clickhousedocker.NewContainer(
			user,
			password,
			dbName,
			clickhousedocker.WithPoolEndpoint("://endpoint"),
		)

		err := c.Start()

		i.Equal(
			"new pool: invalid endpoint",
			err.Error(),
		)
	})

	t.Run("PingFail", func(t *testing.T) {
		t.Parallel()

		i := is.New(t)

		c := clickhousedocker.NewContainer(
			user,
			password,
			dbName,
			clickhousedocker.WithContainerName(containerNameFromTest(t)),
			clickhousedocker.WithPingRetryTimeout(1),
			clickhousedocker.WithImageTag("latest"),
		)

		err := c.Start()

		i.True(strings.Contains(err.Error(), "ping db: reached retry deadline"))
	})
}

func containerNameFromTest(t *testing.T) string {
	t.Helper()

	containerName := strings.Split(t.Name(), "/")

	return containerName[len(containerName)-1]
}
