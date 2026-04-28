---
title: OpenTelemetry Trace Injector Plugin
description: "Inject OpenTelemetry trace_id and span_id into log entries for log/trace correlation on any transport."
---

# OpenTelemetry Trace Injector Plugin

`plugins/oteltrace` adds the active OpenTelemetry span's `trace_id` and `span_id` to every log entry that carries a `context.Context`. Use it when your service runs OpenTelemetry tracing but ships logs to a destination other than the OTel logs pipeline (structured JSON to stdout, Datadog HTTP intake, Loki, custom transports).

```sh
go get go.loglayer.dev/plugins/oteltrace
```

::: info Separate module
`plugins/oteltrace` ships as its own Go module (`go.loglayer.dev/plugins/oteltrace`) so the OpenTelemetry API's Go-version requirement doesn't bind the main `go.loglayer.dev` module. Requires **Go 1.25+** because that's the floor of `go.opentelemetry.io/otel/trace` and `go.opentelemetry.io/otel/baggage` at current versions.
:::

::: info When to use this vs `transports/otellog`
- **Shipping logs through the OTel pipeline?** Use [`transports/otellog`](/transports/otellog). The OTel SDK reads the active span from each emission's context and embeds the trace IDs on the exported `log.Record` automatically. You don't need this plugin.
- **Shipping to a non-OTel destination?** Use this plugin. It surfaces `trace_id` / `span_id` as flat fields so any backend can index them.
- **Doing both?** Use both. The plugin makes the IDs visible on every record regardless of destination.
:::

## Basic Usage

```go
import (
    "go.loglayer.dev"
    "go.loglayer.dev/plugins/oteltrace"
    "go.loglayer.dev/transports/structured"
)

log := loglayer.New(loglayer.Config{
    Transport: structured.New(structured.Config{}),
    Plugins: []loglayer.Plugin{
        oteltrace.New(oteltrace.Config{}),
    },
})

// Inside a handler whose context carries an OTel span, bind once:
handlerLog := log.WithCtx(r.Context())
handlerLog.Info("request served")
handlerLog.Info("downstream call done")
```

A request with an active span produces:

```json
{
  "level": "info",
  "time": "2026-04-26T12:00:00Z",
  "msg": "request served",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span_id": "00f067aa0ba902b7"
}
```

When no span is attached (the context is nil, or it carries no valid span), the plugin emits nothing. The log entry goes through unchanged.

::: tip Using loghttp middleware?
The [`loghttp`](/integrations/loghttp) middleware automatically binds `r.Context()` to the per-request logger, so handlers don't need the `log.WithCtx(r.Context())` step:

```go
func handler(w http.ResponseWriter, r *http.Request) {
    log := loghttp.FromRequest(r)
    log.Info("request served") // r.Context() is already bound
}
```
:::

## Config

```go
type Config struct {
    ID               string              // default "otel-trace-injector"
    TraceIDKey       string              // default "trace_id"
    SpanIDKey        string              // default "span_id"
    TraceFlagsKey    string              // default "" (omit)
    TraceStateKey    string              // default "" (omit)
    BaggageKeyPrefix string              // default "" (omit)
    OnError          func(err error)     // optional, called on plugin panic
}
```

### `TraceIDKey` / `SpanIDKey`

The data keys under which the IDs are emitted. Defaults match OTel's JSON serialization (`trace_id`, `span_id`). For Elastic Common Schema (ECS) compatibility, set them to `trace.id` / `span.id`. For camelCase backends, `traceId` / `spanId`.

```go
oteltrace.New(oteltrace.Config{
    TraceIDKey: "trace.id",
    SpanIDKey:  "span.id",
})
```

The IDs are emitted in OTel's canonical lowercase-hex form (32 chars for trace, 16 for span) via `trace.TraceID.String()` and `trace.SpanID.String()`.

### `TraceFlagsKey`

When non-empty, the plugin also emits the trace flags byte under that key as an `int` (0-255; bit 0 is the sampled flag). Useful when the backend wants to know whether the span was sampled.

```go
oteltrace.New(oteltrace.Config{TraceFlagsKey: "trace_flags"})
// trace_flags: 1 when sampled, 0 otherwise.
```

When empty (the default), the trace flags are not emitted.

### `TraceStateKey`

When non-empty, the plugin emits the W3C [trace state](https://www.w3.org/TR/trace-context/#tracestate-header) as a single string under that key. Trace state carries vendor-specific routing/sampling info that propagates with the trace context (`vendor1=val1,vendor2=val2`).

```go
oteltrace.New(oteltrace.Config{TraceStateKey: "trace_state"})
// trace_state: "vendor1=val1,vendor2=val2"
```

The plugin emits in canonical W3C form via `trace.TraceState.String()`. When the trace state is empty, no attribute is added even if the key is configured.

### `BaggageKeyPrefix`

When non-empty, the plugin emits each W3C [baggage](https://www.w3.org/TR/baggage/) member from the context as a separate attribute, keyed `<prefix><member-key>`.

```go
oteltrace.New(oteltrace.Config{BaggageKeyPrefix: "baggage."})

// In a handler whose ctx carries baggage:
//   user_id=alice, tenant_id=acme
// Output:
//   baggage.user_id: "alice"
//   baggage.tenant_id: "acme"
```

Baggage rides independently of the trace span: a context with baggage but no active span still surfaces baggage attributes (without `trace_id` / `span_id`).

Baggage values are application-controlled. If your application puts unbounded data in baggage (full request bodies, etc.), surfacing all of it on every log line will inflate output and may leak data to log destinations. Use a prefix like `baggage.` so consumers can filter, and consider trimming high-cardinality keys upstream.

### `OnError`

Optional handler invoked if the plugin panics during emission. The framework recovers the panic so logging keeps working; `OnError` lets you observe it (log to stderr, increment a counter, send to your error tracker). Defaults to silent.

### `ID`

Defaults to `"otel-trace-injector"`. Override only if you need to register multiple instances of the plugin (rare).

## Where it Fires

The plugin implements `OnBeforeDataOut`, which runs once per emission after fields and the error are assembled. The trace IDs land alongside your fields in the rendered output.

## Performance

The plugin reads the span context via `trace.SpanContextFromContext` (a single context value lookup) on every emission that has a context attached. No allocations beyond the small `loglayer.Data` map the plugin returns.

The plugin is a no-op for log calls without `WithCtx` and for contexts that don't carry a valid span context, so untraced logs pay zero cost.

## Live Integration Tests

The plugin ships with `//go:build livetest`-tagged tests that exercise a real OpenTelemetry `TracerProvider`. They start real spans, drive entries through LogLayer, and assert the emitted `trace_id` / `span_id` match the SDK's actual span IDs (including nested spans, sampled/never-sampled, and trace flags). Skipped by the default test run; opt-in via:

```sh
go test -tags=livetest ./plugins/oteltrace/
```

CI runs them automatically. See `plugins/oteltrace/livetest_test.go`.

## What it Does NOT Do

- **Doesn't start a tracer.** You set up the OTel SDK or any conformant tracer (Jaeger, Zipkin, vendor SDKs that implement the OTel API) yourself.
- **Doesn't wire trace context across service boundaries.** This plugin only reads from the local `context.Context`. Getting trace IDs *onto* outgoing HTTP/gRPC requests (and *off* incoming ones) is the job of OTel's instrumentation libraries: [`otelhttp`](https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp) for stdlib `net/http`, [`otelgrpc`](https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc) for gRPC, etc. Without one of those installed, downstream services start with no span on their context and this plugin will emit nothing for their logs.
