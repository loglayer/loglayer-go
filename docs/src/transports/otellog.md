---
title: OpenTelemetry Logs Transport
description: Emit LogLayer entries to an OpenTelemetry log.Logger.
---

# OpenTelemetry Logs Transport

<ModuleBadges path="transports/otellog" />

Emits each entry as an OpenTelemetry [`log.Record`](https://pkg.go.dev/go.opentelemetry.io/otel/log#Record) on a `log.Logger`. Use this when your service is wired against the OpenTelemetry SDK and you want LogLayer entries to flow through the same pipeline (OTLP exporter, Collector, observability backend) as your traces and metrics.

The package name is `otellog` to avoid colliding with `go.opentelemetry.io/otel`. Import path: `go.loglayer.dev/transports/otellog/v2`.

```sh
go get go.loglayer.dev/transports/otellog/v2
```

::: info Separate module
`transports/otellog` ships as its own Go module (`go.loglayer.dev/transports/otellog/v2`) so the OpenTelemetry SDK's transitive Go-version requirement doesn't bind the main `go.loglayer.dev/v2` module. Users who don't import the OTel transport never see its dependency graph.

Requires **Go 1.25+** because that's the floor of the upstream `go.opentelemetry.io/otel/sdk/log` packages this transport binds against.
:::

## Basic Usage (Global Provider)

If your app has already registered an OTel `LoggerProvider` globally (the common case when you set up the OTel SDK at startup), pass only an instrumentation scope `Name`:

```go
import (
    "go.loglayer.dev/v2"
    "go.loglayer.dev/transports/otellog/v2"
)

tr := otellog.New(otellog.Config{Name: "checkout-api"})

log := loglayer.New(loglayer.Config{Transport: tr})

log.Info("user signed in")
// Emits a log.Record with severity=INFO, body="user signed in",
// scope name="checkout-api". The OTel SDK exports it through whatever
// processor/exporter chain you've configured.
```

When no global provider has been registered, OTel returns a no-op provider: construction succeeds, emission silently drops. This matches the OTel SDK's default contract.

## Wiring an Explicit LoggerProvider

For explicit wiring (recommended in services that own their SDK setup):

```go
import (
    sdklog "go.opentelemetry.io/otel/sdk/log"
    "go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
    "go.opentelemetry.io/otel/sdk/resource"
    semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

res := resource.NewSchemaless(
    semconv.ServiceName("checkout-api"),
    semconv.ServiceVersion("1.2.3"),
    semconv.DeploymentEnvironment("production"),
)

exporter, _ := otlploghttp.New(ctx)
provider := sdklog.NewLoggerProvider(
    sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
    sdklog.WithResource(res),
)
defer provider.Shutdown(ctx)

tr := otellog.New(otellog.Config{
    Name:           "checkout-api",
    Version:        "1.2.3",
    LoggerProvider: provider,
})
```

::: warning Resource schema URL conflicts
Wiring a resource with `resource.Merge(resource.Default(), resource.NewWithAttributes(semconv.SchemaURL, ...))` can fail with `conflicting Schema URL`: `resource.Default()` carries the SDK's current schema URL, while `semconv.SchemaURL` from a pinned import is older. The merge refuses to combine the two.

Fix by either dropping the schema URL on your additions (`resource.NewSchemaless(...)`, shown above) or upgrading the `semconv` import to match `resource.Default()`'s schema. `resource.NewSchemaless` is the simpler choice for most apps; the schema URL only matters when downstream consumers do strict semantic-conventions validation.
:::

## Pre-Built Logger (Tests, Advanced Wiring)

When you've already constructed a `log.Logger` (or want to inject a stub in tests), pass it directly. `Logger` takes precedence over `LoggerProvider`/`Name`/`Version`/`SchemaURL`:

```go
tr := otellog.New(otellog.Config{Logger: myLogger})
```

### Tests with `logtest.Recorder`

For unit tests, the OTel project's [`logtest.Recorder`](https://pkg.go.dev/go.opentelemetry.io/otel/log/logtest#Recorder) implements `log.LoggerProvider` directly, so you can pass it straight to `Config.LoggerProvider` without an SDK provider, processor, or exporter in between:

```go
import "go.opentelemetry.io/otel/log/logtest"

func TestEmits(t *testing.T) {
    rec := logtest.NewRecorder()
    log := loglayer.New(loglayer.Config{
        Transport: otellog.New(otellog.Config{
            Name:           "test",
            LoggerProvider: rec,
        }),
    })

    log.Info("user signed in")

    for _, records := range rec.Result() {
        for _, r := range records {
            if r.Body.AsString() == "user signed in" {
                return
            }
        }
    }
    t.Fatal("expected log record not captured")
}
```

`rec.Result()` returns a `Recording` keyed by instrumentation scope; each value is a `[]logtest.Record` you can assert against (Body, Severity, Attributes, Context).

## Config

```go
type Config struct {
    transport.BaseConfig

    Name           string             // instrumentation scope name; required when Logger is nil
    Version        string             // optional scope version
    SchemaURL      string             // optional schema URL
    LoggerProvider log.LoggerProvider // defaults to global.GetLoggerProvider()
    Logger         log.Logger         // pre-built logger; takes precedence over the above
}
```

`New` panics on missing configuration; `Build` returns `otellog.ErrNameRequired` instead. Use `Build` when the LoggerProvider is wired at runtime (e.g. from environment variables).

## Fatal Behavior

Fatal entries are emitted at `SeverityFatal` (the OTel severity numeric `21`) on the underlying logger. The OTel SDK never calls `os.Exit`. Whether the process actually exits is the LogLayer core's decision via `Config.DisableFatalExit`. See [Fatal Exits the Process](/logging-api/basic-logging#fatal-exits-the-process).

## Severity Mapping

OTel defines four sub-levels per severity bucket (`SeverityDebug1`-`Debug4`, `Info1`-`Info4`, etc., numeric 1-24). LogLayer has a single level per bucket, so we use the first of each:

| LogLayer Level   | OTel Severity      | Numeric |
|------------------|--------------------|---------|
| `LogLevelTrace`  | `SeverityTrace`    | 1       |
| `LogLevelDebug`  | `SeverityDebug`    | 5       |
| `LogLevelInfo`   | `SeverityInfo`     | 9       |
| `LogLevelWarn`   | `SeverityWarn`     | 13      |
| `LogLevelError`  | `SeverityError`    | 17      |
| `LogLevelFatal`  | `SeverityFatal`    | 21      |
| `LogLevelPanic`  | `SeverityFatal4`   | 24      |

Panic uses `SeverityFatal4` (the highest tier in OTel's Fatal bucket) so it stays distinguishable from Fatal on the OTel side.

The original LogLayer level name (`"info"`, `"error"`, etc.) is also set as `SeverityText` on the record.

## Metadata Handling

<!--@include: ./_partials/metadata-field-name.md-->

### Map metadata flattens to attributes

```go
log.WithMetadata(loglayer.Metadata{"requestId": "abc", "n": 42}).Info("served")
// log.Record attributes: requestId="abc", n=42
```

Each map entry becomes a typed `log.KeyValue`: strings → `StringValue`, ints → `Int64Value`, bools → `BoolValue`, and so on. Nested maps and slices recurse into `MapValue` / `SliceValue`.

### Struct metadata nests under the metadata key

```go
type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

log.WithMetadata(User{ID: 7, Name: "Alice"}).Info("user")
// log.Record attribute: metadata=Map{id=7, name="Alice"}
```

The struct is JSON-roundtripped (so `json:` tags apply) then converted to a nested `MapValue` under the metadata attribute (default `"metadata"`).

Override the key globally via the core's `MetadataFieldName` (which also nests map metadata under the same key):

```go
loglayer.New(loglayer.Config{
    Transport:         otellog.New(otellog.Config{Name: "checkout-api"}),
    MetadataFieldName: "user",
})

log.WithMetadata(User{ID: 9}).Info("hi")
// log.Record attribute: user=Map{id=9, name=""}
```

## context.Context Pass-through

The `context.Context` attached via `WithContext` is forwarded to `log.Logger.Emit`. OTel SDK processors (and the `BatchProcessor` in particular) read the active span from the context to populate the record's `TraceID` / `SpanID`, giving you log/trace correlation automatically:

```go
log.WithContext(ctx).Info("served")
// If ctx carries an active span, the exported log.Record carries
// TraceID and SpanID copied from it.
```

For the persistent-binding pattern in HTTP handlers, see [Go Context](/logging-api/go-context). The `loghttp` middleware binds `r.Context()` automatically so handlers reading via `loghttp.FromRequest(r)` get trace correlation with no per-emission boilerplate.

## Live Integration Tests

The transport ships with `//go:build livetest`-tagged tests that exercise the real OpenTelemetry SDK end-to-end (real `LoggerProvider` with an in-memory `Exporter`, real `TracerProvider` for span correlation). They're skipped by the default test run and opt-in via:

```sh
go test -tags=livetest ./transports/otellog/
```

CI runs them automatically. See `transports/otellog/livetest_test.go` for the full set.

## Reaching the Underlying Logger

`GetLoggerInstance` returns the underlying `log.Logger`:

```go
import otellogapi "go.opentelemetry.io/otel/log"

l := log.GetLoggerInstance("otellog").(otellogapi.Logger)
```

(`"otellog"` is the default `BaseConfig.ID`; set `BaseConfig.ID` explicitly when running multiple OTel transports.)
