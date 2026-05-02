---
title: File (Lumberjack)
description: "One JSON object per log entry written to a rotating file. Backed by lumberjack.v2."
---

# File (Lumberjack)

<ModuleBadges path="transports/lumberjack" />

The `lumberjack` transport writes one JSON object per log entry to a rotating file on disk. Rotation is handled by [lumberjack.v2](https://github.com/natefinch/lumberjack): size-triggered rollover, configurable backup retention, age-based cleanup, and optional gzip compression. The on-disk format matches [`transports/structured`](/transports/structured) exactly.

```sh
go get go.loglayer.dev/transports/lumberjack/v2
```

## Basic Usage

```go
import (
    "go.loglayer.dev/v2"
    "go.loglayer.dev/transports/lumberjack/v2"
)

log := loglayer.New(loglayer.Config{
    Transport: lumberjack.New(lumberjack.Config{
        Filename: "/var/log/myapp/app.log",
    }),
})

log.Info("hello")
// /var/log/myapp/app.log:
// {"level":"info","time":"2026-04-29T12:00:00Z","msg":"hello"}
```

`Filename` is required. Use `lumberjack.Build` instead of `lumberjack.New` if the path is loaded at runtime and you want to handle the missing-config case explicitly:

```go
tr, err := lumberjack.Build(lumberjack.Config{Filename: os.Getenv("LOG_FILE")})
if errors.Is(err, lumberjack.ErrFilenameRequired) {
    // fallback: log to stderr instead
}
```

## Config

```go
type Config struct {
    transport.BaseConfig

    Filename   string                  // required
    MaxSize    int                     // MB; default 100
    MaxBackups int                     // 0 = keep all (subject to MaxAge)
    MaxAge     int                     // days; 0 = no age-based cleanup
    Compress   bool                    // gzip rotated files
    LocalTime  bool                    // backup-filename timestamps in local time (default: UTC)

    MessageField string                // default: "msg"
    DateField    string                // default: "time"
    LevelField   string                // default: "level"
    DateFn       func() string
    LevelFn      func(loglayer.LogLevel) string
    MessageFn    func(loglayer.TransportParams) string

    OnError func(err error)            // called on write/rotate failure;
                                       // default writes to os.Stderr
}
```

## Rotation

Rotation triggers when the active file's size reaches `MaxSize` megabytes. The current file is renamed with a timestamp suffix and a fresh file is opened at `Filename` for subsequent writes.

```go
lumberjack.New(lumberjack.Config{
    Filename:   "/var/log/myapp/app.log",
    MaxSize:    100,  // rotate at 100 MB
    MaxBackups: 7,    // keep the 7 most recent backups
    MaxAge:     30,   // delete backups older than 30 days
    Compress:   true, // gzip rotated files
})
```

After rotation the directory looks like:

```
app.log              ← active
app-2026-04-29T03-15-00.000.log.gz
app-2026-04-28T19-42-11.000.log.gz
...
```

`MaxBackups` and `MaxAge` are independent: a file is kept only if it satisfies *both*. Setting both to zero disables cleanup entirely (rotated files accumulate forever).

::: info Lumberjack creates parent directories
The transport calls into lumberjack, which creates the directory tree for `Filename` on first write. If the parent directory cannot be created (permissions, read-only filesystem), the resulting error surfaces through [`OnError`](#error-handling-onerror).
:::

## Forcing Rotation: `Rotate()`

Call `Rotate()` to force an immediate roll-over even when `MaxSize` hasn't been reached. The active file is renamed to a timestamped backup and a fresh file is opened at `Filename` for subsequent writes.

```go
tr := lumberjack.New(lumberjack.Config{Filename: "/var/log/myapp/app.log"})

sigs := make(chan os.Signal, 1)
signal.Notify(sigs, syscall.SIGHUP)
go func() {
    for range sigs {
        if err := tr.Rotate(); err != nil {
            // surface to a separate logger, metric, etc.
        }
    }
}()
```

## Closing the File

`Close()` releases the underlying file handle. After `Close`, subsequent log calls are silently dropped: lumberjack's natural behavior is to lazy-reopen the file on the next write, which would revive the file from a stray late call. The closed flag suppresses that. `Close` is idempotent.

```go
tr := lumberjack.New(lumberjack.Config{Filename: "/var/log/myapp/app.log"})
defer tr.Close()
```

::: warning Close before the process exits
The transport doesn't buffer entries beyond what the OS does for an open file descriptor, but the OS buffer is per-FD and only fully flushed on `Close`. A process that exits without calling `Close()` may lose the last few writes if the kernel hadn't flushed yet. Wire `Close` into your shutdown path.
:::

## Error Handling: `OnError`

Writes can fail for ordinary reasons: disk full, permission denied, parent directory removed out from under the open file descriptor, lumberjack failing to rotate. The `OnError` callback surfaces those errors so a failing file sink does not silently swallow log entries.

```go
tr := lumberjack.New(lumberjack.Config{
    Filename: "/var/log/myapp/app.log",
    OnError: func(err error) {
        metrics.Increment("log.file.write_failures", 1)
        // Optional: re-log the failure via a different sink so it stays
        // visible if the file sink is the only thing that's broken.
        fallback.WithError(err).Error("file transport write failed")
    },
})
```

If `OnError` is nil, a default writes a one-line message to `os.Stderr`. Replace it with a no-op (`func(error) {}`) only if you have a higher-level monitoring path (a sibling transport, an external log shipper, an `OnTransportPanic` handler) catching the failure already.

::: tip Fan out so a file outage doesn't blind you
Even with `OnError`, the entry that triggered the error is lost. Run a second transport (`pretty`, `structured`, an HTTP shipper) so log lines reach somewhere even when the file sink is down. The `Pretty stdout + JSON file` recipe below is the simplest version.
:::

## Renaming the Standard Fields

The render-tuning fields (`MessageField`, `DateField`, `LevelField`, `DateFn`, `LevelFn`, `MessageFn`) pass through to the structured renderer; see [Structured Transport](/transports/structured#renaming-the-standard-fields) for examples.

## Direct Access: `GetLoggerInstance`

`Transport.GetLoggerInstance()` returns the underlying `*lumberjack.Logger` so callers can pass it to code that expects a `*lumberjack.Logger` directly (a metrics shipper, a custom rotator-aware utility, etc.). For the everyday operations, the transport already exposes `Rotate()` and `Close()`; you don't need to dive into lumberjack for those.

If you keep a reference to the transport, get it directly. The upstream library and this transport share the package name `lumberjack`, so alias the upstream when you import both.

```go
import (
    lj "gopkg.in/natefinch/lumberjack.v2"

    "go.loglayer.dev/v2"
    "go.loglayer.dev/transports/lumberjack/v2"
)

tr := lumberjack.New(lumberjack.Config{
    Filename: "/var/log/myapp/app.log",
})

rotator := tr.GetLoggerInstance().(*lj.Logger)
_ = rotator
```

If you only have the `*loglayer.LogLayer` (e.g. inside a handler that received the logger from elsewhere), look it up by transport ID. This requires assigning a known ID at construction:

```go
import (
    lj "gopkg.in/natefinch/lumberjack.v2"

    "go.loglayer.dev/v2"
    "go.loglayer.dev/v2/transport"
    "go.loglayer.dev/transports/lumberjack/v2"
)

log := loglayer.New(loglayer.Config{
    Transport: lumberjack.New(lumberjack.Config{
        BaseConfig: transport.BaseConfig{ID: "file"}, // required for lookup-by-ID
        Filename:   "/var/log/myapp/app.log",
    }),
})

// ...later, somewhere only the logger is in scope:
rotator := log.GetLoggerInstance("file").(*lj.Logger)
_ = rotator
```

Without `BaseConfig.ID`, the transport gets an auto-generated identifier and `GetLoggerInstance("file")` returns nil.

## Recipes

### Pretty stdout + JSON file at the same time

Render colorized output to the terminal during interactive runs (via the [Pretty](/transports/pretty) transport) and structured JSON to a rotating file in production, both from one logger. Each transport keeps its own filename, format, and level filter.

```go
import (
    "go.loglayer.dev/v2"
    "go.loglayer.dev/transports/lumberjack/v2"
    "go.loglayer.dev/transports/pretty/v2"
)

log := loglayer.New(loglayer.Config{
    Transports: []loglayer.Transport{
        pretty.New(pretty.Config{}),
        lumberjack.New(lumberjack.Config{
            Filename:   "/var/log/myapp/app.log",
            MaxSize:    100,
            MaxBackups: 7,
            Compress:   true,
        }),
    },
})

log.Info("served")
// stdout: 12:00:00.000 INFO served   (colorized)
// /var/log/myapp/app.log: {"level":"info","time":"...","msg":"served"}
```

### Severity-routed files (info.log + error.log)

A common ops pattern: ship everything to `info.log` and ship errors-only to a smaller, longer-retention `error.log`. Each transport gets every entry; the per-transport `Level` filter keeps each file scoped to its own threshold.

```go
import (
    "go.loglayer.dev/v2"
    "go.loglayer.dev/v2/transport"
    "go.loglayer.dev/transports/lumberjack/v2"
)

infoLog := lumberjack.New(lumberjack.Config{
    BaseConfig: transport.BaseConfig{ID: "info-file"},
    Filename:   "/var/log/myapp/info.log",
    MaxSize:    100, MaxBackups: 7, Compress: true,
})
errorLog := lumberjack.New(lumberjack.Config{
    BaseConfig: transport.BaseConfig{
        ID:    "error-file",
        Level: loglayer.LogLevelError, // error and fatal only
    },
    Filename:   "/var/log/myapp/error.log",
    MaxSize:    50, MaxBackups: 30, Compress: true, // longer retention
})

log := loglayer.New(loglayer.Config{
    Transports: []loglayer.Transport{infoLog, errorLog},
})
```

`info.log` ends up with everything at info level and above (including errors). `error.log` contains only the error-and-above subset. To omit info/warn from `info.log` without sending them anywhere, change the logger's own level instead via `loglayer.Config.MinLevel`.

### Daily / time-based rotation

lumberjack rotates on size, not on the clock. For a calendar-aligned roll-over (e.g. a fresh file at 00:00 UTC every day), drive `Rotate()` from a Go ticker.

```go
import (
    "time"
    "go.loglayer.dev/transports/lumberjack/v2"
)

tr := lumberjack.New(lumberjack.Config{
    Filename:   "/var/log/myapp/app.log",
    MaxSize:    1024, // big enough that lumberjack rarely rotates on size
    MaxBackups: 30,
    Compress:   true,
})

go func() {
    for {
        now := time.Now().UTC()
        next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
        time.Sleep(time.Until(next))
        _ = tr.Rotate()
    }
}()
```

Set `MaxSize` large enough that lumberjack's size-trigger doesn't compete with your daily rotation; `MaxBackups` and `MaxAge` still apply to the rotated files the ticker creates.

### Driving rotation from `logrotate(8)`

If your platform already runs `logrotate(8)`, let it own rotation and use the file transport just to write. Wire SIGHUP to `tr.Rotate()` so the external tool can drive a clean roll-over without restarting the process.

`/etc/logrotate.d/myapp`:

```
/var/log/myapp/app.log {
    daily
    rotate 7
    compress
    missingok
    notifempty
    sharedscripts
    postrotate
        kill -HUP $(cat /var/run/myapp.pid) 2>/dev/null || true
    endscript
}
```

Go side: the SIGHUP handler from [Forcing Rotation](#forcing-rotation-rotate). Set `MaxSize` to a value larger than your daily volume so lumberjack doesn't roll over mid-day on top of `logrotate`'s schedule.

::: warning Don't rely on rename-and-truncate
Some `logrotate` configurations use `copytruncate` instead of asking the application to reopen its file. That mode is unsafe with this transport: lumberjack holds the file descriptor open, so writes after the truncate land at large offsets in a sparse file, and the rotated copy ends up with the entries you wrote during the rotation window. Use `postrotate` + SIGHUP + `tr.Rotate()` instead.
:::

## Operational Notes

::: warning No goroutine cleanup on Close
lumberjack starts a background goroutine the first time it has cleanup work to do (driven by `MaxBackups`, `MaxAge`, or `Compress`). That goroutine has no public stop signal, so it outlives `Close()`. Fine for long-lived processes; tests that construct and tear down many transports leak one goroutine per instance.
:::

## Fatal Behavior

<!--@include: ./_partials/fatal-passthrough.md-->

## Alternative: Plug Lumberjack Into Other Transports

The `lumberjack` transport is a turnkey wrapper for JSON-to-rotating-file. For pretty-formatted, console-style, or any other render flavor written to a rotating file, plug a `*lumberjack.Logger` into the renderer's `Writer` field directly. See [Rotating file (lumberjack into any renderer)](/transports/writers#rotating-file-lumberjack-into-any-renderer).
