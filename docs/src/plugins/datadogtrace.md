---
title: Datadog APM Trace Injector Plugin
description: "Inject Datadog APM trace and span IDs into log entries for log/trace correlation."
---

# Datadog APM Trace Injector Plugin

`plugins/datadogtrace` adds Datadog's [log/trace correlation](https://docs.datadoghq.com/tracing/other_telemetry/connect_logs_and_traces/) fields to every log entry that carries an active span via `WithCtx`. Once your logs ship to Datadog, the UI will link each log line to the trace it originated in.

```sh
go get go.loglayer.dev/plugins/datadogtrace
```

The plugin is **tracer-agnostic**: you wire up a small extractor function that pulls the trace and span IDs from a `context.Context`. This avoids forcing a specific dd-trace-go version (or any tracer dependency at all) on LogLayer's main module — your service already imports the tracer it uses.

::: info Go version
The plugin itself inherits the main module's Go floor (1.25+). The optional **livetest module** at `plugins/datadogtrace/livetest/` has its own `go.mod` that pins `dd-trace-go/v2`; whatever Go floor that library demands lands there, isolated from the main module.
:::

## Basic Usage (dd-trace-go v1)

```go
import (
    "context"

    ddtracer "gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

    "go.loglayer.dev"
    "go.loglayer.dev/plugins/datadogtrace"
    "go.loglayer.dev/transports/structured"
)

func main() {
    ddtracer.Start(ddtracer.WithService("checkout-api"))
    defer ddtracer.Stop()

    log := loglayer.New(loglayer.Config{
        Transport: structured.New(structured.Config{}),
        Plugins: []loglayer.Plugin{
            datadogtrace.New(datadogtrace.Config{
                Service: "checkout-api",
                Env:     "production",
                Extract: func(ctx context.Context) (uint64, uint64, bool) {
                    span, ok := ddtracer.SpanFromContext(ctx)
                    if !ok {
                        return 0, 0, false
                    }
                    sc := span.Context()
                    return sc.TraceID(), sc.SpanID(), true
                },
            }),
        },
    })

    // Inside any handler that has a span on the context, bind once:
    handlerLog := log.WithCtx(r.Context())
    handlerLog.Info("request served")
    handlerLog.Info("downstream call done")
    // every emission carries r.Context() — the plugin reads it from
    // each entry's params.Ctx and emits dd.trace_id / dd.span_id.
}
```

::: tip Using loghttp middleware?
The [`loghttp`](/integrations/loghttp) middleware automatically binds `r.Context()` to the per-request logger, so handlers don't need the `log.WithCtx(r.Context())` step at all:

```go
func handler(w http.ResponseWriter, r *http.Request) {
    log := loghttp.FromRequest(r)
    log.Info("request served")          // r.Context() is already bound
    log.Info("downstream call done")
}
```
:::

A request with an active span produces:

```json
{
  "level": "info",
  "time": "2026-04-26T12:00:00Z",
  "msg": "request served",
  "dd.trace_id": "9876543210123456789",
  "dd.span_id": "1234567890",
  "dd.service": "checkout-api",
  "dd.env": "production"
}
```

When no span is attached (the context is nil, or it carries no span), the plugin emits nothing — the log entry goes through unchanged.

## Config

```go
type Config struct {
    ID      string                                                          // default "datadog-trace-injector"
    Extract func(ctx context.Context) (traceID, spanID uint64, ok bool)     // required
    Service string                                                          // optional dd.service
    Env     string                                                          // optional dd.env
    Version string                                                          // optional dd.version
    OnError func(err error)                                                 // optional, called on extractor panic
}
```

### `Extract`

Required. The function reads the active span from the context and returns its trace and span IDs as `uint64`.

For dd-trace-go v1:

```go
Extract: func(ctx context.Context) (uint64, uint64, bool) {
    if span, ok := ddtracer.SpanFromContext(ctx); ok {
        sc := span.Context()
        return sc.TraceID(), sc.SpanID(), true
    }
    return 0, 0, false
}
```

For dd-trace-go v2 (different import path, and `TraceID` now returns the full hex string; use `TraceIDLower` for the 64-bit decimal form Datadog log/trace correlation expects):

```go
import ddtracer2 "github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"

Extract: func(ctx context.Context) (uint64, uint64, bool) {
    span, ok := ddtracer2.SpanFromContext(ctx)
    if !ok {
        return 0, 0, false
    }
    sc := span.Context()
    return sc.TraceIDLower(), sc.SpanID(), true
}
```

The v2 extractor pattern is verified end-to-end against the real dd-trace-go v2 tracer (via its `mocktracer`) in `plugins/datadogtrace/livetest`, a separate test module that keeps Datadog's heavy dep tree out of the main loglayer module.

For OpenTelemetry tracers bridged to Datadog, parse the trace ID from the span context (it's a hex string in OTel) — see the Datadog OTel docs.

The function may return `ok=false` to skip injection for any reason (no active span, sampling decision not yet made, etc.). The plugin emits nothing in that case.

### `Service`, `Env`, `Version`

Optional Datadog reserved attributes. When set, they're emitted as `dd.service`, `dd.env`, `dd.version` on every entry that has a span. Empty strings are omitted.

The values you set here should match `tracer.WithService(...)`, `tracer.WithEnv(...)`, and `tracer.WithServiceVersion(...)` from your dd-trace-go setup.

### `OnError`

Optional handler called if `Extract` panics. The plugin recovers panics so logging never breaks because of tracer issues. Defaults to a silent no-op; pass a function if you want visibility.

```go
OnError: func(err error) {
    // log to stderr, increment a counter, whatever
    fmt.Fprintln(os.Stderr, err)
}
```

### `ID`

Defaults to `"datadog-trace-injector"`. Override only if you need to register multiple instances of the plugin (rare).

## Where it Fires

The plugin implements `OnBeforeDataOut`, which runs once per emission after fields and the error are assembled. Trace IDs land alongside your fields in the rendered output.

## Performance

The extractor runs on the dispatching goroutine for every log entry that has a context attached. dd-trace-go's `SpanFromContext` is a constant-time map lookup — fast enough for hot paths. No allocations beyond the `loglayer.Data` map the plugin returns.

The plugin is a no-op for log calls without `WithCtx`, so untraced logs pay zero cost.

## Live Integration Tests

The plugin ships with a live integration test against the real dd-trace-go v2 tracer (using its in-process `mocktracer`). It validates that the documented v2 extractor pattern produces IDs in the decimal-string format Datadog ingestion expects, including for nested spans.

The livetest lives in **its own Go module** at `plugins/datadogtrace/livetest/` so that dd-trace-go's heavy transitive closure (datadog-agent internals, OTel collector pieces, sketches-go, msgp, ...) stays out of the main `go.loglayer.dev` module. Plugin users get the lean main module; livetest contributors get the full SDK they need.

Run it from the repo root:

```sh
cd plugins/datadogtrace/livetest && go test -race ./...
```

CI runs it automatically on every push.

## What it Does NOT Do

- **Doesn't start its own tracer.** You're responsible for `tracer.Start(...)` and `tracer.Stop()`.
- **Doesn't propagate spans across HTTP/RPC.** Use dd-trace-go's contrib packages (e.g. `contrib/net/http`) for that.
- **Doesn't emit 128-bit trace IDs.** Datadog v1 returns 64-bit IDs; if you need the upper 64 bits for 128-bit traces, write a custom extractor that emits the full hex form.
