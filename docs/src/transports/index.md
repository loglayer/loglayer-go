---
title: Transports
description: How transports work and which ones ship with LogLayer for Go.
---

# Transports

A transport is what actually emits a log entry: to stdout, to a file, to a third-party logger like zerolog or zap, to a remote service. The core LogLayer assembles the entry; the transport renders it.

## Available transports

<!--@include: ./_partials/transport-list.md-->

A single LogLayer can fan out to several transports at once, see [Multiple Transports](/transports/multiple-transports). To wrap a logger LogLayer doesn't ship a transport for, see [Creating Transports](/transports/creating-transports).
