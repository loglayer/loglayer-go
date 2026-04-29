# go.loglayer.dev/plugins/oteltrace

[![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/plugins/oteltrace.svg)](https://pkg.go.dev/go.loglayer.dev/plugins/oteltrace)

LogLayer plugin that injects the active OTel `trace_id` and `span_id` (plus optional trace flags, W3C trace state, and W3C baggage members) into every log entry that carries a `context.Context`. Use with non-OTel transports for log/trace correlation; `transports/otellog` does this automatically via the SDK.

**Requires Go 1.25+** (driven by `go.opentelemetry.io/otel`). Ships as its own Go module so the OTel API's transitive deps don't bind users who don't import them.

## Install

```sh
go get go.loglayer.dev/plugins/oteltrace
```

## Documentation

Full reference and examples: <https://go.loglayer.dev/plugins/oteltrace>

Main library: <https://go.loglayer.dev>
