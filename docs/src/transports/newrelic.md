---
title: New Relic Transport
description: Ship logs to the New Relic Log Ingest API.
---

# New Relic Transport

<ModuleBadges path="transports/newrelic" />

Sends log entries to the [New Relic Log Ingest API](https://docs.newrelic.com/docs/logs/log-api/introduction-log-api/). Built on the [HTTP transport](/transports/http) with a New Relic-specific encoder (timestamp, level, log, attributes), site-aware URL, and `Api-Key` header. Includes attribute validation enforced at encode time (max 255 attributes, 255-char names, 4,094-char values). Log format matches the [TypeScript transport](https://loglayer.dev/transports/new-relic).

```sh
go get go.loglayer.dev/transports/newrelic
```

## Getting a License Key and Site

New Relic identifies your account with a **license key** (also called a user key) and a **site** (the region your account lives in). You need both.

To get a license key:

1. Sign in at the URL that matches your New Relic account (see the Site table below).
2. Navigate to **Account management** → **License keys**. Direct link: `https://<your-site>.newrelic.com/account management/license keys`.
3. Either copy an existing key or generate a new one. New Relic may hide the value after creation, so save it immediately.

Further information on New Relic API keys is available in the [official documentation](https://docs.newrelic.com/docs/apis/intro-apis/new-relic-api-keys/).

To pick the Site:

| Site code | Sign-in URL | Intake URL |
|---|---|---|
| `SiteUS` *(default)* | `newrelic.com` | `https://log-api.newrelic.com/log/v1` |
| `SiteEU` | `eu.newrelic.com` | `https://log-api.eu.newrelic.com/log/v1` |

If you signed up at the bare `newrelic.com`, you are on `SiteUS`. If your account is EU-based, use `SiteEU`.

The license key is a secret. Treat it like a password: load it from an environment variable or secret manager rather than hard-coding it in source.

## Basic Usage

```go
import (
    "os"

    "go.loglayer.dev/v3"
    "go.loglayer.dev/transports/newrelic"
)

tr := newrelic.New(newrelic.Config{
    LicenseKey: os.Getenv("NEW_RELIC_LICENSE_KEY"),
    Site:       newrelic.SiteUS, // or SiteEU
})
defer tr.Close()

log := loglayer.New(loglayer.Config{Transport: tr})
log = log.WithFields(loglayer.Fields{"requestId": "abc"})
log.WithMetadata(loglayer.Metadata{"durationMs": 42}).Info("served request")
```

The transport is async and batched (inherited from the HTTP transport, default 100 entries / 5 seconds). Always call `Close()` on shutdown to flush pending entries.

## Sites

`Site` controls the intake URL. Pick the one that matches your New Relic account:

| Site          | Intake URL                                        |
|---------------|---------------------------------------------------|
| `SiteUS` (default) | `https://log-api.newrelic.com/log/v1`       |
| `SiteEU`      | `https://log-api.eu.newrelic.com/log/v1`          |

### On-prem / custom URL

For on-prem deployments or when testing against a mock endpoint, set `Config.URL` directly. The override wins over `Site`:

```go
newrelic.New(newrelic.Config{
    LicenseKey: "...",
    URL:        "https://newrelic.internal.acme.com/log/v1",
    // Site is ignored when URL is set.
})
```

The transport rejects non-HTTPS URLs by default. To point at an `httptest.Server` or a local plain-HTTP proxy, also set `AllowInsecureURL: true`:

```go
srv := httptest.NewServer(http.HandlerFunc(...))
defer srv.Close()

tr := newrelic.New(newrelic.Config{
    LicenseKey:       "fake-for-tests",
    URL:              srv.URL,        // http:// from httptest
    AllowInsecureURL: true,           // required for non-HTTPS URLs
})
```

`AllowInsecureURL` is a test/debug ergonomic; leave it off for any URL that leaves your machine.

## Config

```go
type Config struct {
    transport.BaseConfig

    LicenseKey       string // required
    Site             Site   // default SiteUS; ignored when URL is set
    URL              string // overrides the Site-derived intake URL (on-prem / mock)
    AllowInsecureURL bool   // permit non-HTTPS URLs (httptest, local proxies)

    HTTP httptransport.Config // batching/client/error handling overrides
}
```

### `LicenseKey`

Required. Set as the `api-key` header on every request. `newrelic.New` panics with `newrelic.ErrLicenseKeyRequired` when this is empty; use `newrelic.Build(cfg) (*Transport, error)` if you load the key from an environment variable and want to handle the missing-config case explicitly.

### `HTTP`

Embedded `httptransport.Config` for batching, client timeout, error handling, and other HTTP-layer concerns. The `URL`, `Encoder`, and `api-key` header are set by the New Relic wrapper and cannot be overridden via this field.

```go
tr := newrelic.New(newrelic.Config{
    LicenseKey: key,
    HTTP: httptransport.Config{
        BatchSize:     500,
        BatchInterval: 2 * time.Second,
        Client:        &http.Client{Timeout: 10 * time.Second},
        OnError: func(err error, entries []httptransport.Entry) {
            metrics.Counter("newrelic.send.failed").Add(int64(len(entries)))
        },
    },
})
```

See the [HTTP transport docs](/transports/http) for the full HTTP config surface.

## Encoded Body Shape

Each log entry becomes one object in a JSON array. The format matches the [TypeScript New Relic transport](https://loglayer.dev/transports/new-relic):

```json
[
  {
    "timestamp":  1745616000123,
    "level":      "info",
    "log":        "served request",
    "attributes": {
      "requestId":  "abc",
      "durationMs": 42
    }
  }
]
```

Every object includes `timestamp` (epoch milliseconds), `level` (the log layer severity), and `log` (the message text). Persistent fields and metadata are merged under the `attributes` key with New Relic's API constraints enforced: maximum 255 attributes, 255-character attribute names, and string values truncated at 4,094 characters. Reserved fields (`timestamp`, `level`, `log`) are excluded from attributes to prevent collisions.

## Level → level Mapping

New Relic uses a `level` string per entry. The transport maps loglayer levels:

| LogLayer Level   | New Relic level  |
|------------------|------------------|
| `LogLevelTrace`  | `trace`          |
| `LogLevelDebug`  | `debug`          |
| `LogLevelInfo`   | `info`           |
| `LogLevelWarn`   | `warn`           |
| `LogLevelError`  | `error`          |
| `LogLevelFatal`  | `critical`       |
| `LogLevelPanic`  | `critical`       |

New Relic has no distinction between Fatal and Panic, so both map to `critical` (the highest-severity level in the Log Ingest API).

## API Limits

The transport enforces these limits at encode time ([reference](https://docs.newrelic.com/docs/logs/log-api/introduction-log-api/#limits)):

- 255 attributes maximum per event (excess attributes are dropped, not merged)
- 255 characters maximum per attribute name (longer names are dropped silently)
- String values truncated at 4,094 characters

New Relic also enforces these on the server side:

- 1MB maximum payload per POST (compression recommended)
- Payload must be UTF-8 encoded
- 4,094 characters stored in NRDB; longer values stored as a blob

The default `BatchSize` of 100 stays well under the 1MB payload limit for typical entries. If you bump `BatchSize` for higher throughput or ship large attributes, watch for the boundary.

## Closing

`newrelic.Transport` embeds `*httptransport.Transport`, so it has the same `Close() error` method. **Always call it on shutdown** so the in-flight batch is flushed:

```go
tr := newrelic.New(...)
defer tr.Close()
```

After `Close`, subsequent log calls drop the entry and invoke the underlying HTTP transport's `OnError` with `httptransport.ErrClosed`.

## Reaching the Underlying HTTP Transport

`newrelic.Transport` embeds `*httptransport.Transport`, so any HTTP-transport method works on it directly:

```go
tr := newrelic.New(...)
tr.Close()                      // from httptransport.Transport
tr.GetLoggerInstance()          // from httptransport.Transport (returns nil)
```

## Fatal Behavior

<!--@include: ./_partials/fatal-passthrough.md-->

Same async caveat as the underlying [HTTP transport](/transports/http#fatal-behavior): set `DisableFatalExit: true` and call `tr.Close()` before `os.Exit(1)` if you need guaranteed delivery of the fatal entry.
