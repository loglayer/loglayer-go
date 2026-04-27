---
title: "LogLayer for Go: a transport-agnostic structured logger"
description: "A layer on top of Go logging libraries that gives you a consistent fluent API for messages, metadata, and errors."

layout: home

hero:
  name: "LogLayer"
  text: "Unifies Go Logging"
  tagline: "A layer on top of Go logging libraries that gives you a consistent fluent API for messages, fields, metadata, and errors."
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
    details: Fluent API for attaching metadata, errors, and persistent fields, separated cleanly so per-call data never bleeds across logs.
  - title: Bring Your Own Logger
    details: Wrap zerolog, zap, or write your own transport. Switch logging libraries without touching your application code.
  - title: Multi-Transport Fan-out
    details: Send the same log entry to several backends at once, for example, zerolog locally and a structured JSON writer for shipping.
  - title: Type-flexible Metadata
    details: Pass a map, struct, or any value to WithMetadata, each transport decides how to serialize it.
  - title: First-class Testing
    details: A TestTransport with a mutex-safe TestLoggingLibrary captures every entry as a typed LogLine for clean assertions.
  - title: No Surprise os.Exit
    details: Fatal logs the entry but never terminates the process, termination is the caller's decision.
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

    log.WithFields(loglayer.Fields{"service": "api"})

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

LogLayer for Go is made with ❤️ by [Theo Gravity](https://suteki.nu) / [Disaresta](https://disaresta.com). Logo by [Akshaya Madhavan](https://www.linkedin.com/in/akshaya-madhavan).
