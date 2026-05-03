# Changelog — `go-commons/otel`

All notable changes to the `github.com/purposeinplay/go-commons/otel`
module are documented here.

## [otel/v0.0.17]

### Internal

- Bumped `go-commons/grpc` dependency from `v0.0.28` → `v0.0.31`. The
  `grpc.WithDebug` and `grpc.WithUnaryServerInterceptorLogger` options
  switched to `*slog.Logger` between those versions.
- Integration test no longer imports `go.uber.org/zap` or `ctxzap`.
  Switched to `slog.Default()` and `slog.InfoContext`.
- Pre-existing test bug fixed: `otel.Init(ctx, endpoint, "test-service")`
  was passing a string literal where a variadic `attribute.KeyValue` is
  required. Now passes `attribute.String("service.name", "test-service")`.
