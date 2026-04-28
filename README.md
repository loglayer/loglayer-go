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

`loglayer-go` is a structured logging facade for Go that sits in front of whatever logger you already use (zerolog, zap, log/slog, logrus, charmbracelet/log, phuslu) or one of the built-in transports (pretty terminal, structured JSON, HTTP, Datadog, OpenTelemetry). The pieces that aren't already trivial in slog or zerolog:

- **Reflective redaction.** A built-in plugin walks structs, maps, slices, and pointers at any depth and replaces matched keys or value patterns before any transport sees the value. Honors `json` tags; preserves runtime types.
- **Multi-transport fan-out with per-transport level filters.** Pretty in dev plus structured to a file plus batched HTTP to Datadog, all from one logger.
- **Group routing.** Tag entries by subsystem (`db`, `auth`, ...) and route each group to specific transports with its own minimum level. Toggle which groups are active at runtime via env var.
- **Two-way slog interop.** Wrap a `*slog.Logger` as a backend, or install a `slog.Handler` so `slog.Info(...)` calls (yours and your dependencies') flow through your full loglayer pipeline.
- **First-class test capture.** A typed `LogLine` capture so tests assert on level, message, fields, metadata, and context independently. No JSON parsing in tests.
- **Caller info opt-in.** `Config.Source.Enabled` captures file/line/function per emission and renders it under `Config.Source.FieldName` (default `"source"`). JSON tags match the `log/slog` convention so output is interchangeable. The slog handler forwards `Record.PC` for free.

Application code uses one fluent API for messages, fields, metadata, and errors regardless of which transport(s) are behind it. Full documentation at [go.loglayer.dev](https://go.loglayer.dev).

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

Requires **Go 1.25+** for the main module. The OpenTelemetry transport (`go.loglayer.dev/transports/otellog`) and trace-injector plugin (`go.loglayer.dev/plugins/oteltrace`) ship as their own Go modules so the OTel SDK's transitive deps don't bind users who don't need them. Individual transports/plugins note any stricter requirement on their doc page.

## Examples

Runnable demos under [`examples/`](./examples):

- [`http-server`](./examples/http-server): `loghttp` middleware in an HTTP handler
- [`multi-transport`](./examples/multi-transport): pretty in dev + structured to file with per-transport level filtering
- [`custom-transport`](./examples/custom-transport): implementing the Transport interface from scratch (renderer / "flatten" policy)
- [`custom-transport-attribute`](./examples/custom-transport-attribute): Transport that forwards to an attribute-style backend (wrapper policy)
- [`custom-plugin`](./examples/custom-plugin): writing a plugin from scratch (`OnBeforeDataOut`, `OnMetadataCalled`, `ShouldSend`)
- [`datadog-shipping`](./examples/datadog-shipping): Datadog Logs intake with tuned batching
- [`otel-end-to-end`](./examples/otel-end-to-end): `transports/otellog` + `plugins/oteltrace` against a real OTel SDK (own module; `cd examples/otel-end-to-end && go run .`)

Run any of the main-module ones with `go run ./examples/<name>`.

## Transports & Integrations

LogLayer ships adapters for the major Go loggers, self-contained renderers, network transports, and HTTP middleware. The full catalog with per-adapter docs lives at **[go.loglayer.dev/transports](https://go.loglayer.dev/transports/)** and **[go.loglayer.dev/integrations/loghttp](https://go.loglayer.dev/integrations/loghttp)**.

Writing your own transport is [a single interface](https://go.loglayer.dev/transports/creating-transports) with four methods.

## Documentation

Full docs at **[go.loglayer.dev](https://go.loglayer.dev)**:

- [Getting Started](https://go.loglayer.dev/getting-started)
- [Configuration](https://go.loglayer.dev/configuration): every Config field
- [Cheat Sheet](https://go.loglayer.dev/cheatsheet): one-page API reference
- [Logging API](https://go.loglayer.dev/logging-api/basic-logging): per-method guides

## TypeScript counterpart

Already using [loglayer for TypeScript](https://loglayer.dev)? The Go port keeps the same mental model: persistent fields, per-call metadata, transport-agnostic facade. See [For TypeScript Developers](https://go.loglayer.dev/for-typescript-developers) for the full API mapping, conventions, and the deliberate Go-specific differences.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for setup, commit conventions, the test/lint/docs workflow, and PR requirements. Deeper architecture context (project structure, design decisions, thread-safety contract) lives in [AGENTS.md](AGENTS.md).

## License

[MIT](LICENSE)

Made with ❤️ by [Theo Gravity](https://suteki.nu) / [Disaresta](https://disaresta.com). Logo by [Akshaya Madhavan](https://www.linkedin.com/in/akshaya-madhavan).
