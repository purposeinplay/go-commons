package rabbitmqdocker

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

// NewContainer creates a new Rabbit MQ container.
func NewContainer(
	user,
	pass string,
	opts ...Option,
) (*dockertest.Resource, error) {
	options := defaultOptions()

	for _, o := range opts {
		o.apply(&options)
	}

	portBindings := ports(options)

	pool, err := pool(options)
	if err != nil {
		return nil, fmt.Errorf("new pool: %w", err)
	}

	// create run options
	dockerRunOptions := &dockertest.RunOptions{
		Hostname:     options.containerName,
		Name:         options.containerName,
		Repository:   "rabbitmq",
		Tag:          "management-alpine",
		PortBindings: portBindings,
		Env:          envVars(user, pass),
	}

	res, err := startContainer(pool, dockerRunOptions, func() error {
		return ping(user, pass, options.managementPort)
	})
	if err != nil {
		return nil, fmt.Errorf("start container: %w", err)
	}

	// set expiration
	_ = res.Expire(uint(options.expiration.Seconds()))

	return res, nil
}

func ports(opts options) (portBindings map[docker.Port][]docker.PortBinding) {
	pB := map[docker.Port][]docker.PortBinding{
		docker.Port(opts.port + "/tcp"): {
			{HostIP: "0.0.0.0", HostPort: opts.port},
		},
		docker.Port(opts.managementPort + "/tcp"): {
			{HostIP: "0.0.0.0", HostPort: opts.managementPort},
		},
	}

	return pB
}

func envVars(user, password string) []string {
	return []string{
		fmt.Sprintf("RABBITMQ_DEFAULT_PASS=%s", password),
		fmt.Sprintf("RABBITMQ_DEFAULT_USER=%s", user),
	}
}

func startContainer(
	pool *dockertest.Pool,
	runOptions *dockertest.RunOptions,
	retryFunc func() error,
) (*dockertest.Resource, error) {
	res, err := pool.RunWithOptions(
		runOptions,
		func(config *docker.HostConfig) {
			config.AutoRemove = true
		},
	)
	if err != nil {
		return nil, fmt.Errorf("docker run: %w", err)
	}

	err = pool.Retry(retryFunc)
	if err != nil {
		_ = res.Close()

		return nil, fmt.Errorf("ping node: %w", err)
	}

	return res, nil
}

func pool(opts options) (*dockertest.Pool, error) {
	pool := opts.pool

	if pool == nil {
		p, err := dockertest.NewPool("")
		if err != nil {
			return nil, fmt.Errorf("new pool: %w", err)
		}

		p.MaxWait = 40 * time.Second

		pool = p
	}

	return pool, nil
}

// ErrInvalidStatus is returned whenever the aliveness message retruns
// a not ok status.
var ErrInvalidStatus = errors.New("invalid status code")

func ping(
	user,
	pass,
	port string,
) error {
	const urlFormat = "http://%s:%s@localhost:%s/api/aliveness-test/%%2F"

	url := fmt.Sprintf(
		urlFormat,
		user,
		pass,
		port,
	)

	// nolint: gosec // G107: reports http request made with variable url
	// but this is only used for testing.
	res, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("get request: %w", err)
	}

	_ = res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return ErrInvalidStatus
	}

	return nil
}
