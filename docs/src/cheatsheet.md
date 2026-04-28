---
title: Cheat Sheet
description: One-page quick reference of the LogLayer for Go API.
---

# Cheat Sheet

## At a Glance

```go
import (
    "go.loglayer.dev"
    "go.loglayer.dev/transports/structured"
)

log := loglayer.New(loglayer.Config{Transport: structured.New(structured.Config{})})

// The fluent chain: persistent fields → per-call metadata + error → terminal level method.
log.
    WithFields(loglayer.Fields{"requestId": "abc"}). // returns a new logger; assign on a real call site
    WithMetadata(loglayer.Metadata{"durationMs": 23}).
    WithError(err).
    Error("request failed")
```

That's the 80% case. The rest of this page is a one-line lookup for everything else.

## Method Conventions

LogLayer uses two distinct method patterns. Knowing which is which avoids one of the few footguns in the API.

| Prefix | Pattern | Example |
|---|---|---|
| `With*` | Returns a **new logger or builder**. The receiver is unchanged; **assign the return value** or your change is lost. | `log = log.WithFields(...)` |
| `Mute`, `Unmute`, `Set`, `Enable`, `Disable`, `Add`, `Remove` | Mutates the receiver in place. Returns `*LogLayer` for chaining; the return value is the same instance. | `log.MuteFields()` |

`Child()` is the one exception to the prefix rule: it returns a new logger (conventional name in Go logging libraries; mirrors zerolog/slog). Treat it the same as `With*` and assign the result.

```go
log = log.WithFields(loglayer.Fields{"req": "abc"}) // ✅ assigned
log.WithFields(loglayer.Fields{"req": "abc"})       // ❌ result discarded; emits without req

log.MuteFields()                                    // ✅ in-place mutation; no assignment needed
```

## Construction

<!--@include: ./_partials/constructors.md-->

## Log Levels

```go
log.Debug("...")
log.Info("...")
log.Warn("...")
log.Error("...")
log.Fatal("...") // calls os.Exit(1) by default; set Config.DisableFatalExit to skip
```

Each method takes `...any`, joined with a space.

For `fmt.Sprintf`-style format strings, register the optional [`fmtlog`](https://pkg.go.dev/go.loglayer.dev/fmtlog) plugin:

```go
import "go.loglayer.dev/fmtlog"

log.AddPlugin(fmtlog.New())

log.Info("user %d signed in", userID)
log.WithMetadata(loglayer.Metadata{"reqId": reqID}).
    Error("request %s failed: %v", reqID, err)
```

Without the plugin, multi-arg messages are space-joined.

## Metadata

```go
// Struct (preferred when the shape is fixed; cheaper, type-checked)
type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}
log.WithMetadata(User{ID: 7, Name: "Alice"}).Info("user")

// Map (use when keys vary at runtime; loglayer.Metadata is map[string]any)
log.WithMetadata(loglayer.Metadata{"id": 1}).Info("ok")

// No message, log just the metadata
log.MetadataOnly(loglayer.Metadata{"status": "healthy"})
log.MetadataOnly(loglayer.Metadata{"status": "warn"}, loglayer.MetadataOnlyOpts{LogLevel: loglayer.LogLevelWarn})
```

`MetadataOnly` is a **terminal call**, not a builder. It dispatches the entry immediately. You cannot chain `WithError` or `WithCtx` onto it; for that, use `log.WithMetadata(...).Info(...)` etc.

## Errors

```go
log.WithError(err).Error("failed")
log.ErrorOnly(err)
log.ErrorOnly(err, loglayer.ErrorOnlyOpts{LogLevel: loglayer.LogLevelFatal})
```

`ErrorOnly` is also a **terminal call** like `MetadataOnly`. Use `log.WithError(err).Error("...")` if you want to attach an error and a message together.

## Fields (persistent)

```go
// WithFields and WithoutFields return a NEW logger; assign the result.
log = log.WithFields(loglayer.Fields{"requestId": "123"})

// Read it back
fields := log.GetFields()

// Remove specific keys (returns new logger)
log = log.WithoutFields("requestId")

// Remove all (returns new logger)
log = log.WithoutFields()

// Mute / unmute output without losing the data (mutates in place)
log.MuteFields().UnmuteFields()
```

### Short aliases: `loglayer.F` and `loglayer.M`

Both are type aliases for `loglayer.Fields` and `loglayer.Metadata` (interchangeable at every call site). Use them to keep dense call sites readable:

```go
log.WithFields(loglayer.F{"requestId": "abc"}).
    WithMetadata(loglayer.M{"durationMs": 23}).
    Info("served")
```

## Go Context

```go
import "context"

// Bind once, every emission carries it (per-request handlers).
log = log.WithCtx(ctx)
log.Info("served")
log.Warn("retrying")

// Or per-call only (override):
log.WithCtx(otherCtx).Info("override for this entry")
```

Surfaced to transports via `TransportParams.Ctx` and to plugin dispatch hooks via `params.Ctx`. The `loghttp` middleware binds `r.Context()` automatically. See [Go Context](/logging-api/go-context).

## Logger in a Go Context (zerolog-style)

```go
// Middleware: store request-scoped logger in ctx
ctx = loglayer.NewContext(ctx, reqLog)

// Handler: pull it back out
log := loglayer.FromContext(ctx)         // nil if not attached
log := loglayer.MustFromContext(ctx)     // panics if not attached
```

## Builder vs Logger chain

The "At a Glance" example shows the typical chain. Two things to know:

- `WithFields`, `WithCtx`, `WithGroup` (when called on `*LogLayer`) and `WithPrefix`, `Child`, `WithoutFields` all return a **new logger**. Assign them: `log = log.WithCtx(ctx)`.
- `WithMetadata`, `WithError`, and the same `WithCtx` / `WithGroup` when called on a `*LogBuilder` return a **`*LogBuilder`**: single-use, terminated by a level method (`Info`, `Warn`, ...). Don't assign the builder.

## Child Loggers

```go
child := log.Child()                       // copy of fields + level state
prefixed := log.WithPrefix("[auth]")       // child with a prefix prepended
```

Mutations on the child do not affect the parent.

## Level Control

```go
log.SetLevel(loglayer.LogLevelWarn)        // enable warn and above
log.EnableLevel(loglayer.LogLevelDebug)    // turn one level on
log.DisableLevel(loglayer.LogLevelDebug)   // turn one level off
log.IsLevelEnabled(loglayer.LogLevelInfo)  // check
log.DisableLogging()                       // master kill switch
log.EnableLogging()
```

## Transport Management

```go
log.AddTransport(t)                    // append (replaces if same ID)
log.RemoveTransport("id")              // returns true if removed
log.SetTransports(t1, t2)              // replace all
log.GetLoggerInstance("id")            // underlying logger from a transport
```

## Groups

```go
// Define routing rules at construction
log := loglayer.New(loglayer.Config{
    Transports: []loglayer.Transport{...},
    Groups: map[string]loglayer.LogGroup{
        "database": {Transports: []string{"datadog"}, Level: loglayer.LogLevelError},
        "auth":     {Transports: []string{"sentry"}, Level: loglayer.LogLevelWarn},
    },
})

// Tag a single entry
log.WithGroup("database").Error("connection lost")
log.WithGroup("database", "auth").Error("auth db failure")  // union of both groups' transports

// Persistent tagging via a child logger
dbLog := log.WithGroup("database")
dbLog.Error("pool exhausted")  // every log routes via 'database'

// Runtime management
log.AddGroup("inbox", loglayer.LogGroup{Transports: []string{"datadog"}})
log.RemoveGroup("inbox")            // returns bool
log.EnableGroup("database")
log.DisableGroup("database")
log.SetGroupLevel("database", loglayer.LogLevelDebug)
log.SetActiveGroups("database")     // restrict to these groups
log.ClearActiveGroups()             // remove the filter
log.GetGroups()                     // shallow copy of current config

// Drive the active filter from an env var
loglayer.ActiveGroupsFromEnv("LOGLAYER_GROUPS") // returns []string for Config.ActiveGroups
```

See [Groups](/logging-api/groups) for the eight-rule routing precedence (defined-but-disabled vs undefined groups, per-group level filtering, ungrouped fallback) and a worked multi-service example.

## Plugins

```go
import "go.loglayer.dev/plugins/redact"

// Inline single-hook plugin via an adapter constructor
log.AddPlugin(loglayer.NewDataHook("tag", func(p loglayer.BeforeDataOutParams) loglayer.Data {
    return loglayer.Data{"service": "checkout"}
}))
// Other adapters: NewFieldsHook, NewMetadataHook, NewMessageHook, NewLevelHook, NewSendGate, NewPlugin (no hooks).
// Multi-hook plugins: declare a type implementing Plugin + the hook interfaces.

// Redact plugin (key + regex matching, walks structs/maps/slices)
log.AddPlugin(redact.New(redact.Config{
    Keys:     []string{"password", "apiKey"},
    Patterns: []*regexp.Regexp{regexp.MustCompile(`^\d{16}$`)}, // credit-card-shaped
}))

// Management
log.RemovePlugin("id")                  // returns true if removed
log.GetPlugin("id")                     // (Plugin, bool)
log.PluginCount()                       // int
```

Six lifecycle hooks (any subset, nil fields skipped): `OnFieldsCalled`, `OnMetadataCalled`, `OnBeforeDataOut`, `OnBeforeMessageOut`, `TransformLogLevel`, `ShouldSend`. See [Plugins overview](/plugins/) for hook semantics, [Creating Plugins](/plugins/creating-plugins) for the authoring tutorial (covers panic recovery, testing, and the `RecoveredPanicError` type), or the [`examples/custom-plugin`](https://github.com/loglayer/loglayer-go/tree/main/examples/custom-plugin) directory for a runnable from-scratch demo.

## Raw

```go
log.Raw(loglayer.RawLogEntry{
    LogLevel: loglayer.LogLevelInfo,
    Messages: []any{"already assembled"},
    Metadata: loglayer.Metadata{"k": "v"},
    Err:      err,
    Fields:   loglayer.Fields{"override": "ctx"}, // optional override
    Ctx:      ctx,                                 // optional Go context
})
```

## Levels

```go
loglayer.LogLevelDebug  // 10
loglayer.LogLevelInfo   // 20
loglayer.LogLevelWarn   // 30
loglayer.LogLevelError  // 40
loglayer.LogLevelFatal  // 50
```

## Source / Caller info

```go
log := loglayer.New(loglayer.Config{
    Transport: structured.New(structured.Config{}),
    AddSource: true,                 // capture file/line/function
    SourceFieldName: "source",       // default; override to "caller" etc.
})

log.Info("served")
// {"level":"info","time":"...","msg":"served","source":{"function":"main.handler","file":"/app/main.go","line":42}}
```

Off by default. Costs ~100 ns / one runtime.Caller per emission when on. The slog Handler forwards `slog.Record.PC` automatically (no capture cost on the slog path).

## slog Interop

```go
import "go.loglayer.dev/integrations/sloghandler"

// Make every slog.Info(...) flow through your loglayer pipeline (plugins,
// fan-out, groups, level state). slog.With / WithAttrs become persistent
// fields; WithGroup nests; LogValuer is resolved.
slog.SetDefault(slog.New(sloghandler.New(log)))
slog.Info("served", "userId", 42)
```

The opposite direction (loglayer emitting through a `*slog.Logger` backend) is the [slog Transport](/transports/slog).
