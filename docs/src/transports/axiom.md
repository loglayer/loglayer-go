---
title: Axiom Transport
description: "Forward LogLayer entries to Axiom via github.com/axiomhq/axiom-go."
---

# Axiom Transport

<ModuleBadges path="transports/axiom" />

Ships structured logs to [Axiom](https://axiom.co) using the official [axiom-go SDK](https://github.com/axiomhq/axiom-go). The transport constructs a JSON object from each entry and sends it via `Client.Ingest()` as NDJSON.

Import path: `go.loglayer.dev/transports/axiom/v2`. Package name: `axiom`.

```sh
go get go.loglayer.dev/transports/axiom/v2
```

## Authenticating

Axiom authenticates with an API token. You can provide this to the transport in two ways:

1. **Pass the client directly**: Construct `*axiom.Client` yourself and pass it to the transport.
2. **Use environment variables**: The transport reads `AXIOM_TOKEN` if no client is provided.

| Env var | Purpose |
|---------|---------|
| `AXIOM_TOKEN` | API token with ingest permission. Used when constructing the client. |
| `AXIOM_ORG_ID` | Organization ID (required for personal tokens). |
| `AXIOM_DATASET` | Dataset name or ID to ingest logs into. |

### Using environment variables

```go
import (
    "github.com/axiomhq/axiom-go/axiom"
    "go.loglayer.dev/v2"
    "go.loglayer.dev/transports/axiom/v2"
)

// Client picks up AXIOM_TOKEN from the environment
client, err := axiom.NewClient(
    axiom.SetAPITokenConfig(os.Getenv("AXIOM_TOKEN")),
)
if err != nil {
    panic(err)
}

log := loglayer.New(loglayer.Config{
    Transport: axiom.New(axiom.Config{
        Client:      client,
        DatasetName: "my-logs",
    }),
})
```

## Basic Usage

```go
import (
    "context"

    "github.com/axiomhq/axiom-go/axiom"
    "go.loglayer.dev/v2"
    "go.loglayer.dev/transports/axiom/v2"
)

ctx := context.Background()
client, err := axiom.NewClient(
    axiom.SetAPITokenConfig("your-api-token"),
)
if err != nil {
    panic(err)
}

log := loglayer.New(loglayer.Config{
    Transport: axiom.New(axiom.Config{
        Client:      client,
        DatasetName: "my-logs",
    }),
})

log.Info("user signed in")
log.WithMetadata(map[string]any{"userId": 42}).Warn("retry exhausted")
```

## Config

```go
type Config struct {
    transport.BaseConfig

    Client       *axiom.Client
    DatasetName  string
    MessageField string
    OnError      func(error)
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `Client` | `*axiom.Client` | (required) | Constructed via `axiom.NewClient()` with authentication options. |
| `DatasetName` | `string` | (required) | Axiom dataset ID or name to ingest logs into. |
| `MessageField` | `string` | `"msg"` | The key under which the joined message text is placed in the JSON object. |
| `OnError` | `func(error)` | stderr | Called when `Client.Ingest()` returns an error. |

## Payload Shape

Each log entry is ingested as a JSON object:

- `msg`: the joined message text (configurable via `MessageField`)
- Persistent fields from `WithFields()`, merged at root
- The serialized error from `WithError()`
- Map metadata flattened at root, or any other metadata nested under `metadata`

```go
log.WithFields(map[string]any{"requestId": "abc"}).
    WithError(errors.New("timeout")).
    WithMetadata(map[string]any{"durationMs": 42}).
    Info("served")
```

results in:

```json
{
  "msg": "served",
  "requestId": "abc",
  "err": { "message": "timeout" },
  "durationMs": 42
}
```

## Metadata Handling

Map metadata (`loglayer.Metadata`) merges at the root of the JSON object. Non-map metadata (structs, scalars) nests under the `metadata` key by default.

Set [`Config.MetadataFieldName`](/configuration#metadatafieldname) on the core to nest all metadata under a fixed key.

## Level Mapping

LogLayer levels map directly to Axiom's expected level strings:

| LogLayer Level | Axiom Level |
|----------------|-------------|
| `LogLevelTrace` | `"trace"` |
| `LogLevelDebug` | `"debug"` |
| `LogLevelInfo`  | `"info"`  |
| `LogLevelWarn`  | `"warn"`  |
| `LogLevelError` | `"error"` |
| `LogLevelFatal` | `"fatal"` |
| `LogLevelPanic` | `"panic"` |

## GetLoggerInstance

`Transport.GetLoggerInstance()` returns the underlying `*axiom.Client`, useful for SDK features not exposed by the transport.

```go
underlying := log.GetLoggerInstance(transportID).(*axiom.Client)
```
