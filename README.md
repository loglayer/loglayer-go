<p align="center">
  <a href="https://go.loglayer.dev" title="LogLayer for Go">
    <img src="docs/src/public/images/loglayer.png" alt="LogLayer logo by Akshaya Madhavan" width="200">
  </a>
</p>

# LogLayer for Go

<p align="center">
  <a href="https://pkg.go.dev/go.loglayer.dev"><img src="https://pkg.go.dev/badge/go.loglayer.dev.svg" alt="Go Reference"></a>
  <a href="https://github.com/loglayer/loglayer-go/actions/workflows/ci.yml"><img src="https://github.com/loglayer/loglayer-go/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License: MIT"></a>
</p>

`loglayer-go` is a structured logging library for Go. Use it standalone with the built-in JSON, pretty terminal, HTTP, or cloud-service transports (Datadog, etc.) — or wrap an existing logger like zerolog, zap, slog, or logrus when you've already invested in one. Either way, application code uses one fluent API for messages, fields, metadata, and errors.

For full documentation, read the [docs](https://go.loglayer.dev).

```go
// Example using the Pretty terminal transport.
// You can start out with one transport and swap to another later
// without touching application code.
import (
    "errors"

    "go.loglayer.dev"
    "go.loglayer.dev/transports/pretty"
)

log := loglayer.New(loglayer.Config{
    Transport: pretty.New(pretty.Config{}),
    // Put fields under a specific key (default is flattened)
    FieldsKey: "context",
})

// Persisted fields that appear on every subsequent log
log = log.WithFields(loglayer.Fields{
    "path":  "/",
    "reqId": "1234",
})

log.WithPrefix("[my-app]").
    WithError(errors.New("test")).
    // Data attached to this log entry only
    WithMetadata(loglayer.Metadata{"some": "data"}).
    Info("my message")
```

```json
{
  "level": "info",
  "time": "2026-04-26T12:00:00Z",
  "msg": "[my-app] my message",
  "context": {
    "path": "/",
    "reqId": "1234"
  },
  "some": "data",
  "err": {
    "message": "test"
  }
}
```

## Install

```sh
go get go.loglayer.dev
```

Requires **Go 1.25+** (driven by `transports/otellog`'s OpenTelemetry SDK floor; the rest of the library would happily run on older versions). Individual transports/plugins note any stricter requirement on their doc page.

## Examples

Runnable demos under [`examples/`](./examples):

- [`http-server`](./examples/http-server) — `loghttp` middleware in an HTTP handler
- [`multi-transport`](./examples/multi-transport) — pretty in dev + structured to file with per-transport level filtering
- [`custom-transport`](./examples/custom-transport) — implementing the Transport interface from scratch
- [`datadog-shipping`](./examples/datadog-shipping) — Datadog Logs intake with tuned batching

Run any with `go run ./examples/<name>`.

## Transports & Integrations

LogLayer ships adapters for the major Go loggers, self-contained renderers, network transports, and HTTP middleware. The full catalog with per-adapter docs lives at **[go.loglayer.dev/transports](https://go.loglayer.dev/transports/)** and **[go.loglayer.dev/integrations/loghttp](https://go.loglayer.dev/integrations/loghttp)**.

Writing your own transport is [a single interface](https://go.loglayer.dev/transports/creating-transports) with four methods.

## Documentation

Full docs at **[go.loglayer.dev](https://go.loglayer.dev)**:

- [Getting Started](https://go.loglayer.dev/getting-started)
- [Configuration](https://go.loglayer.dev/configuration) — every Config field
- [Cheat Sheet](https://go.loglayer.dev/cheatsheet) — one-page API reference
- [Logging API](https://go.loglayer.dev/logging-api/basic-logging) — per-method guides

## TypeScript counterpart

Already using [loglayer for TypeScript](https://loglayer.dev)? The Go port keeps the same mental model: persistent fields, per-call metadata, transport-agnostic facade. See [For TypeScript Developers](https://go.loglayer.dev/for-typescript-developers) for the full API mapping, conventions, and the deliberate Go-specific differences.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for setup, commit conventions, the test/lint/docs workflow, and PR requirements. Deeper architecture context (project structure, design decisions, thread-safety contract) lives in [AGENTS.md](AGENTS.md).

## License

[MIT](LICENSE)

Made with ❤️ by [Theo Gravity](https://suteki.nu) / [Disaresta](https://disaresta.com). Logo by [Akshaya Madhavan](https://www.linkedin.com/in/akshaya-madhavan).
