# Changelog — `go-commons/auth`

All notable changes to the `github.com/purposeinplay/go-commons/auth`
module are documented here.

## [auth/v0.0.5]

### Changed (breaking)

- **`NewAuthInterceptor(logger *slog.Logger, ...)`** — was
  `*zap.Logger`.
- **`NewAuthorizerInterceptor(logger *slog.Logger, ...)`** — was
  `*zap.Logger`.
- Internal logging switched from `zap.Error(err)` to
  `slog.Any("error", err)`.

### Removed (breaking)

- `go.uber.org/zap` dependency.
