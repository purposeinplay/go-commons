# Changelog — `go-commons/psqlutil`

All notable changes to the `github.com/purposeinplay/go-commons/psqlutil`
module are documented here.

## [psqlutil/v0.0.19]

### Removed (breaking)

- **`NewZapLogger(*zap.Logger)`** — deleted. Use the existing
  **`NewSlogLogger(*slog.Logger)`** instead. No external callers found
  inside the org at time of removal.
- `go.uber.org/zap` and `moul.io/zapgorm2` dependencies.
