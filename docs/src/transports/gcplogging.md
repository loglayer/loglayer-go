---
title: Google Cloud Logging Transport
description: "Forward LogLayer entries to Google Cloud Logging via cloud.google.com/go/logging."
---

# Google Cloud Logging Transport

<ModuleBadges path="transports/gcplogging" />

Forwards each entry to a caller-supplied [`*logging.Logger`](https://pkg.go.dev/cloud.google.com/go/logging#Logger) from the official Google Cloud client library. Use this on Google Cloud Run, GKE, App Engine, Cloud Functions, or any environment where you want logs to land in Google Cloud Logging (formerly Stackdriver).

Import path: `go.loglayer.dev/transports/gcplogging`. Package name: `gcplogging` (no collision with the SDK's `logging` package).

```sh
go get go.loglayer.dev/transports/gcplogging
go get cloud.google.com/go/logging
```

The Google Cloud SDK is your responsibility to install and initialize; the transport just hands entries off to a `*logging.Logger` you provide.

## Authenticating

There is no API key for Cloud Logging. The Google Cloud SDK authenticates via [Application Default Credentials](https://cloud.google.com/docs/authentication/application-default-credentials), which are picked up automatically:

- **On GCP runtimes** (Cloud Run, GKE, App Engine, Compute Engine, Cloud Functions): the workload's attached service account is used. Nothing extra to configure as long as the service account has the `roles/logging.logWriter` IAM role.
- **Locally / off-GCP**: either run `gcloud auth application-default login` (uses your user identity) or set `GOOGLE_APPLICATION_CREDENTIALS` to the path of a service-account JSON key file with `roles/logging.logWriter`.

The `*logging.Client` constructed by `logging.NewClient(ctx, projectID)` reads ADC at construction time. No credential is ever passed to this transport directly.

| Env var | Purpose |
|---------|---------|
| `GOOGLE_CLOUD_PLATFORM_PROJECT_ID` | Project ID passed to `logging.NewClient`. Empty triggers SDK auto-detection (metadata server / `GOOGLE_CLOUD_PROJECT`). |
| `GOOGLE_APPLICATION_CREDENTIALS` | Path to a service-account JSON key file. Read by ADC outside GCP runtimes. |

### Local dev with a service-account key

When `GOOGLE_APPLICATION_CREDENTIALS` is set before launch, no Go code changes are needed:

```sh
export GOOGLE_CLOUD_PLATFORM_PROJECT_ID=my-gcp-project
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json
go run ./cmd/your-app
```

To bypass ADC and load the key file explicitly from a known path, pass [`option.WithCredentialsFile`](https://pkg.go.dev/google.golang.org/api/option#WithCredentialsFile) to the client:

```go
import (
    "cloud.google.com/go/logging"
    "google.golang.org/api/option"
)

client, err := logging.NewClient(
    ctx,
    "my-gcp-project",
    option.WithCredentialsFile("/path/to/service-account.json"),
)
```

## Basic Usage

```go
import (
    "context"
    "os"

    "cloud.google.com/go/logging"

    "go.loglayer.dev"
    "go.loglayer.dev/transports/gcplogging"
)

ctx := context.Background()
client, err := logging.NewClient(ctx, os.Getenv("GOOGLE_CLOUD_PLATFORM_PROJECT_ID"))
if err != nil {
    panic(err)
}
defer client.Close()

gcpLogger := client.Logger("my-log")

log := loglayer.New(loglayer.Config{
    Transport: gcplogging.New(gcplogging.Config{
        Logger: gcpLogger,
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

    Logger        *logging.Logger
    RootEntry     logging.Entry
    EntryFn       func(loglayer.TransportParams, *logging.Entry)
    MessageField  string
    Sync          bool
    OnError       func(error)
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `Logger` | `*logging.Logger` | (required) | Constructed via `logging.NewClient(ctx, projectID).Logger(logID)`. |
| `RootEntry` | `logging.Entry` | zero | LogEntry skeleton merged into every entry. `Severity`, `Timestamp`, and `Payload` are managed by the transport. Common fields to set: `Resource`, `Labels`, `HTTPRequest`, `Operation`, `Trace`, `SourceLocation`. |
| `EntryFn` | `func(params, *logging.Entry)` | nil | Per-entry hook that mutates the resolved Entry just before dispatch. Use it to lift values from `params.Metadata` onto typed Entry fields. |
| `MessageField` | `string` | `"message"` | The key under which the joined message text is placed inside the JSON payload. |
| `Sync` | `bool` | `false` | Route entries through `Logger.LogSync` (blocking) instead of `Logger.Log` (async, batched). |
| `OnError` | `func(error)` | stderr | Called when `LogSync` returns an error or `Close`'s `Flush` fails. |

### `RootEntry`

Set fields that should appear on every entry uniformly: a [MonitoredResource](https://cloud.google.com/logging/docs/api/v2/resource-list), static labels, etc.

```go
gcplogging.New(gcplogging.Config{
    Logger: gcpLogger,
    RootEntry: logging.Entry{
        Resource: &mrpb.MonitoredResource{
            Type: "cloud_run_revision",
            Labels: map[string]string{
                "project_id":    "my-project",
                "service_name":  "my-service",
                "revision_name": "my-rev",
            },
        },
        Labels: map[string]string{
            "env":     "production",
            "version": "1.0.0",
        },
    },
})
```

### `EntryFn`

Use `EntryFn` when typed `Entry` fields (`Trace`, `SpanID`, `Labels`, `HTTPRequest`, `SourceLocation`, ...) are computed per-call from `params.Metadata` or `params.Ctx`:

```go
gcplogging.New(gcplogging.Config{
    Logger: gcpLogger,
    EntryFn: func(p loglayer.TransportParams, e *logging.Entry) {
        if md, ok := p.Metadata.(loglayer.Metadata); ok {
            if trace, ok := md["trace"].(string); ok {
                e.Trace = trace
            }
            if spanID, ok := md["spanId"].(string); ok {
                e.SpanID = spanID
            }
        }
    },
})
```

`EntryFn` runs after `Severity`, `Timestamp`, and `Payload` are populated, so it's free to overwrite them but rarely needs to.

### `Sync` mode

By default, `Logger.Log` queues entries and the SDK flushes them in batches. This is right for long-running services. For short-lived processes (Cloud Functions, CI tasks), queued entries can be lost when the process exits before the next flush. Set `Sync: true` to route through `Logger.LogSync` instead, which blocks until each entry is acknowledged.

```go
gcplogging.New(gcplogging.Config{
    Logger: gcpLogger,
    Sync:   true, // blocking dispatch; safe for short-lived processes
})
```

The transport also implements `io.Closer`. `loglayer.AddTransport` / `RemoveTransport` will call `Close()` on swap, which calls `Logger.Flush()` to drain pending async entries.

## Fatal Behavior

The transport never calls `os.Exit` or `panic` itself. `LogLevelFatal` and `LogLevelPanic` are mapped to GCP severities (Critical and Alert respectively) and dispatched normally; LogLayer's core decides whether the process terminates via `Config.DisableFatalExit`. See [Fatal Exits the Process](/logging-api/basic-logging#fatal-exits-the-process).

## Metadata Handling

Persistent fields and per-call metadata are merged into the JSON payload alongside the message:

```go
log.WithFields(loglayer.Fields{"requestId": "abc"}).
    WithMetadata(loglayer.Metadata{"durationMs": 42}).
    Info("served")
```

results in a `LogEntry.jsonPayload` of:

```json
{
  "message": "served",
  "requestId": "abc",
  "durationMs": 42
}
```

Map metadata merges at the payload root. Non-map metadata (structs, scalars) nest under the `metadata` key. Set [`Config.MetadataFieldName` on the core](/configuration#metadatafieldname) to nest map metadata under a fixed key uniformly.

## Level Mapping

| LogLayer Level | GCP Severity |
|----------------|--------------|
| `LogLevelTrace` | `Debug` |
| `LogLevelDebug` | `Debug` |
| `LogLevelInfo`  | `Info`  |
| `LogLevelWarn`  | `Warning` |
| `LogLevelError` | `Error` |
| `LogLevelFatal` | `Critical` |
| `LogLevelPanic` | `Alert` |

GCP has no dedicated trace severity, so `Trace` collapses into `Debug`. `Panic` maps to `Alert` (one above `Critical`) since loglayer's core panics the goroutine separately.

## GetLoggerInstance

`Transport.GetLoggerInstance()` returns the underlying `*logging.Logger`, useful when you need SDK features the transport doesn't expose (custom `EntryByteThreshold`, an `OnError` on the `*logging.Client`, etc.).

```go
underlying := log.GetLoggerInstance(transportID).(*logging.Logger)
```
