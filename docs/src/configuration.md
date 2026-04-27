---
title: Configuration
description: Every field on loglayer.Config explained.
---

# Configuration

`loglayer.New` takes a `loglayer.Config`. Every field has a sensible default; only `Transport` (or `Transports`) is required. If neither is set, `New` panics with `loglayer.ErrNoTransport`.

For callers that prefer explicit error handling on misconfiguration, use `loglayer.Build(Config) (*loglayer.LogLayer, error)` instead. It returns `ErrNoTransport` rather than panicking.

```go
log, err := loglayer.Build(loglayer.Config{
    Transport: structured.New(structured.Config{}),
})
if err != nil {
    return err
}
```

Most application setup should stick with `New`: misconfiguration of the logger is a programmer error, and panicking at construction time fails loudly rather than letting a misconfigured logger drift into production.

```go
type Config struct {
    Transport          Transport       // single transport
    Transports         []Transport     // multiple transports
    Prefix             string          // prepended to first string message
    Enabled            *bool           // master on/off (default: true)
    ErrorSerializer    ErrorSerializer // customize error rendering
    ErrorFieldName     string          // key for serialized error (default: "err")
    CopyMsgOnOnlyError bool            // copy err.Error() into the message in ErrorOnly
    FieldsKey          string          // nest fields under this key (default: merged at root)
    MuteFields         bool            // disable fields in output
    MuteMetadata       bool            // disable metadata in output
    DisableFatalExit   bool            // skip os.Exit(1) after a Fatal log
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

`loglayer.New` panics if neither is set. See [Multiple Transports](/transports/multiple-transports).

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

## Enabled

A pointer to a bool so you can distinguish "default true" from "explicitly false":

```go
disabled := false
log := loglayer.New(loglayer.Config{
    Transport: structured.New(structured.Config{}),
    Enabled:   &disabled, // every level dropped
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

## Transport BaseConfig

Each transport accepts a `transport.BaseConfig` for transport-level concerns:

```go
type BaseConfig struct {
    ID      string          // unique identifier (required for AddTransport / RemoveTransport)
    Enabled *bool           // transport-level on/off
    Level   loglayer.LogLevel // minimum level this transport will process
}
```

Transport-level `Level` filtering happens *in addition to* the logger's level filtering. See [Transport Management](/logging-api/transport-management) and individual transport pages.
