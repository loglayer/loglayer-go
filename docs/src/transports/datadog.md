---
title: Datadog Transport
description: Ship logs to the Datadog Logs HTTP intake API.
---

# Datadog Transport

<ModuleBadges path="transports/datadog" />

Sends log entries to Datadog's [Logs HTTP intake API](https://docs.datadoghq.com/api/latest/logs/#send-logs). Built on the [HTTP transport](/transports/http) with a Datadog-specific encoder, site-aware URL, and `DD-API-KEY` header.

```sh
go get go.loglayer.dev/transports/datadog
```

## Basic Usage

```go
import (
    "go.loglayer.dev"
    "go.loglayer.dev/transports/datadog"
)

tr := datadog.New(datadog.Config{
    APIKey:   os.Getenv("DD_API_KEY"),
    Site:     datadog.SiteUS1, // or SiteEU, SiteUS3, SiteUS5, SiteAP1
    Source:   "go",
    Service:  "checkout-api",
    Hostname: hostname,
    Tags:     "env:prod,team:platform",
})
defer tr.Close()

log := loglayer.New(loglayer.Config{Transport: tr})
log = log.WithFields(loglayer.Fields{"requestId": "abc"})
log.WithMetadata(loglayer.Metadata{"durationMs": 42}).Info("served request")
```

The transport is async and batched (inherited from the HTTP transport, default 100 entries / 5 seconds). Always call `Close()` on shutdown to flush pending entries.

## Sites

`Site` controls the intake URL. Pick the one that matches your Datadog account:

| Site            | Intake URL                                                     |
|-----------------|----------------------------------------------------------------|
| `SiteUS1` (default) | `https://http-intake.logs.datadoghq.com/api/v2/logs`       |
| `SiteUS3`       | `https://http-intake.logs.us3.datadoghq.com/api/v2/logs`       |
| `SiteUS5`       | `https://http-intake.logs.us5.datadoghq.com/api/v2/logs`       |
| `SiteEU`        | `https://http-intake.logs.datadoghq.eu/api/v2/logs`            |
| `SiteAP1`       | `https://http-intake.logs.ap1.datadoghq.com/api/v2/logs`       |

### On-prem / custom URL

For Datadog on-prem deployments or when testing against a mock endpoint, set `Config.URL` directly. The override wins over `Site`:

```go
datadog.New(datadog.Config{
    APIKey: "...",
    URL:    "https://datadog.internal.acme.com/api/v2/logs",
    // Site is ignored when URL is set.
})
```

The transport rejects non-HTTPS URLs by default. To point at an `httptest.Server` or a local plain-HTTP proxy, also set `AllowInsecureURL: true`:

```go
srv := httptest.NewServer(http.HandlerFunc(...))
defer srv.Close()

tr := datadog.New(datadog.Config{
    APIKey:           "fake-for-tests",
    URL:              srv.URL,        // http:// from httptest
    AllowInsecureURL: true,           // required for non-HTTPS URLs
})
```

`AllowInsecureURL` is a test/debug ergonomic; leave it off for any URL that leaves your machine.

## Config

```go
type Config struct {
    transport.BaseConfig

    APIKey           string  // required
    Site             Site    // default SiteUS1; ignored when URL is set
    URL              string  // overrides the Site-derived intake URL (on-prem / mock)
    AllowInsecureURL bool    // permit non-HTTPS URLs (httptest, local proxies)

    Source   string  // ddsource (e.g. "go")
    Service  string  // service name
    Hostname string  // host name
    Tags     string  // ddtags, comma-separated key:value

    HTTP httptransport.Config // batching/client/error handling overrides
}
```

### `APIKey`

Required. Set as the `DD-API-KEY` header on every request. `datadog.New` panics with `datadog.ErrAPIKeyRequired` when this is empty; use `datadog.Build(cfg) (*Transport, error)` if you load the key from an environment variable and want to handle the missing-config case explicitly.

### `Source`, `Service`, `Hostname`, `Tags`

Optional Datadog reserved attributes. Empty values are omitted from the payload.

| Field      | Datadog field | Recommended value                          |
|------------|---------------|--------------------------------------------|
| `Source`   | `ddsource`    | `"go"` (or your framework name)            |
| `Service`  | `service`     | Your service/application name              |
| `Hostname` | `hostname`    | Resolved hostname from `os.Hostname()`     |
| `Tags`     | `ddtags`      | `"env:prod,team:platform,version:1.2.3"`   |

### `HTTP`

Embedded `httptransport.Config` for batching, client timeout, error handling, and other HTTP-layer concerns. The `URL`, `Encoder`, and `DD-API-KEY` header are set by the Datadog wrapper and cannot be overridden via this field.

```go
tr := datadog.New(datadog.Config{
    APIKey: key,
    HTTP: httptransport.Config{
        BatchSize:     500,
        BatchInterval: 2 * time.Second,
        Client:        &http.Client{Timeout: 10 * time.Second},
        OnError: func(err error, entries []httptransport.Entry) {
            metrics.Counter("datadog.send.failed").Add(int64(len(entries)))
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
    "ddsource":  "go",
    "service":   "checkout-api",
    "hostname":  "ip-10-0-0-1",
    "ddtags":    "env:prod,team:platform",
    "status":    "info",
    "message":   "served request",
    "date":      "2026-04-26T12:00:00.123Z",
    "requestId": "abc",
    "durationMs": 42
  }
]
```

Map metadata merges at the root; non-map metadata (struct, scalar, slice) lands under the `metadata` key. Persistent fields (`WithFields`) merge at the root.

## Level → Status Mapping

Datadog uses a `status` string per entry. The transport maps loglayer levels:

| LogLayer Level   | Datadog status |
|------------------|----------------|
| `LogLevelDebug`  | `debug`        |
| `LogLevelInfo`   | `info`         |
| `LogLevelWarn`   | `warning`      |
| `LogLevelError`  | `error`        |
| `LogLevelFatal`  | `critical`     |

## API Limits

Datadog's intake has these limits ([reference](https://docs.datadoghq.com/api/latest/logs/#send-logs)):

- 5MB max body size per request
- 1MB max single log entry
- 1,000 max log entries per array

The default `BatchSize` of 100 stays well under all of these. If you bump `BatchSize` for higher throughput, keep it under 1,000 and watch the body-size limit if your entries are large.

## Closing

`Datadog.Transport` embeds `*httptransport.Transport`, so it has the same `Close() error` method. **Always call it on shutdown** so the in-flight batch is flushed:

```go
tr := datadog.New(...)
defer tr.Close()
```

After `Close`, subsequent log calls drop the entry and invoke the underlying HTTP transport's `OnError` with `httptransport.ErrClosed`.

## Reaching the Underlying HTTP Transport

`datadog.Transport` embeds `*httptransport.Transport`, so any HTTP-transport method works on it directly:

```go
tr := datadog.New(...)
tr.Close()                      // from httptransport.Transport
tr.GetLoggerInstance()          // from httptransport.Transport (returns nil)
```

## Live Test

A build-tagged test (`//go:build livetest`) ships with the package and hits the real Datadog intake. It's gated by build tag *and* by an env-var check so normal `go test ./...` runs ignore it entirely.

```sh
# Minimal
DD_API_KEY=<your-key> go test -tags=livetest -v -run TestLive_Datadog ./transports/datadog/

# With all options
DD_API_KEY=<your-key> \
DD_SITE=us1 \
DD_SOURCE=go-loglayer-livetest \
DD_SERVICE=loglayer-go-livetest \
DD_HOSTNAME=$(hostname) \
DD_TAGS=env:livetest,team:platform \
  go test -tags=livetest -v -run TestLive_Datadog ./transports/datadog/
```

The test sends two entries (one Info with persistent fields, one Warn with metadata) and fails if the intake returns any error. It prints a search query you can paste into the Datadog Logs Explorer to verify the entries landed:

```
source:go-loglayer-livetest @livetest_id:<random-hex>
```

Indexing latency in Datadog is typically 5-60 seconds. Without `DD_API_KEY` the test skips with a clear message, so it's safe to leave the build tag in CI without leaking errors.

| Env var       | Required | Default                  | Purpose                        |
|---------------|----------|--------------------------|--------------------------------|
| `DD_API_KEY`  | Yes      | (none)                   | Datadog API key                |
| `DD_SITE`     | No       | `us1`                    | Datadog region                 |
| `DD_SOURCE`   | No       | `go-loglayer-livetest`   | `ddsource` field               |
| `DD_SERVICE`  | No       | `loglayer-go-livetest`   | `service` field                |
| `DD_HOSTNAME` | No       | empty                    | `hostname` field               |
| `DD_TAGS`     | No       | `env:livetest`           | `ddtags` field                 |

## Fatal Behavior

<!--@include: ./_partials/fatal-passthrough.md-->

Same async caveat as the underlying [HTTP transport](/transports/http#fatal-behavior): set `DisableFatalExit: true` and call `tr.Close()` before `os.Exit(1)` if you need guaranteed delivery of the fatal entry.
