package psqldocker

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	// import for side effects.
	_ "github.com/lib/pq"
	"github.com/moby/moby/api/types/container"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Container represents a Docker container
// running a PostgreSQL image.
type Container struct {
	user,
	password,
	dbName string
	dbPort         string
	host           string
	hostPort       string
	sqls           []string
	containerName  string
	imageTag       string
	startupTimeout time.Duration

	container testcontainers.Container
}

// Start starts the docker container.
func (c *Container) Start(ctx context.Context) error {
	portTCP := c.dbPort + "/tcp"

	req := testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Name:         c.containerName,
			Image:        "postgres:" + c.imageTag,
			Cmd:          []string{"-p", c.dbPort},
			ExposedPorts: []string{portTCP},
			Env: map[string]string{
				"POSTGRES_USER":     c.user,
				"POSTGRES_PASSWORD": c.password,
				"POSTGRES_DB":       c.dbName,
			},
			// The postgres entrypoint logs "database system is ready to accept
			// connections" once when initdb's temporary server boots and again
			// when the real server starts. Waiting for the second occurrence,
			// then confirming the port is listening, avoids flakiness on
			// macOS/Windows where Docker Desktop proxies ports asynchronously.
			WaitingFor: wait.ForAll(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2),
				wait.ForListeningPort(portTCP),
			).WithDeadline(c.startupTimeout),
			HostConfigModifier: func(cfg *container.HostConfig) {
				cfg.RestartPolicy = container.RestartPolicy{Name: "no"}
			},
		},
		Started: true,
	}

	ctr, err := testcontainers.GenericContainer(ctx, req)
	if err != nil {
		return fmt.Errorf("start container: %w", err)
	}

	c.container = ctr

	host, err := ctr.Host(ctx)
	if err != nil {
		_ = ctr.Terminate(ctx)

		return fmt.Errorf("host: %w", err)
	}

	c.host = host

	mapped, err := ctr.MappedPort(ctx, portTCP)
	if err != nil {
		_ = ctr.Terminate(ctx)

		return fmt.Errorf("mapped port: %w", err)
	}

	c.hostPort = mapped.Port()

	if err := executeSQLs(c.DSN(), c.sqls); err != nil {
		_ = ctr.Terminate(ctx)

		return fmt.Errorf("execute sqls: %w", err)
	}

	return nil
}

// Host returns the host the container is reachable on (usually "localhost").
func (c *Container) Host() string {
	return c.host
}

// Port returns the host port mapped to the database running inside the container.
func (c *Container) Port() string {
	return c.hostPort
}

// DSN returns a lib/pq-compatible connection string for the database.
// Only valid after Start has returned successfully.
func (c *Container) DSN() string {
	return fmt.Sprintf(
		"user=%s password=%s dbname=%s host=%s port=%s sslmode=disable",
		c.user, c.password, c.dbName, c.host, c.hostPort,
	)
}

// Close removes the Docker container. It is a no-op if Start has not been
// called or did not succeed.
func (c *Container) Close(ctx context.Context) error {
	if c.container == nil {
		return nil
	}

	return c.container.Terminate(ctx)
}

// NewContainer starts a new psql database in a docker container.
func NewContainer(
	user,
	password,
	dbName string,
	opts ...Option,
) *Container {
	options := defaultOptions()

	for i := range opts {
		opts[i].apply(&options)
	}

	return &Container{
		user:           user,
		password:       password,
		dbName:         dbName,
		dbPort:         options.dbPort,
		sqls:           options.sqls,
		containerName:  options.containerName,
		imageTag:       options.imageTag,
		startupTimeout: options.startupTimeout,
	}
}

func executeSQLs(dsn string, sqls []string) error {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}

	defer func() {
		_ = db.Close()
	}()

	for i := range sqls {
		_, err = db.Exec(sqls[i])
		if err != nil {
			return fmt.Errorf("execute sql %d: %w", i, err)
		}
	}

	return nil
}
