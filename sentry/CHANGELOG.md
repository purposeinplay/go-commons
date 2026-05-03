# Changelog — `go-commons/sentry`

All notable changes to the `github.com/purposeinplay/go-commons/sentry`
module are documented here.

## [sentry/v0.0.5]

### Added

- **`Middleware(*Client) func(next http.Handler) http.Handler`** — chi-
  compatible HTTP middleware that recovers from panics, reports them to
  Sentry via `sentrygo.CurrentHub().Recover(rvr)`, and flushes the
  client's buffered events at the end of the request via
  `client.Close()`. Pair with `chi/middleware.Recoverer` (or
  equivalent) if you also need a 500 response written for the client.

  Logging on Close errors goes through `slog.Default()`.

## [sentry/v0.0.1] – [sentry/v0.0.4]

Existing `Client` with `NewClient`, `CaptureException`, `Close`, etc.
See git history for per-version changes.
