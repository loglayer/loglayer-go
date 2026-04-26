---
title: Raw Logging
description: Bypass the builder and dispatch a fully-specified log entry.
---

# Raw Logging

`Raw(entry)` dispatches a `RawLogEntry` straight to the configured transports, skipping the fluent builder. Level filtering, prefix application, and transport fan-out still happen, but you control every field directly.

```go
log.Raw(loglayer.RawLogEntry{
    LogLevel: loglayer.LogLevelInfo,
    Messages: []any{"event"},
    Metadata: loglayer.Metadata{"k": "v"},
    Err:      err,
})
```

`RawLogEntry` is:

```go
type RawLogEntry struct {
    LogLevel LogLevel
    Messages []any
    Metadata any
    Err      error
    Fields   Fields // optional override
}
```

## Overriding Fields

If `Fields` is non-nil it **replaces** the logger's fields for this one entry. Use this when you have a fully-formed snapshot you want to ship without merging:

```go
log.WithFields(loglayer.Fields{"persistent": "value"})

log.Raw(loglayer.RawLogEntry{
    LogLevel: loglayer.LogLevelInfo,
    Messages: []any{"snapshot"},
    Fields:   loglayer.Fields{"override": "yes"}, // logger fields not included
})
```

If `Fields` is nil, the logger's current fields are used.

## When to Use Raw

- **Replaying logs** from a queue, file, or other LogLayer instance: you've already got the full entry; you just want to dispatch it.
- **Bridging from another logging system**: when an upstream framework hands you `(level, msg, fields, err)` you can construct the entry directly without the builder dance.
- **Testing transports**: feed transport-shaped values without going through the builder.

For ordinary application logging, prefer the builder API, it's harder to misuse.
