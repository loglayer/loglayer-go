---
title: Transports
description: How transports work and which ones ship with LogLayer for Go.
---

# Transports

A transport is what actually emits a log entry: to stdout, to a JSON file, to a third-party logger like zerolog or zap, to a remote service. The core LogLayer assembles the entry; the transport renders it.

Transports come in two flavors: whether you bring your own logger or whether the transport renders the entry directly.

::: tip Picking a transport
- **Local development:** [Pretty](/transports/pretty): colorized, theme-aware output that's easy to scan.
- **Production:** [Structured](/transports/structured) for self-contained JSON, or [Zerolog](/transports/zerolog) / [Zap](/transports/zap) to wrap an existing logger.
- **Tests:** [Testing](/transports/testing) for assertions, or `loglayer.NewMock()` to silence logs entirely. See [Mocking](/logging-api/mocking).
:::

<!--@include: ./_partials/transport-list.md-->

## Multiple Transports

A single LogLayer can fan out to several transports at once. See [Multiple Transports](/transports/multiple-transports).

## Writing Your Own

Any logger you can reach with code can become a LogLayer transport in ~60 lines. See [Creating Transports](/transports/creating-transports).
