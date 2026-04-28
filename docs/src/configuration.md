---
title: Configuration
description: Every field on loglayer.Config explained.
---

# Configuration

`loglayer.New` takes a `loglayer.Config`. Every field has a sensible default; only `Transport` (or `Transports`) is required.

<!--@include: ./_partials/constructors.md-->

Most application setup should stick with `New`: misconfiguration of the logger is a programmer error, and panicking at construction time fails loudly rather than letting a misconfigured logger drift into production.

```go
type Config struct {
    Transport          Transport            // single transport (mutually exclusive with Transports)
    Transports         []Transport          // multiple transports (mutually exclusive with Transport)
    Plugins            []Plugin             // plugins to register at construction time
    Groups             map[string]LogGroup  // named routing rules (see Groups)
    ActiveGroups       []string             // restrict routing to these groups (nil/empty = no filter)
    UngroupedRouting   UngroupedRouting     // how to route entries with no group tag
    Prefix             string               // prepended to first string message
    Disabled           bool                 // suppress all output (default: false)
    ErrorSerializer    ErrorSerializer      // customize error rendering
    ErrorFieldName     string               // key for serialized error (default: "err")
    CopyMsgOnOnlyError bool                 // copy err.Error() into the message in ErrorOnly
    FieldsKey          string               // nest fields under this key (default: merged at root)
    MuteFields         bool                 // disable fields in output
    MuteMetadata       bool                 // disable metadata in output
    DisableFatalExit   bool                 // skip os.Exit(1) after a Fatal log
}
```

## Transports

Set exactly one of `Transport` or `Transports`:

```go
// Single transport
loglayer.New(loglayer.Config{
    Transport: structured.New(structured.Config{}),
})

// Multiple: every entry fans out to all of them
loglayer.New(loglayer.Config{
    Transports: []loglayer.Transport{
        console.New(console.Config{}),
        structured.New(structured.Config{Writer: jsonFile}),
    },
})
```

`loglayer.New` panics if neither is set. Setting both panics with `ErrTransportAndTransports` (or `Build` returns it). See [Multiple Transports](/transports/multiple-transports).

## Plugins

Plugins to register at construction time. Equivalent to calling `log.AddPlugin` for each entry after `New`; either form is fine.

```go
import "go.loglayer.dev/plugins/redact"

log := loglayer.New(loglayer.Config{
    Transport: structured.New(structured.Config{}),
    Plugins: []loglayer.Plugin{
        redact.New(redact.Config{Keys: []string{"password", "apiKey"}}),
    },
})
```

Plugin order matters: hooks run in the order plugins were added, and each plugin sees the previous plugin's output. See [Plugins](/plugins/) for the full lifecycle.

Plugin `ID` is optional; LogLayer auto-generates one when you omit it. Supply your own ID when you intend to call `RemovePlugin` / `GetPlugin` later.

## Groups, ActiveGroups, UngroupedRouting

Named routing rules for sending log entries to specific transports based on tags. When `Groups` is nil/empty there is no routing: every transport receives every entry. Once configured, tag entries via `WithGroup` to opt them into a group's routing.

```go
log := loglayer.New(loglayer.Config{
    Transports: []loglayer.Transport{...},
    Groups: map[string]loglayer.LogGroup{
        "database": {Transports: []string{"datadog"}, Level: loglayer.LogLevelError},
    },
    ActiveGroups: loglayer.ActiveGroupsFromEnv("LOGLAYER_GROUPS"), // optional env-driven filter
})
```

See [Groups](/logging-api/groups) for the full reference: per-group level filters, multi-group routing, ungrouped behavior modes, runtime mutators.

## Prefix

A string prepended (with one space) to the first string message of every log call. Useful for tagging a logger as belonging to a subsystem:

```go
log := loglayer.New(loglayer.Config{
    Transport: structured.New(structured.Config{}),
    Prefix:    "[auth]",
})

log.Info("started") // → "msg":"[auth] started"
```

`WithPrefix(prefix)` returns a child logger with the prefix overridden, leaving the parent untouched.

## Disabled

Set to `true` to suppress all log output from construction. Equivalent to calling `log.DisableLogging()` immediately after `New`:

```go
log := loglayer.New(loglayer.Config{
    Transport: structured.New(structured.Config{}),
    Disabled:  true, // every level dropped
})
```

You can flip it at runtime with `log.EnableLogging()` / `log.DisableLogging()`. See [Adjusting Log Levels](/logging-api/adjusting-log-levels).

## ErrorSerializer

A function that converts `error` to a `map[string]any`. The default returns `{"message": err.Error()}`. Override to capture stack traces, error chains, or library-specific fields. We recommend [`github.com/rotisserie/eris`](https://github.com/rotisserie/eris), its `ToJSON` function plugs in directly:

```go
import "github.com/rotisserie/eris"

loglayer.New(loglayer.Config{
    Transport: structured.New(structured.Config{}),
    ErrorSerializer: func(err error) map[string]any {
        return eris.ToJSON(err, true) // true = include stack trace
    },
})
```

See [Error Handling](/logging-api/error-handling) for more options including writing your own serializer.

## ErrorFieldName

The key used for the serialized error inside the assembled data object. Defaults to `"err"`:

```go
loglayer.New(loglayer.Config{
    Transport:      structured.New(structured.Config{}),
    ErrorFieldName: "error",
})
```

## CopyMsgOnOnlyError

When `true`, `log.ErrorOnly(err)` also uses `err.Error()` as the log message. Defaults to `false`, `ErrorOnly` produces an entry with no message and just the error in `data.err`. Per-call override is available via `ErrorOnlyOpts.CopyMsg`.

## FieldsKey

By default, fields are merged at the root of the data object. Set this to nest them under a single key:

```go
loglayer.New(loglayer.Config{
    Transport: structured.New(structured.Config{}),
    FieldsKey: "fields",
})

log.WithFields(loglayer.Fields{"requestId": "abc"}).Info("ok")
// {"msg":"ok","fields":{"requestId":"abc"}}
```

See [Fields](/logging-api/fields).

## DisableFatalExit

When `true`, the core skips the `os.Exit(1)` call that normally follows a fatal-level dispatch. Defaults to `false` (fatal exits, matching Go convention).

```go
log := loglayer.New(loglayer.Config{
    Transport:        structured.New(structured.Config{}),
    DisableFatalExit: true,
})
log.Fatal("logged, but process keeps running")
```

`loglayer.NewMock()` enables this automatically. See [Mocking](/logging-api/mocking) and [Fatal Exits the Process](/logging-api/basic-logging#fatal-exits-the-process).

## MuteFields / MuteMetadata

Boolean flags that suppress fields or metadata from output. The data is still tracked on the logger, only the emit step skips it. Useful in development to cut log noise without removing the calls.

```go
loglayer.New(loglayer.Config{
    Transport:    structured.New(structured.Config{}),
    MuteMetadata: true, // metadata still attached, just not emitted
})
```

You can flip these at runtime with `log.MuteFields()`, `log.UnmuteFields()`, `log.MuteMetadata()`, `log.UnmuteMetadata()`.

## AddSource / SourceFieldName

`AddSource: true` captures the call site (file, line, function) of every log emission and includes it in the assembled `Data` under `SourceFieldName` (default `"source"`). Off by default; opt in for production-debuggable output.

```go
log := loglayer.New(loglayer.Config{
    Transport: structured.New(structured.Config{}),
    AddSource: true,
})

log.Info("served")
// {"level":"info","time":"...","msg":"served","source":{"function":"main.handler","file":"/app/main.go","line":42}}
```

The captured `Source` value is a `*loglayer.Source` with `Function`, `File`, `Line`. JSON tags match the [`log/slog`](https://pkg.go.dev/log/slog) source convention so structured output is interchangeable with standard slog setups. The struct also implements `fmt.Stringer` (compact `func file:line` rendering for console / pretty transports) and `slog.LogValuer` (nested group when forwarded to a slog handler).

Override the output key with `SourceFieldName: "caller"` (or any string) when matching an existing log schema.

```go
loglayer.New(loglayer.Config{
    Transport:       structured.New(structured.Config{}),
    AddSource:       true,
    SourceFieldName: "caller",
})
```

Cost: about **620 ns and 5 extra allocations per emission** on amd64 (`BenchmarkLoglayer_SimpleMessage` goes from ~40 ns / 1 alloc to ~660 ns / 6 allocs). The dominant terms are `runtime.Caller`'s frame walk, `runtime.FuncForPC().Name()` materializing the function-name string, and the heap-allocated `*Source`. Paid only when `AddSource` is true; the dispatch path is untouched otherwise. If per-emission cost matters more than caller info, leave it off and rely on transport-level rendering plus inline metadata.

::: tip Adapters can supply Source explicitly
If you're calling `log.Raw(...)` from an adapter that already has a program counter (the [slog handler](/integrations/sloghandler) extracts it from `slog.Record.PC`), pass `Source: loglayer.SourceFromPC(pc)` on the `RawLogEntry` and skip runtime capture. The slog handler does this automatically.
:::

## Transport BaseConfig

Each transport accepts a `transport.BaseConfig` for transport-level concerns:

```go
type BaseConfig struct {
    ID       string            // unique identifier (required for AddTransport / RemoveTransport)
    Disabled bool              // suppress this transport (default: false)
    Level    loglayer.LogLevel // minimum level this transport will process
}
```

Transport-level `Level` filtering happens *in addition to* the logger's level filtering. See [Transport Configuration](/transports/configuration) and individual transport pages.
