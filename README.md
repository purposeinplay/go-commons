# go-commons
[![gitleaks](https://img.shields.io/badge/protected%20by-gitleaks-blue)](https://github.com/zricethezav/gitleaks-action)

This is a core library that will add common features for our services.

Mostly this deals with configuring logging, messaging (rabbitmq), and loading configuration.

## grpc
[![lint-test](https://github.com/purposeinplay/go-commons/actions/workflows/lint-test_grpc.yml/badge.svg)](https://github.com/purposeinplay/go-commons/actions/workflows/lint-test_grpc.yml?query=workflow%3ALint+%26+Test+grpc+)
[![lint-test](https://github.com/purposeinplay/go-commons/actions/workflows/codeql_grpc.yaml/badge.svg)](https://github.com/purposeinplay/go-commons/actions/workflows/codeql_grpc.yaml?query=workflow%3A%22CodeQL+grpc%22++)
[![lint-test](https://github.com/purposeinplay/go-commons/actions/workflows/grype_grpc.yaml/badge.svg)](https://github.com/purposeinplay/go-commons/actions/workflows/grype_grpc.yaml?query=workflow%3A%22Grype+grpc%22)
---
## httpserver
[![lint-test](https://github.com/purposeinplay/go-commons/actions/workflows/lint-test_httpserver.yml/badge.svg)](https://github.com/purposeinplay/go-commons/actions/workflows/lint-test_httpserver.yml?query=workflow%3ALint+%26+Test+grpc+)
[![lint-test](https://github.com/purposeinplay/go-commons/actions/workflows/codeql_httpserver.yaml/badge.svg)](https://github.com/purposeinplay/go-commons/actions/workflows/codeql_httpserver.yaml?query=workflow%3A%22CodeQL+grpc%22++)
[![lint-test](https://github.com/purposeinplay/go-commons/actions/workflows/grype_httpserver.yaml/badge.svg)](https://github.com/purposeinplay/go-commons/actions/workflows/grype_httpserver.yaml?query=workflow%3A%22Grype+grpc%22)
---
## gcpslog
A `slog.Handler` that emits JSON in the shape Google Cloud Logging
expects: `level` → `severity` (DEBUG/INFO/WARNING/ERROR/CRITICAL), `msg`
→ `message`. No factory wrappers — compose with stdlib `slog.New`.

```go
logger := slog.New(gcpslog.NewHandler(os.Stdout, nil)).With("service", "myservice")
```

Replaces the deprecated `logger` and `logs` packages, which wrapped
[`zap`](https://github.com/uber-go/zap). Services should use stdlib
`log/slog` directly; reach for `gcpslog` only when GCP Cloud Logging
severity rendering matters.
---
## psqltest
[![lint-test](https://github.com/purposeinplay/go-commons/actions/workflows/lint-test_psqltest.yml/badge.svg)](https://github.com/purposeinplay/go-commons/actions/workflows/lint-test_psqltest.yml?query=workflow%3ALint+%26+Test+grpc+)
[![lint-test](https://github.com/purposeinplay/go-commons/actions/workflows/codeql_psqltest.yaml/badge.svg)](https://github.com/purposeinplay/go-commons/actions/workflows/codeql_psqltest.yaml?query=workflow%3A%22CodeQL+grpc%22++)
[![lint-test](https://github.com/purposeinplay/go-commons/actions/workflows/grype_psqltest.yaml/badge.svg)](https://github.com/purposeinplay/go-commons/actions/workflows/grype_psqltest.yaml?query=workflow%3A%22Grype+grpc%22)
---
## pubsub
[![lint-test](https://github.com/purposeinplay/go-commons/actions/workflows/lint-test_pubsub.yml/badge.svg)](https://github.com/purposeinplay/go-commons/actions/workflows/lint-test_pubsub.yml?query=workflow%3ALint+%26+Test+grpc+)
[![lint-test](https://github.com/purposeinplay/go-commons/actions/workflows/codeql_pubsub.yaml/badge.svg)](https://github.com/purposeinplay/go-commons/actions/workflows/codeql_pubsub.yaml?query=workflow%3A%22CodeQL+grpc%22++)
[![lint-test](https://github.com/purposeinplay/go-commons/actions/workflows/grype_pubsub.yaml/badge.svg)](https://github.com/purposeinplay/go-commons/actions/workflows/grype_pubsub.yaml?query=workflow%3A%22Grype+grpc%22)
---
## sentry
[![lint-test](https://github.com/purposeinplay/go-commons/actions/workflows/lint-test_sentry.yml/badge.svg)](https://github.com/purposeinplay/go-commons/actions/workflows/lint-test_sentry.yml?query=workflow%3ALint+%26+Test+grpc+)
[![lint-test](https://github.com/purposeinplay/go-commons/actions/workflows/codeql_sentry.yaml/badge.svg)](https://github.com/purposeinplay/go-commons/actions/workflows/codeql_sentry.yaml?query=workflow%3A%22CodeQL+grpc%22++)
[![lint-test](https://github.com/purposeinplay/go-commons/actions/workflows/grype_sentry.yaml/badge.svg)](https://github.com/purposeinplay/go-commons/actions/workflows/grype_sentry.yaml?query=workflow%3A%22Grype+grpc%22)
---
## value
[![lint-test](https://github.com/purposeinplay/go-commons/actions/workflows/lint-test_value.yml/badge.svg)](https://github.com/purposeinplay/go-commons/actions/workflows/lint-test_value.yml?query=workflow%3ALint+%26+Test+grpc+)
[![lint-test](https://github.com/purposeinplay/go-commons/actions/workflows/codeql_value.yaml/badge.svg)](https://github.com/purposeinplay/go-commons/actions/workflows/codeql_value.yaml?query=workflow%3A%22CodeQL+grpc%22++)
[![lint-test](https://github.com/purposeinplay/go-commons/actions/workflows/grype_value.yaml/badge.svg)](https://github.com/purposeinplay/go-commons/actions/workflows/grype_value.yaml?query=workflow%3A%22Grype+grpc%22)
---