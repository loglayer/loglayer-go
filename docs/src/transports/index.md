---
title: Transports
description: How transports work and which ones ship with LogLayer for Go.
---

# Transports

A transport is what actually emits a log entry: to stdout, to a file, to a third-party logger like zerolog or zap, to a remote service. The core LogLayer assembles the entry; the transport renders it.

## Picking a transport

A small decision tree:

**Are you in development?** Use [`pretty`](/transports/pretty) — colorized, theme-aware terminal output. Switch to a production transport later by changing one line.

**Do you already have a logging stack you want to keep?** Wrap it. Pick the matching wrapper:
- [`zerolog`](/transports/zerolog) — `*zerolog.Logger`
- [`zap`](/transports/zap) — `*zap.Logger`
- [`slog`](/transports/slog) — stdlib `*slog.Logger` (forwards `WithCtx` to handlers, useful when your slog stack already has handlers wired to OTel etc.)
- [`logrus`](/transports/logrus) — `*logrus.Logger` (the classic)
- [`charmlog`](/transports/charmlog) — `charmbracelet/log`
- [`phuslu`](/transports/phuslu) — `phuslu/log` (the fastest of the bunch, but **always exits on fatal**)

**Are you starting from scratch and want JSON output?** Use [`structured`](/transports/structured). One JSON object per line, no third-party dependencies. Recommended for production.

**Are you shipping to a service over the network?**
- Datadog Logs HTTP intake → [`datadog`](/transports/datadog) (site-aware URL, status mapping, batched async).
- Generic HTTP endpoint (Loki, Splunk, custom backend) → [`http`](/transports/http) with a custom Encoder.
- OpenTelemetry Logs pipeline → [`otellog`](/transports/otellog) (separate Go module, requires Go 1.25+).

**Are you writing tests?**
- Asserting on log output → [`testing`](/transports/testing) — captures entries to typed `LogLine` structs.
- Want a logger that just doesn't emit anything → `loglayer.NewMock()` (see [Mocking](/logging-api/mocking)).

**Are you prototyping a one-off integration?** [`blank`](/transports/blank) takes a function, calls it for every emission. Useful for "just send these to Slack" or similar.

::: tip Multiple transports at once
A single `LogLayer` can fan out to several transports simultaneously — pretty during development *plus* structured to a file *plus* HTTP shipping. See [Multiple Transports](/transports/multiple-transports). For per-transport routing rules, see [Groups](/logging-api/groups).
:::

::: tip Output shapes vary
Each transport has its own way of rendering an entry: `structured` produces flat JSON; `zerolog` and `zap` follow their library's conventions; `pretty` is colorized terminal output; `otellog` emits an OTel `log.Record`. Every per-transport page has a "Metadata Handling" or "Basic Usage" section showing a representative output for that transport — that's the place to look when you need to know exactly what your aggregator will receive.
:::

## Available transports

<!--@include: ./_partials/transport-list.md-->

A single LogLayer can fan out to several transports at once — see [Multiple Transports](/transports/multiple-transports). To wrap a logger LogLayer doesn't ship a transport for, see [Creating Transports](/transports/creating-transports).

## Level mapping

LogLayer has six levels (`Trace`, `Debug`, `Info`, `Warn`, `Error`, `Fatal`). Most transports map them straight through. Two deviations to know about, both documented in detail on the transport's own page:

- **Trace collapses to Debug** in transports whose underlying library has no Trace level: `zap`, `slog`, `charmlog`. `Trace` calls go through but render at the Debug level downstream.
- **Fatal always exits the process** in `phuslu` because the underlying library calls `os.Exit` from its fatal dispatch path. `Config.DisableFatalExit` cannot suppress it. Every other wrapper neutralizes its library's Fatal so the LogLayer core controls the exit decision.

The per-transport page calls out its own level mapping (and the underlying library's quirks) when there's anything beyond a straight pass-through.
