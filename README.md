# LogLayer for Go

[![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev.svg)](https://pkg.go.dev/go.loglayer.dev)
[![CI](https://github.com/loglayer/loglayer-go/actions/workflows/ci.yml/badge.svg)](https://github.com/loglayer/loglayer-go/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

A transport-agnostic structured logging facade for Go. Write your application code against one fluent API; pick (or swap) the underlying logger as a configuration choice.

```go
log.
    WithFields(loglayer.Fields{"requestId": "abc-123"}).
    WithMetadata(loglayer.Metadata{"durationMs": 42}).
    WithError(err).
    Error("request failed")
```

Same call shape whether the backend is `zerolog`, `zap`, `slog`, `logrus`, `phuslu/log`, `charmbracelet/log`, a JSON writer, an HTTP endpoint, or your own transport.

## Install

```sh
go get go.loglayer.dev
```

## Quickstart

```go
package main

import (
    "go.loglayer.dev"
    "go.loglayer.dev/transports/structured"
)

func main() {
    log := loglayer.New(loglayer.Config{
        Transport: structured.New(structured.Config{}),
    })

    log.Info("hello world")
    log.WithFields(loglayer.Fields{"userId": 42}).Info("user logged in")
}
```

```json
{"level":"info","time":"2026-04-26T12:00:00Z","msg":"hello world"}
{"level":"info","time":"2026-04-26T12:00:00Z","msg":"user logged in","userId":42}
```

For local development, swap `structured` for `pretty` to get colorized terminal output.

## Why use it

- **Transport-agnostic facade.** Swap zerolog for zap (or any other backend) without touching application code.
- **Multi-transport fan-out.** Send the same entry to `pretty` in dev *and* `structured` to a file in prod. One log call, two destinations.
- **Three-way data separation.** `WithFields` for persistent context, `WithMetadata` for per-call payloads, `WithCtx` for `context.Context`. The convention you reach for matches the lifetime of the data.
- **Type-flexible metadata.** Pass a `map`, a struct, a slice, or any value to `WithMetadata`. The transport decides serialization.
- **Thread-safe by design.** Every method on `*LogLayer` is safe to call concurrently with emission, including runtime level toggling and transport hot-swap. No "setup-only" carve-outs.
- **First-class testing.** `loglayer.NewMock()` for silent test mocks; `transports/testing` for capture-and-assert.

## Transports

Renderers (self-contained):

- [`pretty`](https://go.loglayer.dev/transports/pretty) — colorized terminal output (recommended for local dev)
- [`structured`](https://go.loglayer.dev/transports/structured) — one JSON object per entry (recommended for production)
- [`console`](https://go.loglayer.dev/transports/console) — plain `fmt.Println`-style output
- [`testing`](https://go.loglayer.dev/transports/testing) — in-memory capture for tests
- [`blank`](https://go.loglayer.dev/transports/blank) — bring-your-own function transport

Network:

- [`http`](https://go.loglayer.dev/transports/http) — generic batched HTTP POST with a pluggable encoder
- [`datadog`](https://go.loglayer.dev/transports/datadog) — Datadog Logs HTTP intake (US/EU/AP regions)

Logger wrappers:

- [`zerolog`](https://go.loglayer.dev/transports/zerolog) wraps [github.com/rs/zerolog](https://github.com/rs/zerolog)
- [`zap`](https://go.loglayer.dev/transports/zap) wraps [go.uber.org/zap](https://github.com/uber-go/zap)
- [`slog`](https://go.loglayer.dev/transports/slog) wraps the stdlib `*log/slog.Logger`
- [`logrus`](https://go.loglayer.dev/transports/logrus) wraps [github.com/sirupsen/logrus](https://github.com/sirupsen/logrus)
- [`phuslu`](https://go.loglayer.dev/transports/phuslu) wraps [github.com/phuslu/log](https://github.com/phuslu/log)
- [`charmlog`](https://go.loglayer.dev/transports/charmlog) wraps [github.com/charmbracelet/log](https://github.com/charmbracelet/log)

Writing your own is [a single interface](https://go.loglayer.dev/transports/creating-transports) with four methods.

## Integrations

- [`loghttp`](https://go.loglayer.dev/integrations/loghttp) — HTTP middleware that derives a per-request logger with `requestId`/`method`/`path`, stores it in the request context, and emits a "request completed" log with status and duration. One line at server setup.

## Examples

Runnable demos under [`examples/`](./examples):

- [`http-server`](./examples/http-server) — `loghttp` middleware in an HTTP handler
- [`multi-transport`](./examples/multi-transport) — pretty in dev + structured to file with per-transport level filtering
- [`custom-transport`](./examples/custom-transport) — implementing the Transport interface from scratch
- [`datadog-shipping`](./examples/datadog-shipping) — Datadog Logs intake with tuned batching

Run any with `go run ./examples/<name>`.

## Documentation

Full docs at **[go.loglayer.dev](https://go.loglayer.dev)**.

- [Getting Started](https://go.loglayer.dev/getting-started)
- [Configuration](https://go.loglayer.dev/configuration) — every Config field
- [Cheat Sheet](https://go.loglayer.dev/cheatsheet) — one-page API reference
- [Logging API](https://go.loglayer.dev/logging-api/basic-logging) — per-method guides
- [Transports](https://go.loglayer.dev/transports/) — overview and per-transport pages
- [Benchmarks](https://go.loglayer.dev/benchmarks)

## TypeScript counterpart

Already using [loglayer for TypeScript](https://loglayer.dev)? The Go port keeps the same mental model: `WithFields` for persistent context, `WithMetadata` for per-call payloads, transport-agnostic facade. Method names map directly. The biggest Go-specific divergence is the `Context` → `Fields` rename to avoid colliding with `context.Context`.

## Contributing

See [AGENTS.md](AGENTS.md) for project structure, coding conventions, and the verification workflow. Pre-commit and pre-push hooks are managed by [lefthook](https://github.com/evilmartians/lefthook); install with `go install github.com/evilmartians/lefthook@latest && lefthook install`.

## License

[MIT](LICENSE)
