---
title: Structured Transport
description: One JSON object per log entry. The default for production logging.
---

# Structured Transport

<ModuleBadges path="transports/structured" />

The `structured` transport always writes one JSON object per log entry. By default each entry has `level`, `time`, and `msg` fields, with fields merged at the root.

```sh
go get go.loglayer.dev/transports/structured/v3
```

## Basic Usage

```go
import (
    "go.loglayer.dev/v3"
    "go.loglayer.dev/transports/structured/v3"
)

log := loglayer.New(loglayer.Config{
    Transport: structured.New(structured.Config{}),
    FieldsKey: "context",
})

log.Info("hello")
// {"level":"info","time":"2026-04-25T12:00:00Z","msg":"hello"}

log.WithFields(loglayer.Fields{"requestId": "abc"}).
    WithMetadata(loglayer.Metadata{"user": "alice"}).
    Info("served")
// {"level":"info","time":"...","msg":"served","context":{"requestId":"abc"},"metadata":{"user":"alice"}}
```

## Config

```go
type Config struct {
    transport.BaseConfig

    MessageField string                              // default: "msg"
    DateField    string                              // default: "time"
    LevelField   string                              // default: "level"

    DateFn    func() string                          // override timestamp generation
    LevelFn   func(loglayer.LogLevel) string         // override level rendering
    MessageFn func(loglayer.TransportParams) string  // format the message text

    Writer io.Writer                                  // default: os.Stdout
}
```

## Renaming the Standard Fields

```go
structured.New(structured.Config{
    MessageField: "message",
    DateField:    "timestamp",
    LevelField:   "severity",
})

log.Info("renamed")
// {"severity":"info","timestamp":"...","message":"renamed"}
```

## Custom Timestamp / Level

```go
structured.New(structured.Config{
    DateFn:  func() string { return strconv.FormatInt(time.Now().Unix(), 10) },
    LevelFn: func(l loglayer.LogLevel) string { return strings.ToUpper(l.String()) },
})

log.Warn("loud")
// {"level":"WARN","time":"1714060800","msg":"loud"}
```

## Writing to a File or Buffer

The `Writer` field accepts any `io.Writer`. See [Writers](/transports/writers) for recipes covering files, rotating files, `bytes.Buffer`, `io.MultiWriter`, and network sockets, plus a concurrency-safety table.

## Struct Metadata

Metadata (map or struct) nests under [`MetadataFieldName`](/configuration#metadatafieldname), which the v3 core defaults to `"metadata"`:

```go
type User struct {
    ID    int    `json:"id"`
    Email string `json:"email"`
}

log.WithMetadata(User{ID: 7, Email: "alice@example.com"}).Info("user")
// {"level":"info","time":"...","msg":"user","metadata":{"id":7,"email":"alice@example.com"}}

log.WithMetadata(loglayer.Metadata{"total": 3}).Info("cart")
// {"level":"info","time":"...","msg":"cart","metadata":{"total":3}}
```

A struct is JSON-marshaled + unmarshaled into a `map[string]any` before encoding, so `json:` tags apply. The core LogLayer does not touch the value, see [Metadata](/logging-api/metadata). With `Config.FlattenMetadata: true`, the v2 shape returns: struct fields and map entries merge at the root instead of nesting.

## Errors

Errors are serialized via the logger's `ErrorSerializer` (default `{"message": err.Error()}`) and placed under `ErrorFieldName` (default `err`):

```go
log.WithError(err).Error("failed")
// {"level":"error","time":"...","msg":"failed","err":{"message":"connection refused"}}
```

See [Error Handling](/logging-api/error-handling).

## When Marshaling Fails

If `json.Marshal` returns an error (typically because metadata contains an unsupported type like a channel), the transport writes a fallback JSON error object instead of dropping the entry silently:

```json
{"level":"error","msg":"loglayer: failed to marshal log entry","error":"json: unsupported type: chan int"}
```

Catch these in monitoring, they indicate a code-side bug, not a runtime issue.

## Fatal Behavior

<!--@include: ./_partials/fatal-passthrough.md-->

