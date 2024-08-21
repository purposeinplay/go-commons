package psqldocker

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"time"

	// import for side effects.
	_ "github.com/lib/pq"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

// ensure Container implements the io.Closer interface.
var _ io.Closer = (*Container)(nil)

// Container represents a Docker container
// running a PostgreSQL image.
type Container struct {
	user,
	password,
	dbName string
	dbPort   string
	hostPort string
	sqls     []string

	runOptions       *dockertest.RunOptions
	expiration       uint
	res              *dockertest.Resource
	pool             *dockertest.Pool
	poolEndpoint     string
	pingRetryTimeout time.Duration
}

// Start starts the docker container.
func (c *Container) Start() error {
	pool, err := newPool(c.pool, c.poolEndpoint, c.pingRetryTimeout)
	if err != nil {
		return err
	}

	res, err := startContainer(pool, c.runOptions)
	if err != nil {
		return fmt.Errorf("start container: %w", err)
	}

	c.res = res

	// set expiration
	_ = res.Expire(c.expiration)

	c.hostPort = c.res.GetPort(c.dbPort + "/tcp")

	err = pool.Retry(
		func() error {
			return pingDB(
				c.user,
				c.password,
				c.dbName,
				c.hostPort,
			)
		})
	if err != nil {
		_ = res.Close()

		return fmt.Errorf("ping db: %w", err)
	}

	err = executeSQLs(c.user, c.password, c.dbName, c.hostPort, c.sqls)
	if err != nil {
		_ = res.Close()

		return fmt.Errorf("execute sqls: %w", err)
	}

	return nil
}

// Port returns the container host hostPort mapped
// to the database running inside it.
func (c *Container) Port() string {
	return c.hostPort
}

// Close removes the Docker container.
func (c *Container) Close() error {
	return c.res.Close()
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
		user:     user,
		password: password,
		dbName:   dbName,
		dbPort:   options.dbPort,
		sqls:     options.sqls,
		runOptions: &dockertest.RunOptions{
			Name:         options.containerName,
			Cmd:          []string{"-p " + options.dbPort},
			Repository:   "postgres",
			Tag:          options.imageTag,
			ExposedPorts: []string{options.dbPort},
			Env:          envVars(user, password, dbName),
		},
		expiration:       options.expirationSeconds,
		pool:             options.pool,
		poolEndpoint:     options.poolEndpoint,
		pingRetryTimeout: options.pingRetryTimeout,
	}
}

func startContainer(
	pool *dockertest.Pool,
	runOptions *dockertest.RunOptions,
) (*dockertest.Resource, error) {
	return pool.RunWithOptions(
		runOptions,
		func(config *docker.HostConfig) {
			config.AutoRemove = true
			config.RestartPolicy = docker.RestartPolicy{
				Name: "no",
			}
		},
	)
}

// ErrWithPoolAndWithPoolEndpoint is returned when both
// WithPool and WithPoolEndpoint options are given to the
// NewContainer constructor.
var ErrWithPoolAndWithPoolEndpoint = errors.New(
	"with pool and with pool endpoint are mutually exclusive",
)

func newPool(
	pool *dockertest.Pool,
	poolEndpoint string,
	pingRetryTimeout time.Duration,
) (*dockertest.Pool, error) {
	if pool != nil && poolEndpoint != "" {
		return nil, ErrWithPoolAndWithPoolEndpoint
	}

	if pool != nil {
		pool.MaxWait = pingRetryTimeout

		return pool, nil
	}

	p, err := dockertest.NewPool(poolEndpoint)
	if err != nil {
		return nil, fmt.Errorf("new pool: %w", err)
	}

	p.MaxWait = pingRetryTimeout

	return p, nil
}

func envVars(
	user,
	password,
	dbName string,
) []string {
	return []string{
		"POSTGRES_PASSWORD=" + password,
		"POSTGRES_USER=" + user,
		"POSTGRES_DB=" + dbName,
	}
}

func pingDB(
	user,
	password,
	dbName,
	port string,
) error {
	db, err := sql.Open("postgres", fmt.Sprintf(
		"user=%s "+
			"password=%s "+
			"dbname=%s "+
			"host=localhost "+
			"port=%s "+
			"sslmode=disable",
		user,
		password,
		dbName,
		port))
	if err != nil {
		return fmt.Errorf("sql open: %w", err)
	}

	defer func() {
		_ = db.Close()
	}()

	err = db.Ping()
	if err != nil {
		return fmt.Errorf("ping: %w", err)
	}

	return nil
}

func executeSQLs(
	user,
	password,
	dbName,
	hostPort string,
	sqls []string,
) error {
	db, err := sql.Open(
		"postgres",
		fmt.Sprintf(
			"user=%s "+
				"password=%s "+
				"dbname=%s "+
				"host=localhost "+
				"port=%s "+
				"sslmode=disable",
			user,
			password,
			dbName,
			hostPort),
	)
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
