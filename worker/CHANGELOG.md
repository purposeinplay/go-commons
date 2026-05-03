# Changelog — `go-commons/worker`

All notable changes to the `github.com/purposeinplay/go-commons/worker`
module are documented here.

## [worker/v0.0.2]

### Changed (breaking)

- **`amqpw.Options.Logger`** and **`amqpw.Adapter.Logger`** — now
  `*slog.Logger`, was `*zap.Logger`. When omitted, defaults to
  `slog.Default()`. All internal log call sites switched from
  `zap.Any(...)` field constructors to `slog.String/Any(...)`.
- **`amqpw.New`** — no longer returns an error from logger
  initialisation. Falls back to `slog.Default()` when no logger is
  supplied.

### Removed (breaking)

- `go.uber.org/zap` and `go-commons/logs` dependencies.

### Internal

- Module bumped to `go 1.24`.
