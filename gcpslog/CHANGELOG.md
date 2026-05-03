# Changelog — `go-commons/gcpslog`

All notable changes to the `github.com/purposeinplay/go-commons/gcpslog`
module are documented here.

## [gcpslog/v0.0.1]

Initial release.

### Added

- **`NewHandler(io.Writer, *slog.HandlerOptions) slog.Handler`** —
  wraps `slog.NewJSONHandler` with the field renames Google Cloud
  Logging expects: `level` → `severity` (mapping
  DEBUG/INFO/WARNING/ERROR/CRITICAL from the slog level), `msg` →
  `message`. Composes with stdlib slog primitives — no factory
  wrappers, no options struct. Service name is added by the caller via
  `slog.Logger.With("service", name)`.

  Replaces the deleted `go-commons/logger` package, which wrapped
  `blendle/zapdriver` to produce a Stackdriver-formatted `*zap.Logger`.
  Consumers compose with `slog.New` directly:

  ```go
  logger := slog.New(gcpslog.NewHandler(os.Stdout, nil)).With("service", "myservice")
  ```
