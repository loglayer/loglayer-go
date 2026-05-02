---
title: Sentry Transport
description: Forward LogLayer entries to a sentry.Logger.
---

# Sentry Transport

<ModuleBadges path="transports/sentry" />

Forwards each entry to a caller-supplied [`sentry.Logger`](https://pkg.go.dev/github.com/getsentry/sentry-go#Logger), Sentry's structured-logs API. Use this when you've already wired Sentry into your service for error reporting and want LogLayer entries to flow through the same pipeline.

The package directory is `transports/sentry`; the package name is `sentrytransport` to avoid colliding with the imported `sentry` identifier. Import path: `go.loglayer.dev/transports/sentry/v2`.

```sh
go get go.loglayer.dev/transports/sentry/v2
go get github.com/getsentry/sentry-go
```

The Sentry SDK is your responsibility to install and initialize; the transport just hands entries off to a `sentry.Logger` you provide.

## Getting a DSN

Sentry identifies your project with a **DSN** (Data Source Name): a URL that looks like `https://<key>@o<org>.ingest.<region>.sentry.io/<project>`. To get one:

1. Sign in at <https://sentry.io> and create a project if you don't have one (any platform; the DSN works regardless).
2. Open **Settings** → **Projects** → your project → **Client Keys (DSN)**.
3. Copy the value labelled **DSN**.

Sentry's logs feature is opt-in. In the project settings, ensure **Logs** is enabled (it's on by default for new projects on most plans). The `EnableLogs: true` flag below is the client-side switch; the server-side switch has to match.

The DSN is non-secret in the sense that it's safe to ship in client-side apps, but treat it the same way you'd treat any other configuration: load it from an environment variable rather than hard-coding it.

## Basic Usage

```go
import (
    "context"

    "github.com/getsentry/sentry-go"

    "go.loglayer.dev/v2"
    sentrytransport "go.loglayer.dev/transports/sentry/v2"
)

// Initialize Sentry as you normally would.
sentry.Init(sentry.ClientOptions{
    Dsn:        "https://<key>@<org>.ingest.sentry.io/<project>",
    EnableLogs: true,
})
defer sentry.Flush(2 * time.Second)

ctx := context.Background()
log := loglayer.New(loglayer.Config{
    Transport: sentrytransport.New(sentrytransport.Config{
        Logger: sentry.NewLogger(ctx),
    }),
})

log.Info("user signed in")
log.WithMetadata(loglayer.Metadata{"userId": 42}).Warn("retry exhausted")
```

`Config.Logger` is required; `New` panics when it's nil. Use `Build` for the error-returning variant when the logger is wired at runtime.

## Config

```go
type Config struct {
    transport.BaseConfig

    Logger sentry.Logger // required; typically sentry.NewLogger(ctx)
}
```

## Fatal Behavior

The transport routes `LogLevelFatal` and `LogLevelPanic` through `sentry.Logger.LFatal()`, which emits at fatal severity **without** triggering Sentry's `os.Exit` (`Fatal()`) or `panic()` (`Panic()`) behavior. LogLayer's core decides whether the process actually terminates via `Config.DisableFatalExit`, matching every other transport in this repo. See [Fatal Exits the Process](/logging-api/basic-logging#fatal-exits-the-process).

## Metadata Handling

<!--@include: ./_partials/metadata-field-name.md-->

### Map metadata flattens to attributes

```go
log.WithMetadata(loglayer.Metadata{"requestId": "abc", "n": 42}).Info("served")
// Sentry attributes: requestId="abc", n=42
```

Each map entry becomes a typed Sentry attribute via the matching `LogEntry` setter (`String`, `Int`, `Int64`, `Float64`, `Bool`, plus the slice variants). Values that don't match a typed setter (nested maps, structs, mixed-type slices) are JSON-encoded into a single `String` attribute, so the structure is preserved in Sentry's UI.

### Struct metadata nests under the metadata key

Non-map metadata (structs, scalars, slices) is JSON-encoded and stored under a single configurable attribute key. The default key is `"metadata"`:

```go
import sentrytransport "go.loglayer.dev/transports/sentry/v2"

tr := sentrytransport.New(sentrytransport.Config{
    Logger: sentry.NewLogger(ctx),
})

type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

log.WithMetadata(User{ID: 7, Name: "Alice"}).Info("user")
// Sentry attribute: metadata=`{"id":7,"name":"Alice"}`
```

Override the key globally via the core's `MetadataFieldName` (which also nests map metadata under the same key):

```go
loglayer.New(loglayer.Config{
    Transport:         sentrytransport.New(sentrytransport.Config{Logger: sentry.NewLogger(ctx)}),
    MetadataFieldName: "payload",
})

log.WithMetadata(User{ID: 7, Name: "Alice"}).Info("user")
// Sentry attribute: payload=`{"id":7,"name":"Alice"}`
```

The Sentry `LogEntry` API only accepts a fixed set of typed attribute shapes (no `Map` / `Any` / `Object` setter), so non-scalar values land as a JSON-encoded string. `json:` tags on struct fields apply normally.

## Fields

Persistent fields attached via `WithFields` flow through the same typed-attribute dispatch:

```go
log = log.WithFields(loglayer.Fields{"service": "api"})
log.Info("request")
// Sentry attributes: service="api"
```

If `Config.FieldsKey` is set on the LogLayer core, fields are nested under that key first, then sent as a single JSON-encoded attribute (since the nested `map[string]any` falls through to the JSON path).

## Errors

Errors attached via `WithError` are serialized by LogLayer's `ErrorSerializer` (default produces `{"message": "..."}`) and arrive in the transport as a map under `Config.ErrorFieldName` (default `"err"`). The map JSON-encodes into a single `String` attribute:

```go
log.WithError(errors.New("boom")).Error("failed")
// Sentry attribute: err=`{"message":"boom"}`
```

To enrich the rendering, set a custom `Config.ErrorSerializer` on the LogLayer core that returns additional fields (`stack`, `code`, etc.); they'll appear inside the same JSON-encoded attribute.

## context.Context Pass-through

The `context.Context` attached via `WithContext` is forwarded to the chain via `LogEntry.WithCtx` so Sentry's hub/scope correlation picks up trace/span IDs from the active context:

```go
log.WithContext(ctx).Info("served")
```

For the persistent-binding pattern in HTTP handlers, see [Go Context](/logging-api/go-context).

## Reaching the Underlying Logger

`GetLoggerInstance` returns the wrapped `sentry.Logger`:

```go
import "github.com/getsentry/sentry-go"

l := log.GetLoggerInstance("sentry").(sentry.Logger)
```

(`"sentry"` is whatever you set as `BaseConfig.ID`; defaults to an auto-generated ID when unset.)

## Level Mapping

| LogLayer Level   | Sentry Method | Notes |
|------------------|---------------|-------|
| `LogLevelTrace`  | `Trace()`     |       |
| `LogLevelDebug`  | `Debug()`     |       |
| `LogLevelInfo`   | `Info()`      |       |
| `LogLevelWarn`   | `Warn()`      |       |
| `LogLevelError`  | `Error()`     |       |
| `LogLevelFatal`  | `LFatal()`    | Emits at fatal severity without `os.Exit`. The core decides whether the process exits. |
| `LogLevelPanic`  | `LFatal()`    | Same. Sentry has no level above fatal. The core decides whether to `panic()` after dispatch. |

The transport never calls `Logger.Fatal()` or `Logger.Panic()` (which would terminate the process), regardless of how the wrapped Sentry instance is configured.
