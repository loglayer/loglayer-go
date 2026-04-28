# go.loglayer.dev/transports/otellog

OpenTelemetry Logs transport for LogLayer. Emits each entry as an OTel
`log.Record` on a configured `LoggerProvider`, propagating `WithCtx` so
the SDK's span correlation works.

**Requires Go 1.25+** (driven by `go.opentelemetry.io/otel/sdk/log`).

## Install

```sh
go get go.loglayer.dev/transports/otellog
```

## Documentation

Full reference and examples: <https://go.loglayer.dev/transports/otellog>

The framework itself: <https://go.loglayer.dev>
