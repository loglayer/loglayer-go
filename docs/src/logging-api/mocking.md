---
title: Mocking
description: Replace LogLayer with a no-op or capturing mock when writing tests.
---

# Mocking

When writing tests for code that takes a `*loglayer.LogLayer`, you usually want one of three things:

1. **Silence the logger** so test output stays clean and behaviors aren't accidentally tied to log writes.
2. **Assert on what was logged**: verify your code emits the right entry under the right conditions.
3. **Use the real logger but quiet it**: keep the production wiring, just suppress output.

LogLayer ships a primitive for each.

## 1. Silent Mock: `loglayer.NewMock()`

Use this when logs aren't part of what you're testing. It's a drop-in `*loglayer.LogLayer` backed by a discard transport. Every call is accepted but produces no output.

```go
import "go.loglayer.dev/loglayer"

func TestSomething(t *testing.T) {
    log := loglayer.NewMock()

    result, err := DoWork(log, "input")
    if err != nil {
        t.Fatal(err)
    }
    if result != "expected" {
        t.Errorf("got %q", result)
    }
}
```

`NewMock()` returns the same concrete `*loglayer.LogLayer` type a real construction returns, so it satisfies any signature that takes `*loglayer.LogLayer` (no interface needed). All instance state still works (fields, level filtering, child loggers, prefixes); only the emit step is a no-op.

It also sets [`DisableFatalExit: true`](/configuration#disablefatalexit) so test code that exercises fatal paths doesn't crash the test runner.

```go
log := loglayer.NewMock()
log.WithFields(loglayer.Fields{"requestId": "abc"})
log.SetLevel(loglayer.LogLevelWarn)

log.Info("dropped: below threshold AND silent")
log.Warn("silent: emit step does nothing")

log.GetFields()                            // {"requestId": "abc"}
log.IsLevelEnabled(loglayer.LogLevelInfo)   // false
```

This is the right default for unit tests of business logic.

## 2. Capturing Mock: `transports/testing`

Use this when the test's purpose is to verify *what* was logged. The `transports/testing` package provides a transport that captures every entry into an in-memory library, exposed as typed `LogLine` values.

```go
import (
    "go.loglayer.dev/loglayer"
    "go.loglayer.dev/loglayer/transport"
    lltest "go.loglayer.dev/loglayer/transports/testing"
)

func TestRequestLogging(t *testing.T) {
    lib := &lltest.TestLoggingLibrary{}
    log := loglayer.New(loglayer.Config{
        Transport: lltest.New(lltest.Config{
            BaseConfig: transport.BaseConfig{ID: "test"},
            Library:    lib,
        }),
    })

    handleRequest(log, "abc-123")

    line := lib.PopLine()
    if line == nil {
        t.Fatal("expected a log entry")
    }
    if line.Level != loglayer.LogLevelInfo {
        t.Errorf("level = %s, want info", line.Level)
    }
    if line.Data["requestId"] != "abc-123" {
        t.Errorf("requestId not in fields data: %v", line.Data)
    }
    meta, _ := line.Metadata.(loglayer.Metadata)
    if meta["status"] != 200 {
        t.Errorf("status not in metadata: %v", line.Metadata)
    }
}
```

### LogLine shape

Every captured entry is a `lltest.LogLine` with typed fields. No parsing flat arg lists or string scraping.

```go
type LogLine struct {
    Level    loglayer.LogLevel
    Messages []any
    Data     loglayer.Data // assembled fields + error map (nil when HasData is false)
    HasData  bool
    Metadata any           // raw value passed to WithMetadata
}
```

### Library API

| Method            | Purpose                                                  |
|-------------------|----------------------------------------------------------|
| `Lines()`         | Snapshot copy of all captured lines                      |
| `GetLastLine()`   | Most recent line (does not remove); nil if empty         |
| `PopLine()`       | Most recent line (removes it); nil if empty              |
| `ClearLines()`    | Drop all captured lines                                  |
| `Len()`           | Number of captured lines                                 |

All methods are safe for concurrent use.

### Asserting on struct metadata

Because `WithMetadata(any)` passes structs through untouched, you can assert on the original type:

```go
type orderEvent struct {
    OrderID string
    Total   int
}

log.WithMetadata(orderEvent{OrderID: "o-1", Total: 42}).Info("order placed")

line := lib.PopLine()
event, ok := line.Metadata.(orderEvent)
if !ok {
    t.Fatalf("expected orderEvent, got %T", line.Metadata)
}
if event.OrderID != "o-1" {
    t.Errorf("OrderID: %s", event.OrderID)
}
```

See the [transports/testing](/transports/testing) page for the full transport reference.

## 3. Quiet the Real Logger: `DisableLogging()`

When you want the production wiring intact (real transports, real config) but no output during a particular test:

```go
log := buildProductionLogger() // your normal construction
log.DisableLogging()           // master kill switch: no entries emitted
```

This is rarer than the first two patterns but useful for integration tests that exercise startup/shutdown paths where the log call sites matter (won't panic, won't deadlock) but you don't want the noise.

`EnableLogging()` restores the previous per-level state. See [Adjusting Log Levels](/logging-api/adjusting-log-levels).

## Choosing a Pattern

| Want to…                                                    | Use                          |
|-------------------------------------------------------------|------------------------------|
| Test business logic; ignore log output                      | `loglayer.NewMock()`          |
| Verify a specific log was emitted with the right fields     | `transports/testing`          |
| Run real transports but silence them for one test           | `log.DisableLogging()`        |

## Why no `Logger` interface?

Go's idiomatic pattern is **the consumer defines the interface**. Your application code declares the methods it needs:

```go
type RequestLogger interface {
    WithFields(loglayer.Fields) *loglayer.LogLayer
    Info(...any)
    WithError(error) *loglayer.LogBuilder
}
```

Both real `*loglayer.LogLayer` and the mock from `NewMock()` implicitly satisfy that. Shipping a `loglayer.Logger` interface would push a one-size-fits-all shape on every consumer (over-broad for most call sites and too narrow for some). Keep the concrete type, swap with `NewMock()` in tests.
