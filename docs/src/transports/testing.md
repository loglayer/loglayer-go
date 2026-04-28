---
title: Testing Transport
description: Capture log entries in memory for assertions in tests.
---

# Testing Transport

[![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/testing.svg)](https://pkg.go.dev/go.loglayer.dev/transports/testing) [![Version](https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=v*&label=go.loglayer.dev)](https://github.com/loglayer/loglayer-go/releases) [![Source](https://img.shields.io/badge/source-github-181717?logo=github)](https://github.com/loglayer/loglayer-go/tree/main/transports/testing) [![Changelog](https://img.shields.io/badge/changelog-md-blue)](https://github.com/loglayer/loglayer-go/blob/main/CHANGELOG.md)

The `transports/testing` package is the transport you want in test code: it captures every log entry into a mutex-safe in-memory library, exposing `Messages`, `Data`, and `Metadata` as typed fields on each captured `LogLine`.

For a usage walkthrough see the [Mocking](/logging-api/mocking) page in the logging API section. This page covers the package surface.

```sh
go get go.loglayer.dev/transports/testing
```

```go
import lltest "go.loglayer.dev/transports/testing"
```

(The package name is `testing`, which collides with the standard `testing` package, most users alias the import as `lltest`.)

## Setup

```go
lib := &lltest.TestLoggingLibrary{}
trans := lltest.New(lltest.Config{
    BaseConfig: transport.BaseConfig{ID: "test"},
    Library:    lib,
})

log := loglayer.New(loglayer.Config{Transport: trans})
```

If `Library` is nil, the transport allocates one and exposes it as `trans.Library`.

## Config

```go
type Config struct {
    transport.BaseConfig
    Library *TestLoggingLibrary // nil → auto-allocate
}
```

## LogLine Shape

```go
type LogLine struct {
    Level    loglayer.LogLevel
    Messages []any
    Data     loglayer.Data // assembled fields + error map; nil when neither were set
    Metadata any           // raw value passed to WithMetadata
}
```

## TestLoggingLibrary API

| Method            | Purpose                                                  |
|-------------------|----------------------------------------------------------|
| `Lines()`         | Snapshot copy of all captured lines                      |
| `GetLastLine()`   | Most recent line (does not remove); nil if empty         |
| `PopLine()`       | Most recent line (removes it); nil if empty              |
| `ClearLines()`    | Drop all captured lines                                  |
| `Len()`           | Number of captured lines                                 |
| `Log(line)`       | Manually append a line (useful for adapter tests)        |

All methods are safe for concurrent use.

## Reaching the Library Back

The transport's `GetLoggerInstance` returns `*TestLoggingLibrary`:

```go
lib := log.GetLoggerInstance("test").(*lltest.TestLoggingLibrary)
lib.ClearLines()
```

## See Also

- [Mocking](/logging-api/mocking), examples and patterns for using this transport in your test suite, plus `loglayer.NewMock()` for the silent-mock case.
