# go.loglayer.dev/plugins/oteltrace

LogLayer plugin that injects the active OTel `trace_id` and `span_id`
(plus optional trace flags, W3C trace state, and W3C baggage members)
into every log entry that carries a `context.Context`. Use with non-OTel
transports for log/trace correlation.

## Install

```sh
go get go.loglayer.dev/plugins/oteltrace
```

## Documentation

Full reference and examples: <https://go.loglayer.dev/plugins/oteltrace>

The framework itself: <https://go.loglayer.dev>
