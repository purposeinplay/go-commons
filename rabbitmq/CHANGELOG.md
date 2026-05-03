# Changelog — `go-commons/rabbitmq`

All notable changes to the `github.com/purposeinplay/go-commons/rabbitmq`
module are documented here.

## [rabbitmq/v0.0.3]

### Changed (breaking)

- **`ZapLogger`** — renamed to **`SlogLogger`**. Implements the same
  `watermill.LoggerAdapter` interface, but wraps `*slog.Logger` instead
  of `*zap.Logger`.
- **`NewZapLoggerAdapter(*zap.Logger, trace, debug bool)`** — renamed
  to **`NewSlogLoggerAdapter(*slog.Logger, trace, debug bool)`**.
- Internal call sites switched from `zap.Any(...)` to `slog.Any(...)`.

### Fixed

- `With(fields)` — fixed a latent bug where the loop reassigned
  `newLogger = l.log.With(...)` (rebasing on `l.log` each iteration)
  so all but the last field were dropped. Now uses `l.log.With(all
  fields...)` once.

### Removed (breaking)

- `go.uber.org/zap` dependency.

### Internal

- Module bumped to `go 1.22` (required for `log/slog`). CI Go runner
  bumped from `1.19` to `1.22`.
