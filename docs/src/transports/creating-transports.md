---
title: Creating Transports
description: Implement the Transport interface to send LogLayer entries anywhere.
---

# Creating Transports

A transport is anything implementing four methods:

```go
type Transport interface {
    ID() string
    IsEnabled() bool
    SendToLogger(params loglayer.TransportParams)
    GetLoggerInstance() any
}
```

The `transport.BaseTransport` struct in this module handles `ID`, `IsEnabled`, and level filtering, embed it and you only need to implement `SendToLogger` and `GetLoggerInstance`.

## Minimal Example

```go
package mytransport

import (
    "fmt"
    "io"

    "go.loglayer.dev"
    "go.loglayer.dev/transport"
)

type Config struct {
    transport.BaseConfig
    Writer io.Writer
}

type Transport struct {
    transport.BaseTransport
    cfg Config
}

func New(cfg Config) *Transport {
    return &Transport{
        BaseTransport: transport.NewBaseTransport(cfg.BaseConfig),
        cfg:           cfg,
    }
}

func (t *Transport) ID() string                  { return t.BaseTransport.ID() }
func (t *Transport) IsEnabled() bool             { return t.BaseTransport.IsEnabled() }
func (t *Transport) GetLoggerInstance() any      { return nil }

func (t *Transport) SendToLogger(p loglayer.TransportParams) {
    if !t.BaseTransport.ShouldProcess(p.LogLevel) {
        return
    }

    fmt.Fprintf(t.cfg.Writer, "[%s] %v %v\n", p.LogLevel, p.Messages, p.Data)
}
```

That's the whole shape. From here it's a question of how you want to render the entry.

## What's in TransportParams

```go
type TransportParams struct {
    LogLevel LogLevel
    Messages []any   // already prefix-applied
    Data     Data    // assembled fields + error map; nil when both are absent. Use len(Data) > 0 to check.
    Metadata any     // raw value passed to WithMetadata, your transport decides serialization
    Err      error
    Fields   Fields
    Ctx      context.Context // per-call WithCtx value, or nil
}
```

`Data` is the convenience map combining fields + error. `Metadata` is `any`, you choose how to render it. `Err` and `Fields` are also exposed raw if you want to inspect them directly.

## Handling `any` Metadata

Map metadata is the common case; structs and other values come through too. The pattern used by the renderer transports:

```go
func metadataAsMap(v any) map[string]any {
    if v == nil {
        return nil
    }
    if m, ok := v.(map[string]any); ok {
        return m
    }
    b, err := json.Marshal(v)
    if err != nil {
        return nil
    }
    var m map[string]any
    _ = json.Unmarshal(b, &m)
    return m
}
```

If your backend can render structs natively (zerolog, zap, slog), prefer that path, skip the JSON roundtrip. The point of LogLayer's "metadata is any" design is exactly to let transports use the cheapest serialization for the runtime they're targeting.

## Level Filtering

`transport.BaseTransport.ShouldProcess(level)` returns false when:

- The transport is disabled (`SetEnabled(false)`), or
- The level is below the transport's `BaseConfig.Level`.

Always call it at the top of `SendToLogger`. The core LogLayer also filters by its own level state before reaching your transport, `ShouldProcess` is the second gate.

## Returning the Underlying Logger

If your transport wraps a third-party library, return that library from `GetLoggerInstance`. Callers can use it for backend-specific features the LogLayer API doesn't cover:

```go
func (t *Transport) GetLoggerInstance() any { return t.underlying }
```

For transports with no underlying library (anything you write from scratch), return `nil`.

## Concurrency

`SendToLogger` may be called from any goroutine. Make sure whatever you're writing to is safe, `os.Stdout`, `bytes.Buffer` (with a mutex), or a pre-locked `io.Writer`. The built-in transports rely on the writer being concurrency-safe; if yours isn't, wrap it.

## Don't Mutate TransportParams

When multiple transports are configured, they share the same `TransportParams`. Don't modify `params.Data`, `params.Messages`, or `params.Metadata` in place, copy first if you need to transform.

## Handling Errors

`SendToLogger` doesn't return an error. The dispatch path can't propagate transport failures back to the caller, so error handling is the transport's responsibility:

- **Synchronous renderer transports** (writing to `io.Writer`, terminal, file): if the write fails, there's nowhere to escalate. Print to `os.Stderr` and continue. Don't panic, don't `os.Exit`. A logging library that takes down the host process on a transient I/O hiccup is a bug.
- **Async / network transports** (HTTP, Datadog, anything with a worker goroutine and batching): expose an `OnError func(err error, ...)` field on your `Config` so the application can decide. Built-in transports follow this pattern, see [transports/http](/transports/http) and [transports/datadog](/transports/datadog) for the canonical shape.
- **Wrapper transports** (zerolog, zap, slog, etc.): the underlying library has its own error path (zap's `Sync` errors, zerolog's writer errors). Forward them or let the user reach the underlying logger via `GetLoggerInstance` and inspect there.

Transports that drop entries silently are valid: a logging library should never block, panic, or crash on its own write failure. But always make the failure observable somehow, even if it's just an `OnError` callback the user can hook into.

## Testing your transport

Drive entries through your transport via a real `*loglayer.LogLayer` and assert on whatever your transport actually produced (a buffer, a captured request, a wrapped logger's calls). The pattern mirrors the built-in transport tests:

```go
import (
    "bytes"
    "testing"

    "go.loglayer.dev"
    "go.loglayer.dev/transport"
)

func TestMyTransport_Basic(t *testing.T) {
    buf := &bytes.Buffer{}
    tr := mytransport.New(mytransport.Config{
        BaseConfig: transport.BaseConfig{ID: "test"},
        Writer:     buf,
    })
    log := loglayer.New(loglayer.Config{
        Transport:        tr,
        DisableFatalExit: true,
    })

    log.WithFields(loglayer.Fields{"k": "v"}).Info("served")

    // Assert on whatever shape your transport produced.
    if !strings.Contains(buf.String(), `"k":"v"`) {
        t.Errorf("k=v missing from output: %q", buf.String())
    }
}
```

For wrapper transports (those that hand entries off to a third-party logger), assert on the wrapped logger's output rather than the transport's. The slog/zerolog/zap test files in `transports/` show this pattern.

Cover the level-filtering case, the `MetadataFieldName` non-map path, and `WithCtx` propagation when applicable. The existing wrapper-transport test files are good templates: same structure, same assertion shape.
