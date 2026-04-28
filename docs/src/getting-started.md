---
title: Get started with LogLayer for Go
description: Install LogLayer, pick a transport, and write your first structured log.
---

# Getting Started

LogLayer for Go targets **Go 1.25+** for the main module: `go.loglayer.dev`. Most transports are sub-packages of that module, so you only pull in dependencies for the transports you actually use. Individual transports and plugins call out any stricter requirement on their per-page docs.

## Installation

```sh
go get go.loglayer.dev
```

## Basic Usage with the Structured Transport

The simplest way to start is the [Structured Transport](/transports/structured), which writes one JSON object per log entry to `os.Stdout`:

```go
package main

import (
    "errors"

    "go.loglayer.dev"
    "go.loglayer.dev/transports/structured"
)

func main() {
    log := loglayer.New(loglayer.Config{
        Transport: structured.New(structured.Config{}),
    })

    // Basic logging
    log.Info("Hello world!")
    // {"level":"info","time":"2026-04-25T12:00:00Z","msg":"Hello world!"}

    // With metadata (loglayer.Metadata is an alias for map[string]any)
    log.WithMetadata(loglayer.Metadata{"user": "alice"}).Info("User logged in")
    // {"level":"info","time":"...","msg":"User logged in","user":"alice"}

    // With persistent fields (WithFields returns a NEW logger; assign it)
    reqLog := log.WithFields(loglayer.Fields{"requestId": "123"})
    reqLog.Info("Processing request")
    // {"level":"info","time":"...","msg":"Processing request","requestId":"123"}

    // With an error
    log.WithError(errors.New("something went wrong")).Error("Failed")
    // {"level":"error","time":"...","msg":"Failed","err":{"message":"something went wrong"}}
}
```

::: tip Pretty terminal output
For local development, the [Pretty Transport](/transports/pretty) gives you colorized, theme-aware output with three view modes. Much easier to scan than raw JSON or the basic [Console Transport](/transports/console).
:::

## Configure an Error Serializer

The default error format is `{"message": err.Error()}`. To expand `fmt.Errorf("...: %w", err)` chains and `errors.Join` lists into a `causes` array, use `loglayer.UnwrappingErrorSerializer`:

```go
log := loglayer.New(loglayer.Config{
    Transport:       structured.New(structured.Config{}),
    ErrorSerializer: loglayer.UnwrappingErrorSerializer,
})

log.WithError(fmt.Errorf("op failed: %w", io.EOF)).Error("oops")
// {"err":{"message":"op failed: EOF","causes":[{"message":"EOF"}]}}
```

For stack traces, custom shapes, or other options, see [Error Handling](/logging-api/error-handling).

## Using a Logger Wrapper

If you already have an existing logging stack, LogLayer can wrap it so your call sites use the LogLayer API while emission goes through the underlying logger you've already configured. Here it is for `zerolog`:

```go
import (
    zlog "github.com/rs/zerolog"
    "os"

    "go.loglayer.dev"
    llzero "go.loglayer.dev/transports/zerolog"
)

z := zlog.New(os.Stderr).With().Timestamp().Logger()
log := loglayer.New(loglayer.Config{
    Transport: llzero.New(llzero.Config{Logger: &z}),
})

log.WithFields(loglayer.Fields{"requestId": "abc"}).Info("served")
```

The same shape works for `zap`, `log/slog`, `logrus`, `charmbracelet/log`, and `phuslu/log`. See the [Transports overview](/transports/) for the full list and per-wrapper config.

## Next Steps

- [Configuration](/configuration): every option in `Config`.
- [Cheat Sheet](/cheatsheet): quick reference for the common methods.
- [Logging API](/logging-api/basic-logging): the full method-by-method tour.
- [Transports](/transports/): pick a backend and configure it.
