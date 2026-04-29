# Examples

Runnable demos showing common LogLayer wiring patterns. Each example is a standalone `main` package.

| Example | Demonstrates |
|---------|--------------|
| [`http-server`](./http-server) | `integrations/loghttp` middleware: per-request logger derivation in an HTTP handler |
| [`multi-transport`](./multi-transport) | Fan-out: pretty in dev + structured to a file with per-transport level filtering |
| [`custom-transport`](./custom-transport) | Implementing the `loglayer.Transport` interface from scratch (renderer / "flatten" policy) |
| [`custom-transport-attribute`](./custom-transport-attribute) | Transport that forwards to an attribute-style backend (wrapper policy) |
| [`custom-plugin`](./custom-plugin) | Writing a plugin from scratch: `OnBeforeDataOut`, `OnMetadataCalled`, and `ShouldSend` hooks |
| [`datadog-shipping`](./datadog-shipping) | Datadog Logs intake setup with tuned batching, error callback, and graceful Close |
| [`otel-end-to-end`](./otel-end-to-end) | `transports/otellog` + `plugins/oteltrace` against a real OTel SDK with stdout exporters |

Run any of them with:

```sh
go run ./examples/<name>
```

Each `main.go` is intentionally short (50-100 lines) so it doubles as a copy-paste starting point. None require a third-party service unless explicitly noted (the Datadog example degrades to a console transport when `DD_API_KEY` is not set).

The `otel-end-to-end` example lives in its own Go module because the OpenTelemetry transport and plugin are themselves split out (see [AGENTS.md](../AGENTS.md) "When to Split a Transport into Its Own Module"). Because Go treats it as a separate module, you have to `cd` into it to run:

```sh
cd examples/otel-end-to-end
go run .
```
