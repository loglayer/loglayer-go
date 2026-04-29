# go.loglayer.dev/transports/otellog

[![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/otellog.svg)](https://pkg.go.dev/go.loglayer.dev/transports/otellog)

OpenTelemetry Logs transport for LogLayer. Emits each entry as an OTel `log.Record` on a configured `LoggerProvider`, propagating `WithContext` so the SDK's span correlation works.

**Requires Go 1.25+** (driven by `go.opentelemetry.io/otel/sdk/log`). Ships as its own Go module so the OTel SDK's transitive deps don't bind users who don't import them.

## Install

```sh
go get go.loglayer.dev/transports/otellog
```

## Documentation

Full reference and examples: <https://go.loglayer.dev/transports/otellog>

Main library: <https://go.loglayer.dev>
