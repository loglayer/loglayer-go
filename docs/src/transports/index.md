---
title: Transports
description: How transports work and which ones ship with LogLayer for Go.
---

# Transports

A transport is what actually emits a log entry: to stdout, to a file, to a third-party logger like zerolog or zap, to a remote service. The core LogLayer assembles the entry; the transport renders it.

::: tip Picking one
- **Local development:** [Pretty](/transports/pretty) — colorized terminal output.
- **Production:** [Structured](/transports/structured) for self-contained JSON, or wrap an existing logger like [Zerolog](/transports/zerolog), [Zap](/transports/zap), or [slog](/transports/slog).
- **Tests:** [Testing](/transports/testing) for assertions, or `loglayer.NewMock()` to silence logs (see [Mocking](/logging-api/mocking)).
:::

## Available transports

<!--@include: ./_partials/transport-list.md-->

A single LogLayer can fan out to several transports at once — see [Multiple Transports](/transports/multiple-transports). To wrap a logger LogLayer doesn't ship a transport for, see [Creating Transports](/transports/creating-transports).
