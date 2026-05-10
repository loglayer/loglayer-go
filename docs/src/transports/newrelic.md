---
title: New Relic Transport
description: Ships log entries to New Relic Log API as NDJSON.
---

# New Relic Transport

<ModuleBadges path="transports/newrelic" />

The `newrelic` transport ships log entries to the [New Relic Log API](https://docs.newrelic.com/docs/logs/log-api/introduction-log-api/) as NDJSON. Authentication is via the `Api-Key` header from `Config.APIKey` by default. `X-License-Key` is optionally supported for accounts that require it. Built on `transports/http` for async batching.

```sh
go get go.loglayer.dev/transports/newrelic
```

## Basic Usage

```go
import (
	"go.loglayer.dev/v2"
	"go.loglayer.dev/transports/newrelic/v2"
)

tr := newrelic.New(newrelic.Config{
	APIKey: "your-new-relic-api-key",
	Zone:   newrelic.ZoneUS, // optional; ZoneUS is the default
})

log := loglayer.New(loglayer.Config{Transport: tr})
log.Info("hello")

defer tr.Close() // flushes pending entries
```

## Config

| Field        | Type                | Default | Description |
|--------------|---------------------|---------|-------------|
| `APIKey`     | `string`            | (required) | New Relic user key for the `Api-Key` header |
| `LicenseKey` | `string`            | empty   | Optional `X-License-Key` header value |
| `Zone`       | `newrelic.IntakeZone` | `ZoneUS` | Region selection |
| `URL`        | `string`            | derived from `Zone` | Custom endpoint override |
| `Hostname`   | `string`            | empty   | Per-entry `hostname` attribute |
| `AllowInsecureURL` | `bool`        | false   | Permit non-https `Config.URL` |
| `HTTP`       | `httptr.Config`     | batching defaults | HTTP transport overrides |

`Config.HTTP.URL` and `Config.HTTP.Encoder` cannot be set (the transport sets them itself). Custom headers go through `Config.HTTP.Headers`.

### Zone

| Constant   | Region        | Endpoint                                     |
|------------|---------------|----------------------------------------------|
| `ZoneUS`   | US (default)  | `https://log-api.newrelic.com/log/v1`        |
| `ZoneEU`   | EU            | `https://log-api.eu.newrelic.com/log/v1`     |

### API Key and License Key

The `APIKey` and `LicenseKey` fields are tagged `json:"-"` and redacted in `Config.String()` to prevent accidental exposure when the config is logged or passed through a JSON transport.

## Log Format

Each entry serializes as a JSON object with `timestamp`, `message`, `loglevel`, and optionally `hostname`. All user fields and metadata merge as root-level attributes. Lines are newline-delimited (NDJSON).

```json
{"hostname":"prod-web-01","loglevel":"warning","message":"high latency","requestId":"abc","timestamp":"2026-04-26T12:00:00.000Z"}
```

### Level Mapping

| LogLayer Level   | New Relic level |
|------------------|-----------------|
| `LogLevelTrace`  | `debug`         |
| `LogLevelDebug`  | `debug`         |
| `LogLevelInfo`   | `info`          |
| `LogLevelWarn`   | `warning`       |
| `LogLevelError`  | `error`         |
| `LogLevelFatal`  | `critical`      |
| `LogLevelPanic`  | `critical`      |

## Fatal Behavior

<!--@include: ./_partials/fatal-passthrough.md-->
