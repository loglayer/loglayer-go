---
title: Writers
description: "Plug any io.Writer as the output sink for a renderer transport: files, rotating files, buffers, tees, network sockets."
---

# Writers

The three renderer transports ([Structured](/transports/structured), [Console](/transports/console), [Pretty](/transports/pretty)) expose a `Writer io.Writer` field on their Config. Whatever satisfies `io.Writer` is a valid output sink. You don't need a custom transport to send entries somewhere new; pick a renderer and plug a writer in.

```go
log := loglayer.New(loglayer.Config{
    Transport: structured.New(structured.Config{
        Writer: anyIoWriter, // os.Stdout, *os.File, *lumberjack.Logger, ...
    }),
})
```

## Defaults

| Transport | Default sink |
|-----------|-------------|
| `structured` | `os.Stdout` |
| `pretty` | `os.Stdout` |
| `console` | `os.Stdout` for debug/info, `os.Stderr` for warn/error/fatal |
| `testing` | In-memory; the Writer field is intentionally absent |

## Recipes

### File (basic, no rotation)

```go
import (
    "os"
    "go.loglayer.dev"
    "go.loglayer.dev/transports/structured"
)

f, err := os.OpenFile("/var/log/app.log",
    os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
if err != nil {
    panic(err)
}
defer f.Close()

log := loglayer.New(loglayer.Config{
    Transport: structured.New(structured.Config{Writer: f}),
})
```

The file grows forever. For rotation use [`transports/lumberjack`](/transports/lumberjack) or the lumberjack-as-Writer recipe below.

### Rotating file (lumberjack into any renderer)

The dedicated [lumberjack](/transports/lumberjack) transport already does this for JSON output. Use the recipe below when you want rotation under a non-JSON renderer ([Pretty](/transports/pretty), [Console](/transports/console)).

```go
import (
    "gopkg.in/natefinch/lumberjack.v2"
    "go.loglayer.dev"
    "go.loglayer.dev/transports/pretty"
)

rotator := &lumberjack.Logger{
    Filename:   "/var/log/app.log",
    MaxSize:    100, // MB
    MaxBackups: 7,
    Compress:   true,
}
defer rotator.Close()

log := loglayer.New(loglayer.Config{
    Transport: pretty.New(pretty.Config{Writer: rotator}),
})
```

### Tee to multiple sinks

`io.MultiWriter` duplicates writes to every wrapped writer. One transport, one render pass, two destinations.

```go
import (
    "io"
    "os"
)

log := loglayer.New(loglayer.Config{
    Transport: structured.New(structured.Config{
        Writer: io.MultiWriter(os.Stdout, file),
    }),
})
```

::: warning Head-of-line blocking
`io.MultiWriter` writes to each wrapped writer sequentially in the calling goroutine. A slow sink (e.g. a network connection) blocks every other sink and stalls the caller. Use multiple LogLayer transports if any sink can be slow.
:::

For finer control (different formats per sink, per-sink level filtering, independent OnError), prefer multiple LogLayer transports instead. See [Multiple Transports](/transports/multiple-transports).

### Buffer (ad-hoc capture)

A `*bytes.Buffer` works but isn't safe for concurrent writes. For real test assertions on log output, use the dedicated [testing transport](/transports/testing) instead, which captures structured `LogLine` values rather than rendered bytes.

```go
import (
    "bytes"
    "go.loglayer.dev"
    "go.loglayer.dev/transports/structured"
)

var buf bytes.Buffer
log := loglayer.New(loglayer.Config{
    Transport:        structured.New(structured.Config{Writer: &buf}),
    DisableFatalExit: true,
})

log.Info("hello")
// buf.String(): {"level":"info","time":"...","msg":"hello"}
```

### Network socket (TCP, UDP, syslog)

Anything that satisfies `io.Writer` works, including `net.Conn`. The example below ships one JSON object per line over a plain TCP connection (some syslog daemons accept this directly).

```go
import (
    "net"
    "go.loglayer.dev"
    "go.loglayer.dev/transports/structured"
)

conn, err := net.Dial("tcp", "logsink.local:514")
if err != nil {
    panic(err)
}
defer conn.Close()

log := loglayer.New(loglayer.Config{
    Transport: structured.New(structured.Config{Writer: conn}),
})
```

::: warning Synchronous network writes block the dispatch path
Each log call blocks until `conn.Write` returns. A slow or unreachable sink stalls the caller's goroutine. For production network shipping use the [HTTP transport](/transports/http) (async batching, configurable buffer, `OnError` callback) or wrap your `net.Conn` in a buffered/async writer of your own.
:::

### Custom writer with mutation

A `WriterFunc` adapter is sometimes useful for redaction, rate limiting, or sampling. Anything that implements `Write([]byte) (int, error)` works.

```go
type sampledWriter struct {
    w    io.Writer
    keep func() bool
}

func (s *sampledWriter) Write(p []byte) (int, error) {
    if !s.keep() {
        return len(p), nil // pretend we wrote it
    }
    return s.w.Write(p)
}
```

`keep` runs on every emission across every logging goroutine. Implement it with a lock-free counter (`atomic.Uint64` with modulo, or `math/rand/v2`); a shared `sync.Mutex` would serialize the whole dispatch path.

For dispatch-layer concerns (sampling, redaction, level routing), prefer plugins or a sibling transport with a different Level filter. The Writer layer is the wrong place for anything that depends on the entry's structure.

## Concurrency

Each emission calls `Writer.Write(b)` exactly once with a complete newline-terminated entry. Multiple goroutines logging through the same transport call `Write` concurrently, so the writer must be safe for concurrent use.

| Writer | Concurrent-safe? |
|--------|----------------|
| `os.Stdout` / `os.Stderr` / `*os.File` | Yes (each `Write` is one `write(2)` call; short entries don't tear in practice) |
| `*lumberjack.Logger` | Yes (internal mutex) |
| `*bytes.Buffer` | No |
| `net.Conn` (TCP / UDP) | Yes (per Go's `net` docs). TCP writes can interleave at the byte level for entries larger than the kernel send buffer. |
| `io.MultiWriter` | Only if every wrapped writer is safe |

For unsafe writers, wrap with a mutex:

```go
type lockedWriter struct {
    mu sync.Mutex
    w  io.Writer
}

func (l *lockedWriter) Write(p []byte) (int, error) {
    l.mu.Lock()
    defer l.mu.Unlock()
    return l.w.Write(p)
}
```

Don't wrap writers that are already safe (the rows above marked Yes). Adding a second lock around `*os.File` or `*lumberjack.Logger` just serializes the dispatch path with no benefit.

## Wrapper Transports and the `Writer` Field

Wrapper transports ([Zerolog](/transports/zerolog), [Zap](/transports/zap), [logrus](/transports/logrus), [phuslu](/transports/phuslu), [slog](/transports/slog), [charmbracelet/log](/transports/charmlog)) also expose a `Writer` field, but it's a fallback. It's used only when you don't supply a pre-built `Logger`. The right place to configure output for those transports is the underlying logger you pass in:

```go
import (
    "github.com/rs/zerolog"
    "go.loglayer.dev"
    llzero "go.loglayer.dev/transports/zerolog"
)

// zerolog: configure on the *zerolog.Logger you build
z := zerolog.New(rotator).With().Timestamp().Logger()
log := loglayer.New(loglayer.Config{
    Transport: llzero.New(llzero.Config{Logger: &z}),
})
```

Reach for a wrapper's `Writer` field only when you want a default Logger configured by LogLayer.
