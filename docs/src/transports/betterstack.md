---
title: Better Stack Transport
description: Ship logs to Better Stack's HTTP intake API.
---

# Better Stack Transport

<ModuleBadges path="transports/betterstack" />

Sends log entries to [Better Stack](https://betterstack.com) via their HTTP Logs Endpoint. Built on the [HTTP transport](/transports/http) with a Better Stack-specific encoder and bearer token authentication.

```sh
go get go.loglayer.dev/transports/betterstack
```

## Getting a Source Token

Better Stack identifies your log source with a **Source Token**. You need this to send logs.

To get the source token:

1. Sign in to Better Stack at [https://betterstack.com](https://betterstack.com).
2. Navigate to **Logs** → **Sources**.
3. Either copy an existing source's token or create a new log source and copy its token.
4. Store the token securely—treat it like a password.

The source token is a secret. Load it from an environment variable or secret manager rather than hard-coding it in source.

## Basic Usage

```go
import (
    "os"
    "go.loglayer.dev/v3"
    "go.loglayer.dev/transports/betterstack"
)

tr := betterstack.New(betterstack.Config{
    SourceToken: os.Getenv("BETTERSTACK_SOURCE_TOKEN"),
})
defer tr.Close()

log := loglayer.New(loglayer.Config{Transport: tr})
log = log.WithFields(loglayer.Fields{"requestId": "abc"})
log.WithMetadata(loglayer.Metadata{"durationMs": 42}).Info("served request")
```

The transport is async and batched (inherited from the HTTP transport, default 100 entries / 5 seconds). Always call `Close()` on shutdown to flush pending entries.

## URL

By default, logs are sent to Better Stack's public intake endpoint:

| Setting      | Value                                 |
|--------------|---------------------------------------|
| **URL**      | `https://in.logs.betterstack.com`     |

### On-prem / custom URL

When testing against a mock endpoint or using a proxy, set `Config.URL` directly:

```go
tr := betterstack.New(betterstack.Config{
    SourceToken: "fake-for-tests",
    URL:         "https://logs.internal.acme.com",  // on-prem intake URL
})
```

The transport rejects non-HTTPS URLs by default. To point at an `httptest.Server` or a local plain-HTTP proxy, also set `AllowInsecureURL: true`:

```go
srv := httptest.NewServer(http.HandlerFunc(...))
defer srv.Close()

tr := betterstack.New(betterstack.Config{
    SourceToken:      "fake-for-tests",
    URL:              srv.URL,        // http:// from httptest
    AllowInsecureURL: true,           // required for non-HTTPS URLs
})
```

`AllowInsecureURL` is a test/debug ergonomic; leave it off for any URL that leaves your machine.

## Config

```go
type Config struct {
    transport.BaseConfig

    SourceToken      string  // required
    URL              string  // default https://in.logs.betterstack.com
    AllowInsecureURL bool    // permit non-HTTPS URLs (httptest, local proxies)
    TimestampField   string  // field name for timestamp, default "dt"

    HTTP httptransport.Config // BatchSize, BatchInterval, Client timeout, OnError; URL, Encoder, and Headers cannot be overridden
}
```

### `SourceToken`

Required. Set as the Bearer token in the `Authorization` header on every request (`Bearer <source-token>`). `betterstack.New` panics with `betterstack.ErrSourceTokenRequired` when this is empty; use `betterstack.Build(cfg) (*Transport, error)` if you load the token from an environment variable and want to handle the missing-config case explicitly.

### `URL`

Optional. Overrides the default Better Stack intake endpoint. When set, the `AllowInsecureURL` config knob controls whether non-HTTPS URLs are permitted.

### `AllowInsecureURL`

Optional. When `true`, permits non-HTTPS URLs (useful for testing with `httptest.Server`). Defaults to `false` to prevent accidental plaintext logging in production.

### `TimestampField`

Optional. The field name used for timestamps in the log entry. Defaults to `"dt"` (Better Stack's convention). You can change this if your Better Stack source expects a different timestamp field name.

### `HTTP`

Embedded `httptransport.Config` for batching, client timeout, error handling, and other HTTP-layer concerns. The `URL`, `Encoder`, and `Authorization` header are set by the Better Stack wrapper and cannot be overridden via this field.

```go
tr := betterstack.New(betterstack.Config{
    SourceToken: token,
    HTTP: httptransport.Config{
        BatchSize:     500,
        BatchInterval: 2 * time.Second,
        Client:        &http.Client{Timeout: 10 * time.Second},
        OnError: func(err error, entries []httptransport.Entry) {
            metrics.Counter("betterstack.send.failed").Add(int64(len(entries)))
        },
    },
})
```

See the [HTTP transport docs](/transports/http) for the full HTTP config surface.

## Encoded Body Shape

Each log entry becomes one object in a JSON array:

```json
[
  {
    "dt": "2026-05-06T14:30:00.123Z",
    "level": "info",
    "message": "served request",
    "requestId": "abc",
    "metadata": {
      "durationMs": 42
    }
  }
]
```

- **Timestamp**: Controlled by `TimestampField` (default `"dt"`), formatted as ISO 8601 UTC.
- **Level**: String representation of the log level (trace, debug, info, warn, error, fatal, panic).
- **Message**: The log message string.
- **Fields**: Persistent fields from `WithFields()` and `Child()` are merged at the root level.
- **Metadata**: Nests under `MetadataFieldName` (default `"metadata"`, configurable via `loglayer.Config.MetadataFieldName`).

Persistent fields (`WithFields`) and metadata (`WithMetadata`) follow the [core placement rules](/configuration#fieldskey): when `FieldsKey` is empty, fields merge at the root of each log object; metadata nests under `MetadataFieldName` (default `"metadata"`). Set either knob on `loglayer.Config` to nest under a configured key instead.

## Level Mapping

Better Stack supports these log levels. The transport maps loglayer levels:

| LogLayer Level   | Better Stack level |
|------------------|--------------------|
| `LogLevelTrace`  | `"trace"`          |
| `LogLevelDebug`  | `"debug"`          |
| `LogLevelInfo`   | `"info"`           |
| `LogLevelWarn`   | `"warn"`           |
| `LogLevelError`  | `"error"`          |
| `LogLevelFatal`  | `"fatal"`          |
| `LogLevelPanic`  | `"panic"`          |

## API Limits

Better Stack's intake has these limits:

- Max body size per request: **5MB**
- Max single log entry: **1MB**
- Max entries per array: **1,000**

The default `BatchSize` of 100 stays well under all of these. If you bump `BatchSize` for higher throughput, keep it under 1,000 and watch the body-size limit if your entries are large.

## Closing

`BetterStack.Transport` embeds `*httptransport.Transport`, so it has the same `Close() error` method. **Always call it on shutdown** so the in-flight batch is flushed:

```go
tr := betterstack.New(...)
defer tr.Close()
```

After `Close`, subsequent log calls drop the entry and invoke the underlying HTTP transport's `OnError` with `httptransport.ErrClosed`.

## Reaching the Underlying HTTP Transport

`betterstack.Transport` embeds `*httptransport.Transport`, so any HTTP-transport method works on it directly:

```go
tr := betterstack.New(...)
tr.Close()                      // from httptransport.Transport
tr.GetLoggerInstance()          // from httptransport.Transport (returns nil)
```

## Fatal Behavior

<!--@include: ./_partials/fatal-passthrough.md-->

Same async caveat as the underlying [HTTP transport](/transports/http#fatal-behavior): set `DisableFatalExit: true` and call `tr.Close()` before `os.Exit(1)` if you need guaranteed delivery of the fatal entry.
