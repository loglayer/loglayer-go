---
title: "LogLayer for Go: structured logging with a fluent API"
description: A structured logging library with a fluent API for specifying log messages, fields, metadata, and errors.

layout: home

hero:
  name: "LogLayer"
  text: "Unifies Go Logging"
  tagline: "A consistent logging experience for Go, on top of any logging library."
  image:
    src: /images/loglayer.png
    alt: LogLayer logo by Akshaya Madhavan
  actions:
    - theme: brand
      text: Why Use LogLayer?
      link: /introduction
    - theme: alt
      text: Quickstart
      link: /getting-started
    - theme: alt
      text: GitHub (MIT Licensed)
      link: https://github.com/loglayer/loglayer-go

features:
  - title: Structured Logging
    details: Write logs with a fluent API that makes adding fields, metadata, and errors simple.
  - title: Bring Your Own Logger
    details: Start with the built-in pretty or structured transport, then switch to zerolog, zap, slog, or your own logger later without changing application code.
  - title: Extensible Plugin System
    details: Transform, enrich, redact, and filter logs at six lifecycle points to customize the pipeline end to end.
  - title: Multi-Transport Fan-out
    details: Send the same entry to several transports at once, and use named groups to route specific entries to specific transports.
  - title: HTTP and Cloud Shipping
    details: Built-in batched HTTP transport plus a Datadog Logs intake transport. Drop-in support for any service with a JSON-over-HTTP intake.
  - title: First-class Testing
    details: Capture entries with the testing transport for assertions, or use loglayer.NewMock() for a silent logger in code under test.
---

## Quick Example

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

    // WithFields returns a NEW logger; assign it.
    log = log.WithFields(loglayer.Fields{"service": "api"})

    log.WithMetadata(loglayer.Metadata{"userId": "1234"}).
        WithError(errors.New("something went wrong")).
        Error("user action failed")
}
```

```json
{
  "level": "error",
  "time": "2026-04-25T12:00:00Z",
  "msg": "user action failed",
  "service": "api",
  "userId": "1234",
  "err": { "message": "something went wrong" }
}
```

## Transports

<!--@include: ./transports/_partials/transport-list.md-->

## Plugins

Hook into the LogLayer pipeline to transform metadata, fields, data, messages, log level, or per-transport dispatch.

<!--@include: ./plugins/_partials/plugin-list.md-->

LogLayer for Go is made with ❤️ by [Theo Gravity](https://suteki.nu) / [Disaresta](https://disaresta.com). Logo by [Akshaya Madhavan](https://www.linkedin.com/in/akshaya-madhavan).
