# go-commons

[![gitleaks](https://img.shields.io/badge/protected%20by-gitleaks-blue)](https://github.com/zricethezav/gitleaks-action)

Shared Go libraries for purposeinplay services. Each top-level
directory is its own Go module with its own `go.mod` and version tag,
so consumers depend only on the pieces they use and upgrade them
independently.

## Modules

### Logging & observability

- **[`gcpslog`](./gcpslog)** — `slog.Handler` that emits the field
  renames Google Cloud Logging expects (`level` → `severity`,
  `msg` → `message`). No factory wrappers; compose with `slog.New`.
- **[`otel`](./otel)** — OpenTelemetry bootstrap: tracing/log/metric
  exporters, meter providers, and a slog bridge.
- **[`sentry`](./sentry)** — thin wrapper around `getsentry/sentry-go`
  plus a chi recover-and-report middleware.
- **[`smartbear`](./smartbear)** — SmartBear/AlertSite error reporting
  helpers.

### HTTP & RPC

- **[`apigrpc`](./apigrpc)** — generated protobuf bindings for shared
  internal APIs.
- **[`auth`](./auth)** — gRPC server interceptors for JWT-based
  authentication and authorization.
- **[`grpc`](./grpc)** — opinionated wrapper around the gRPC ecosystem:
  production server bootstrap, gateway, OTEL/logging interceptors.
- **[`http`](./http)** — HTTP error types, structured response
  rendering, and a chi-compatible slog request logger.
- **[`httpserver`](./httpserver)** — production-ready HTTP server
  bootstrap including HMAC-signed cookies.

### Persistence

- **[`clickhousedocker`](./clickhousedocker)** — programmatic ClickHouse
  container for tests.
- **[`pagination`](./pagination)** — Relay-style cursor pagination for
  SQL and Redis.
- **[`psqldocker`](./psqldocker)** — programmatic PostgreSQL container
  for tests.
- **[`psqltest`](./psqltest)** — `httptest`-style helpers for testing
  services backed by PostgreSQL.
- **[`psqlutil`](./psqlutil)** — common PostgreSQL utilities: connect
  with retries, GORM slog adapter, error plugin.

### Messaging

- **[`kafkadocker`](./kafkadocker)** — Kafka cluster in containers for
  integration tests.
- **[`pubsub`](./pubsub)** — abstract publisher/subscriber interfaces
  with Kafka and Sarama implementations.
- **[`pubsublite`](./pubsublite)** — Google Cloud Pub/Sub Lite client
  wrapper.
- **[`rabbitmq`](./rabbitmq)** — RabbitMQ helpers and a Watermill
  `LoggerAdapter` over `*slog.Logger`.
- **[`worker`](./worker)** — background worker abstraction with AMQP,
  asynq, and in-memory adapters.

### Utilities

- **[`blockingqueue`](./blockingqueue)** — generic bounded blocking FIFO
  queue.
- **[`errors`](./errors)** — typed errors with HTTP status mapping,
  error codes, and structured details.
- **[`rand`](./rand)** — small random helpers (strings, integers).
- **[`uuid`](./uuid)** — UUIDv7 generation and canonical-string parse
  helpers.
- **[`value`](./value)** — big-number wrappers that persist as
  PostgreSQL `NUMERIC` and round-trip safely through JSON.

## Versioning

Every sub-directory is an independent Go module. Import the one you
need:

```go
import "github.com/purposeinplay/go-commons/<module>"
```

Each module is tagged separately as `<module>/v0.x.y`, so different
modules can be pinned and upgraded independently.

## Logging convention

All modules log through stdlib `log/slog`. There are no logger
factories in this repo — services compose their own:

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
```

Wire it into the components that need one
(`grpc.WithDebug(logger, ...)`, `worker/amqpw.Options{Logger: logger}`,
`http.NewStructuredLogger(logger)`, etc.).

For Google Cloud Logging severity rendering, wrap the handler with
`gcpslog.NewHandler` and add the service name via `.With`:

```go
logger := slog.New(gcpslog.NewHandler(os.Stdout, nil)).With("service", "myservice")
```

`zap` and the previous `logger`/`logs` wrappers were removed in
[#73](https://github.com/purposeinplay/go-commons/pull/73); pinned
consumers keep building until they migrate.
