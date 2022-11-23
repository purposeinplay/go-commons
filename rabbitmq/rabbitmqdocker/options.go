package rabbitmqdocker

import (
	"github.com/ory/dockertest/v3"
	"time"
)

type options struct {
	containerName,
	port,
	managementPort string
	pool       *dockertest.Pool
	expiration time.Duration
}

func defaultOptions() options {
	return options{
		containerName:  "go-rabbitmqdocker",
		port:           "5672",
		managementPort: "15672",
		pool:           nil,
		expiration:     20,
	}
}

// Option configures an BTC Node Docker.
type Option interface {
	apply(*options)
}

type containerNameOption string

func (c containerNameOption) apply(opts *options) {
	opts.containerName = string(c)
}

// WithContainerName configures the PSQL Container Name, if
// empty, a random one will be picked.
func WithContainerName(name string) Option {
	return containerNameOption(name)
}

type portOption string

func (c portOption) apply(opts *options) {
	opts.port = string(c)
}

// WithPort sets the port to bind the container, default 5432.
func WithPort(port string) Option {
	return portOption(port)
}

type poolOption struct {
	p *dockertest.Pool
}

func (p poolOption) apply(opts *options) {
	opts.pool = p.p
}

// WithPool sets the docker container pool.
func WithPool(pool *dockertest.Pool) Option {
	return poolOption{pool}
}

type expiration time.Duration

func (e expiration) apply(opts *options) {
	opts.expiration = time.Duration(e)
}
