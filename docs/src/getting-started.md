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

The default error format is intentionally minimal: `{"message": err.Error()}`. No chain expansion, no stack trace. **For most production code you'll want richer error output**, and the configuration is one knob away.

**Recommended: `loglayer.UnwrappingErrorSerializer`.** It walks the standard-library error machinery (`errors.Unwrap` chains and `errors.Join`'s `Unwrap() []error`) and surfaces every wrapped cause as a `causes` array. Zero dependencies; works with idiomatic `fmt.Errorf("...: %w", err)` exactly the way you already write it:

```go
log := loglayer.New(loglayer.Config{
    Transport:       structured.New(structured.Config{}),
    ErrorSerializer: loglayer.UnwrappingErrorSerializer,
})

log.WithError(fmt.Errorf("op failed: %w", io.EOF)).Error("oops")
// {"err":{"message":"op failed: EOF","causes":[{"message":"EOF"}]}}
```

This is the right default for almost all services. Stack traces aren't free (allocation per error) and the stack you'd get is the construction site, not the failure point — which often duplicates information [`Config.Source`](/configuration#source-caller-info) already provides at zero per-error cost.

### When you also want stack traces: `eris`

Reach for [`rotisserie/eris`](https://github.com/rotisserie/eris) when you specifically need stack capture (debugging unfamiliar code paths, panic-hunting, async work where call-site info is hard to come by). `eris.ToJSON` returns `map[string]any`, which slots straight into `ErrorSerializer`:

```sh
go get github.com/rotisserie/eris
```

```go
import (
    "github.com/rotisserie/eris"
    "go.loglayer.dev"
    "go.loglayer.dev/transports/structured"
)

log := loglayer.New(loglayer.Config{
    Transport: structured.New(structured.Config{}),
    ErrorSerializer: func(err error) map[string]any {
        return eris.ToJSON(err, true) // true = include stack trace
    },
})

err := eris.New("connection refused")
log.WithError(err).Error("db query failed")
// {"level":"error","msg":"db query failed","err":{"root":{"message":"connection refused","stack":[...]}}}
```

Note that eris captures stacks at construction (`eris.New`, `eris.Wrap`); plain stdlib errors won't get stacks unless you wrap them with `eris.Wrap`.

See [Error Handling](/logging-api/error-handling) for the full serializer reference, including writing your own from scratch.

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
