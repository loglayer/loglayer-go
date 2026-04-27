---
title: HTTP Transport
description: Generic batched HTTP POST transport with a pluggable encoder.
---

# HTTP Transport

The `http` transport ships log entries to an HTTP endpoint as JSON in async batches. Use it directly to talk to any log-ingestion API, or as the foundation for a service-specific wrapper (the [Datadog transport](/transports/datadog) is built on it).

```sh
go get go.loglayer.dev/transports/http
```

The directory is `transports/http`; the package name is `httptransport` to avoid colliding with `net/http`.

## Basic Usage

```go
import (
    "go.loglayer.dev"
    httptr "go.loglayer.dev/transports/http"
)

tr := httptr.New(httptr.Config{
    URL: "https://logs.example.com/ingest",
    Headers: map[string]string{
        "Authorization": "Bearer " + token,
    },
})

log := loglayer.New(loglayer.Config{Transport: tr})
log.Info("hello")

// On shutdown:
defer tr.Close() // flushes pending entries
```

By default the transport batches up to **100 entries** or every **5 seconds**, whichever comes first. The default encoder produces a JSON array of `{level, time, msg, ...fields, metadata?}` objects.

## How It Works

1. `SendToLogger` enqueues the entry into a buffered channel.
2. A background worker drains the channel into batches.
3. When the batch hits `BatchSize` or `BatchInterval` elapses, the worker calls `Encoder.Encode([]Entry)` and POSTs the body.
4. On shutdown, `Close()` drains the channel and flushes pending entries before returning.

The dispatch path (the `log.Info(...)` call) never blocks on network I/O. If the buffer is full, entries are dropped silently and `OnError` is invoked with `ErrBufferFull` so callers can observe loss.

## Config

```go
type Config struct {
    transport.BaseConfig

    URL     string            // required
    Method  string            // default POST
    Headers map[string]string // sent on every request

    Encoder Encoder           // default JSONArrayEncoder
    Client  *http.Client      // default has 30s timeout

    BatchSize     int           // default 100
    BatchInterval time.Duration // default 5s
    BufferSize    int           // default 1024

    OnError func(err error, entries []Entry) // default writes to os.Stderr
}
```

### `URL` and `Method`

Required. The transport POSTs (or whatever `Method` you set) the encoded body to `URL`. `httptr.New` panics with `httptr.ErrURLRequired` when `URL` is empty; use `httptr.Build(cfg) (*Transport, error)` if you load the URL from an environment variable and want to handle the missing-config case explicitly.

### `Headers`

Static headers added to every request. `Content-Type` is set automatically from the Encoder's return value but can be overridden here.

### `Encoder`

Serializes a batch of entries into the request body. The interface:

```go
type Encoder interface {
    Encode(entries []Entry) (body []byte, contentType string, err error)
}
```

`Entry` is the canonical shape passed to encoders:

```go
type Entry struct {
    Level    loglayer.LogLevel
    Time     time.Time
    Messages []any
    Data     map[string]any // fields + error (may be nil)
    Metadata any            // raw value passed to WithMetadata
}
```

Use `EncoderFunc` to adapt a function:

```go
encoder := httptr.EncoderFunc(func(entries []httptr.Entry) ([]byte, string, error) {
    // produce ndjson, protobuf, gzipped JSON, whatever
    return body, "application/x-ndjson", nil
})
```

The default `JSONArrayEncoder` produces:

```json
[
  {"level":"info","time":"2026-04-26T12:00:00Z","msg":"hello","requestId":"abc"}
]
```

Map metadata merges at the root; non-map metadata (struct, scalar, slice) lands under the `metadata` key.

### `Client`

The `*http.Client` used to send requests. Defaults to a fresh client with a 30-second timeout. Override to plug in retries, custom transports (proxies, mTLS, OpenTelemetry instrumentation), or shorter timeouts.

### `BatchSize` and `BatchInterval`

The worker flushes whenever it accumulates `BatchSize` entries OR `BatchInterval` elapses since the last flush. Tune for your endpoint's batch limits and latency tolerance:

| Use case               | BatchSize | BatchInterval |
|------------------------|-----------|---------------|
| Low-latency dev/debug  | 1         | 100ms         |
| Standard production    | 100       | 5s            |
| High-volume shipping   | 500-1000  | 1-5s          |

### `BufferSize`

Size of the internal channel buffering entries between `SendToLogger` and the worker. Defaults to 1024. Larger values absorb traffic spikes; smaller values drop sooner under sustained backpressure.

When the buffer is full, entries are **dropped** and `OnError(ErrBufferFull, [entry])` is called. The dispatch path (the `log.Info(...)` caller's goroutine) never blocks.

### `OnError`

Called when something goes wrong:

| Error                | When                                    |
|----------------------|-----------------------------------------|
| `ErrBufferFull`      | Buffer was full; the entry was dropped  |
| `ErrClosed`          | `SendToLogger` was called after `Close` |
| `*HTTPError`         | Server returned status >= 400           |
| Wrapped encode error | The encoder returned an error           |
| Wrapped send error   | `client.Do` returned an error           |

The default writes a one-line message to `os.Stderr`. Override to plumb errors into a separate logger, a metrics counter, or a dead-letter queue:

```go
OnError: func(err error, entries []httptr.Entry) {
    if errors.Is(err, httptr.ErrBufferFull) {
        droppedCounter.Inc()
        return
    }
    var httpErr *httptr.HTTPError
    if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusTooManyRequests {
        // implement retry via re-enqueue, write to dead-letter queue, etc.
    }
    backupLogger.WithError(err).Warn("log shipping failed")
}
```

## Closing

`*Transport` exposes `Close() error` to drain the queue and stop the worker. Call it on shutdown:

```go
tr := httptr.New(...)
log := loglayer.New(loglayer.Config{Transport: tr, ...})
defer tr.Close()
```

After `Close`, subsequent `SendToLogger` calls drop the entry and invoke `OnError(ErrClosed, nil)`. `Close` is idempotent.

## Custom Encoder Example

To send NDJSON (one object per line) instead of a JSON array:

```go
ndjson := httptr.EncoderFunc(func(entries []httptr.Entry) ([]byte, string, error) {
    var buf bytes.Buffer
    enc := json.NewEncoder(&buf)
    for _, e := range entries {
        obj := map[string]any{
            "level": e.Level.String(),
            "time":  e.Time.UTC().Format(time.RFC3339Nano),
            "msg":   transport.JoinMessages(e.Messages),
        }
        for k, v := range e.Data {
            obj[k] = v
        }
        if err := enc.Encode(obj); err != nil {
            return nil, "", err
        }
    }
    return buf.Bytes(), "application/x-ndjson", nil
})

tr := httptr.New(httptr.Config{
    URL:     "https://logs.example.com/ndjson",
    Encoder: ndjson,
})
```

## Fatal Behavior

This transport writes fatal entries normally. The core decides whether to call `os.Exit(1)` based on `Config.DisableFatalExit`. See [Fatal Exits the Process](/logging-api/basic-logging#fatal-exits-the-process).

Note: a Fatal log followed by `os.Exit(1)` may not finish flushing if the worker hasn't picked up the entry yet. If you need guaranteed shipping for fatal entries, set `Config.DisableFatalExit: true` on the LogLayer config and call `tr.Close()` followed by `os.Exit(1)` yourself.

## Level Mapping

The `level` field in the default encoder uses the standard loglayer names:

| LogLayer Level   | level string |
|------------------|--------------|
| `LogLevelTrace`  | `trace`      |
| `LogLevelDebug`  | `debug`      |
| `LogLevelInfo`   | `info`       |
| `LogLevelWarn`   | `warn`       |
| `LogLevelError`  | `error`      |
| `LogLevelFatal`  | `fatal`      |

For Datadog-specific status mapping, use the [Datadog transport](/transports/datadog) which provides the right strings.
