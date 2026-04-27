---
title: Error Handling
description: Attach errors to logs with WithError and ErrorOnly, and customize how they serialize.
---

# Error Handling

Errors get their own first-class slot on every log entry. Attach one with `WithError`, or log only an error with `ErrorOnly`.

## WithError

```go
log.WithError(err).Error("operation failed")
```

By default the error is serialized as `{"message": err.Error()}` and placed under the `err` field of the data object:

```json
{
  "msg": "operation failed",
  "err": { "message": "connection refused" }
}
```

`WithError` can be chained with `WithMetadata` and a level method:

```go
log.WithMetadata(loglayer.Metadata{"host": "db1"}).
    WithError(err).
    Error("failed to connect")
```

The error is associated with a single log entry; calling `WithError` on a builder doesn't persist to future logs.

## ErrorOnly

When you want to log just an error, with no companion message:

```go
log.ErrorOnly(err)
```

Default level is `Error`. To use a different level:

```go
log.ErrorOnly(err, loglayer.ErrorOnlyOpts{LogLevel: loglayer.LogLevelFatal})
```

To use the error's text as the message body, set `CopyMsgOnOnlyError: true` on the config (or override per-call):

```go
log.ErrorOnly(err, loglayer.ErrorOnlyOpts{CopyMsg: loglayer.CopyMsgEnabled})
log.ErrorOnly(err, loglayer.ErrorOnlyOpts{CopyMsg: loglayer.CopyMsgDisabled}) // explicit opt-out
```

The zero value (`CopyMsgDefault`) keeps the config setting; use `CopyMsgEnabled` or `CopyMsgDisabled` to override per call.

## Customizing Error Serialization

The default serializer captures only `err.Error()`. To capture stack traces, error chains, or library-specific fields, set an `ErrorSerializer`. The serializer is called once per `WithError` / `ErrorOnly` invocation, only when an error is actually present.

### Recommended: `eris` for stack traces and error chains

[`github.com/rotisserie/eris`](https://github.com/rotisserie/eris) is purpose-built for this: its `ToJSON` function returns `map[string]any`, which is exactly the shape `ErrorSerializer` expects.

```sh
go get github.com/rotisserie/eris
```

```go
import "github.com/rotisserie/eris"

log := loglayer.New(loglayer.Config{
    Transport: structured.New(structured.Config{}),
    ErrorSerializer: func(err error) map[string]any {
        return eris.ToJSON(err, true) // true = include stack trace
    },
})

err := eris.New("connection refused")
log.WithError(err).Error("db query failed")
```

```json
{
  "msg": "db query failed",
  "err": {
    "root": {
      "message": "connection refused",
      "stack": [
        "main.queryDB:/app/db.go:42",
        "main.main:/app/main.go:12"
      ]
    }
  }
}
```

`eris` also supports wrapping (`eris.Wrap(err, "context")`), which renders as a chain of root + wrap entries: handy for tracing how an error propagated.

### Rolling your own

If you have specific shaping needs (different field names, redaction, library-specific fields), write the serializer yourself:

```go
loglayer.New(loglayer.Config{
    Transport: structured.New(structured.Config{}),
    ErrorSerializer: func(err error) map[string]any {
        return map[string]any{
            "message": err.Error(),
            "type":    fmt.Sprintf("%T", err),
            "trace":   stackTrace(err),
        }
    },
})
```

### Working with errors.Is / errors.As

The serializer receives the original `error`, so you can branch on its type:

```go
ErrorSerializer: func(err error) map[string]any {
    out := map[string]any{"message": err.Error()}

    var pgErr *pgconn.PgError
    if errors.As(err, &pgErr) {
        out["pg_code"] = pgErr.Code
        out["pg_constraint"] = pgErr.ConstraintName
    }

    return out
}
```

## Renaming the Error Field

Change the key the serialized error is placed under via `ErrorFieldName`:

```go
loglayer.New(loglayer.Config{
    Transport:      structured.New(structured.Config{}),
    ErrorFieldName: "error",
})

log.WithError(err).Error("failed")
```

```json
{
  "msg": "failed",
  "error": { "message": "..." }
}
```

The default is `"err"`.

## Combining with Fields and Metadata

Errors compose with fields and metadata:

```go
log.WithFields(loglayer.Fields{"requestId": "abc"})

log.WithMetadata(loglayer.Metadata{"retry_count": 3}).
    WithError(err).
    Error("retry exhausted")
```

```json
{
  "msg": "retry exhausted",
  "requestId": "abc",
  "retry_count": 3,
  "err": { "message": "..." }
}
```

## Fatal Exits By Default

`log.WithError(err).Fatal(...)` writes the entry then calls `os.Exit(1)`. See [Fatal Exits the Process](/logging-api/basic-logging#fatal-exits-the-process) for opt-out via `DisableFatalExit`.
