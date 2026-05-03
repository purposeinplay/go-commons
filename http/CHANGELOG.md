# Changelog — `go-commons/http`

All notable changes to the `github.com/purposeinplay/go-commons/http`
module are documented here.

## [http/v0.0.2]

### Added

- **`NewStructuredLogger(*slog.Logger)`** — chi `RequestLogger`
  middleware backed by stdlib `log/slog`. Emits `request started` /
  `request complete` events per request, decorated with method, path,
  request ID, scheme, proto, remote addr, user agent, and full URI.
  Pairs with the existing zap-based `go-commons/logs` for projects
  ready to migrate to slog.
- **`StructuredLoggerEntry`** — slog-backed per-request log entry,
  with `Write` / `Panic` / `WithError` callbacks for chi.
- **`GetSlogEntry(*http.Request)`** — fetch the current request's
  slog-based entry. Distinct name from the existing zap-based
  `GetLogEntry` so consumers in mid-migration can tell them apart.

### Internal

- Module bumped to `go 1.24` (required for `log/slog`).

## [http/v0.0.1]

Initial release of the `http` subpackage with `OAuthError`,
`HTTPError`, response builders (`BadRequestError`, etc.),
`HandleError`, plus the `render` and `router` sub-subpackages.
