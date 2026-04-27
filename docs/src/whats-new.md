---
title: What's New
description: User-visible changes to LogLayer for Go.
---

# What's New

Changes are recorded here once they ship. Until v0.1.0 lands, this page describes the initial release at a high level; the full API is documented across the rest of the site.

From v0.1.0 forward, [Release Please](https://github.com/googleapis/release-please) maintains the project's [`CHANGELOG.md`](https://github.com/loglayer/loglayer-go/blob/main/CHANGELOG.md) automatically from conventional commits. This page captures user-visible highlights for each release; it's curated by hand to keep the narrative tight.

## v0.1.0 (unreleased)

Initial release. Pre-1.0; the API may still shift before v1.

LogLayer for Go is a transport-agnostic structured logging facade. One fluent API on top of any logging library, JSON, terminal, HTTP, OpenTelemetry, or your own transport.

### What ships

- **Core**: fluent builder API with persistent fields, per-call metadata, errors, Go context, and group-based routing. Distinct `Fields`/`Metadata`/`Data` types so the compiler catches misuse. Every method safe from any goroutine. See [Getting Started](/getting-started).
- **Renderers**: [pretty](/transports/pretty) (colorized terminal, recommended for dev), [structured](/transports/structured) (JSON, recommended for production), [console](/transports/console), [testing](/transports/testing), [blank](/transports/blank).
- **Logger wrappers**: [zerolog](/transports/zerolog), [zap](/transports/zap), [log/slog](/transports/slog), [logrus](/transports/logrus), [charmbracelet/log](/transports/charmlog), [phuslu/log](/transports/phuslu). Drop LogLayer in front of an existing stack without rewriting your call sites.
- **Network**: [HTTP](/transports/http) (generic batched POST with pluggable encoder), [Datadog](/transports/datadog) (Logs HTTP intake with on-prem URL override).
- **OpenTelemetry**: [`transports/otellog`](/transports/otellog) ships entries through the OTel logs pipeline; [`plugins/oteltrace`](/plugins/oteltrace) injects `trace_id`/`span_id` on any transport. Both live in their own Go modules so the OTel SDK's dep graph doesn't bind users who don't import them.
- **Plugins**: six lifecycle hooks with centralized panic recovery. Built-in: [redact](/plugins/redact), [datadogtrace](/plugins/datadogtrace), [oteltrace](/plugins/oteltrace). Author your own per [Creating Plugins](/plugins/creating-plugins).
- **HTTP middleware**: [`integrations/loghttp`](/integrations/loghttp) derives a per-request logger, binds `r.Context()`, emits a "request completed" log on every response (or "request panicked" with the panic value if a handler crashes).
- **Runtime control**: change levels, hot-swap transports, add/remove plugins, mute fields/metadata, all live and concurrency-safe.
- **Defensive defaults**: console and pretty sanitize message strings against CRLF / ANSI / Unicode-bidi injection; `loghttp` sanitizes incoming HTTP headers; `maputil.Cloner` is cycle-safe; the Datadog config redacts its API key via `String()` and a `json:"-"` tag.

### Migrating from another logger?

- [From log/slog](/migrating/from-slog)
- [From zerolog](/migrating/from-zerolog)
- [From zap](/migrating/from-zap)

For Go users coming from the TypeScript [`loglayer`](https://loglayer.dev) library, see [For TypeScript Developers](/for-typescript-developers) for the API mapping.

### Known gotchas

The [Common Pitfalls](/common-pitfalls) page collects the failure modes that bite first-time users — chiefly the "returns a new logger" pattern (assign the result of `WithFields` etc.), plugin nil-drop semantics, and the per-transport divergence in fatal-exit behavior (phuslu cannot be suppressed; everyone else honors `Config.DisableFatalExit`).
