# Examples

Runnable demos showing common LogLayer wiring patterns. Each example is a standalone `main` package.

| Example | Demonstrates |
|---------|--------------|
| [`http-server`](./http-server) | `integrations/loghttp` middleware: per-request logger derivation in an HTTP handler |
| [`multi-transport`](./multi-transport) | Fan-out: pretty in dev + structured to a file with per-transport level filtering |
| [`custom-transport`](./custom-transport) | Implementing the `loglayer.Transport` interface from scratch |
| [`datadog-shipping`](./datadog-shipping) | Datadog Logs intake setup with tuned batching, error callback, and graceful Close |

Run any of them with:

```sh
go run ./examples/<name>
```

Each `main.go` is intentionally short (50-80 lines) so it doubles as a copy-paste starting point. None require a third-party service unless explicitly noted (the Datadog example degrades to a console transport when `DD_API_KEY` is not set).
