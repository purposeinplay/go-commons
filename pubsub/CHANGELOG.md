# Changelog — `go-commons/pubsub`

All notable changes to the `github.com/purposeinplay/go-commons/pubsub`
module are documented here.

## [pubsub/v0.0.27]

### Changed (breaking)

- **`kafka.NewSubscriber(logger *slog.Logger, ...)`** — was
  `*zap.Logger`.
- **`kafka.NewPublisher(logger *slog.Logger, ...)`** — was
  `*zap.Logger`.
- Internal `zapLogger` watermill adapter rewritten as a private
  `slogLogger` wrapping `*slog.Logger`. Same `watermill.LoggerAdapter`
  interface; consumers that only used `NewSubscriber`/`NewPublisher`
  see only the constructor signature change.
- `kafkasarama` — production code was already on slog. Tests dropped
  the `zap.NewExample` + `zapslog.NewHandler` bridge in favour of
  `slog.Default()`.

### Fixed

- kafka adapter `With(fields)` had the same bug as rabbitmq's adapter
  — only the last field was retained. Now applies all fields in one
  `slog.Logger.With(...)` call.

### Removed (breaking)

- `go.uber.org/zap` and `go.uber.org/zap/exp` dependencies.
