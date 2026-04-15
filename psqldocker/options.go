package psqldocker

import (
	"time"
)

type options struct {
	containerName,
	imageTag,
	dbPort string
	sqls           []string
	startupTimeout time.Duration
}

func defaultOptions() options {
	return options{
		containerName:  "go-psqldocker",
		imageTag:       "16-alpine",
		dbPort:         "5432",
		sqls:           nil,
		startupTimeout: 20 * time.Second,
	}
}

// Option configures a PSQL Docker container.
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

type imageTagOption string

func (t imageTagOption) apply(opts *options) {
	opts.imageTag = string(t)
}

// WithImageTag configures the PSQL Container image tag, default: 16-alpine.
func WithImageTag(tag string) Option {
	return imageTagOption(tag)
}

type sqlOption string

func (c sqlOption) apply(opts *options) {
	opts.sqls = append(opts.sqls, string(c))
}

// WithSQL specifies a sqls file, to initiate the
// db with.
func WithSQL(sql string) Option {
	return sqlOption(sql)
}

type dbPortOption string

func (c dbPortOption) apply(opts *options) {
	opts.dbPort = string(c)
}

// WithDBPort sets the database port running in the container, default 5432.
func WithDBPort(port string) Option {
	return dbPortOption(port)
}

type startupTimeoutOption time.Duration

func (s startupTimeoutOption) apply(opts *options) {
	opts.startupTimeout = time.Duration(s)
}

// WithStartupTimeout sets the container startup timeout in seconds, i.e.
// how long to wait for PostgreSQL to become ready to accept connections.
// Default: 20 seconds.
func WithStartupTimeout(seconds uint) Option {
	return startupTimeoutOption(time.Duration(seconds) * time.Second)
}
