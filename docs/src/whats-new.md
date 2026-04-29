---
title: What's New
description: User-visible changes to LogLayer for Go.
---

# What's New

## v1.1.0 (2026-04-29)

Two structural changes since v1.0.x. Behaviour is unchanged for callers who weren't using the old `fmtlog` import path.

### Module layout

Eight previously-bundled packages are now their own Go modules — their import paths are unchanged but they version independently from the framework core:

| Package | New module path |
|---|---|
| Redact plugin | `go.loglayer.dev/plugins/redact` |
| Sampling plugin | `go.loglayer.dev/plugins/sampling` |
| Format-string plugin | `go.loglayer.dev/plugins/fmtlog` |
| Datadog APM trace injector | `go.loglayer.dev/plugins/datadogtrace` |
| HTTP middleware | `go.loglayer.dev/integrations/loghttp` |
| `slog.Handler` adapter | `go.loglayer.dev/integrations/sloghandler` |
| `log/slog` wrapper transport | `go.loglayer.dev/transports/slog` |
| Blank transport | `go.loglayer.dev/transports/blank` |

You may need a `go mod tidy` to pick up the new sub-module entries. Existing source code keeps compiling without changes.

The four packages that main's own tests/examples depend on (`transports/console`, `transports/structured`, `transports/testing`, `plugins/plugintest`) stay bundled in `go.loglayer.dev` to avoid require cycles.

The split's payoff: a future breaking change in any one of these now bumps only that sub-module's major version (`<package>/v2.x.x`), leaving `go.loglayer.dev` itself stable on v1.

### `go.loglayer.dev/fmtlog` → `go.loglayer.dev/plugins/fmtlog`

The fmtlog plugin moved from the top of the repo to under `plugins/` for consistency with every other plugin. If you were using the old import:

```diff
- import "go.loglayer.dev/fmtlog"
+ import "go.loglayer.dev/plugins/fmtlog"
```

No other API change. `fmtlog.New()` works identically.

This is technically a SemVer-breaking change (the old import path is gone), but ships under v1.1.0 rather than v2.0.0 — see the [CHANGELOG note](https://github.com/loglayer/loglayer-go/blob/main/CHANGELOG.md#110-2026-04-29) for the version-rationale.

### Coordinated v1.1.0 across all sub-modules

Every loglayer-go module — main, all transports, all plugins, all integrations — now sits at v1.1.0. Most sub-modules had no actual code change for this release; their v1.1.0 tag is a coordinated bump alongside main caused by the version-override commit applying globally. Each sub-module's own `CHANGELOG.md` carries a one-line "no code changes" note for the v1.1.0 entry.

---

## v1.0.0 (2026-04-29)

Initial release. Stable API; SemVer applies from this point forward.

Subsequent v1.0.x releases (v1.0.1, v1.0.2) are build- and test-only fixes (`go.sum` tidies, livetest repairs, release-please config). No user-visible behavior changes; full notes in [CHANGELOG.md](https://github.com/loglayer/loglayer-go/blob/main/CHANGELOG.md).

LogLayer for Go is a transport-agnostic structured logging facade. One fluent API on top of any logging library, JSON, terminal, HTTP, OpenTelemetry, or your own transport.

### What ships

- **Core API**: fluent builder with persistent fields, per-call metadata, errors, Go context, and group routing. Seven log levels (Trace through Panic). Every method safe from any goroutine. See [Getting Started](/getting-started).
- **Renderers**: [pretty](/transports/pretty), [structured](/transports/structured), [console](/transports/console), [testing](/transports/testing), [blank](/transports/blank).
- **Logger wrappers**: [zerolog](/transports/zerolog), [zap](/transports/zap), [log/slog](/transports/slog), [logrus](/transports/logrus), [charmbracelet/log](/transports/charmlog), [phuslu/log](/transports/phuslu).
- **Network transports**: [HTTP](/transports/http), [Datadog](/transports/datadog).
- **OpenTelemetry**: [`transports/otellog`](/transports/otellog) ships through the OTel logs pipeline; [`plugins/oteltrace`](/plugins/oteltrace) injects `trace_id`/`span_id` on any transport.
- **Plugins**: [redact](/plugins/redact), [sampling](/plugins/sampling), [fmtlog](/plugins/fmtlog), [datadogtrace](/plugins/datadogtrace), [oteltrace](/plugins/oteltrace). Six lifecycle hooks with centralized panic recovery; author your own per [Creating Plugins](/plugins/creating-plugins).
- **HTTP middleware**: [`integrations/loghttp`](/integrations/loghttp) binds a per-request logger to `r.Context()` and logs each response.
- **slog interop**: [`integrations/sloghandler`](/integrations/sloghandler) routes every `slog.Info(...)` call (yours and your dependencies') through the loglayer pipeline. Pairs with the [slog transport](/transports/slog) for the opposite direction.
- **stdlib log bridge**: `log.Writer(level)` / `log.NewLogLogger(level)` plumb anything that wants a `*log.Logger` or `io.Writer` (`http.Server.ErrorLog`, gorm, etc.) through loglayer.
- **Error chain expansion**: opt-in [`loglayer.UnwrappingErrorSerializer`](/logging-api/error-handling#built-in-unwrappingerrorserializer) surfaces every wrapped cause as a `causes` array.
- **Source / caller info**: opt-in via `Config.Source.Enabled`; the slog handler forwards `Record.PC` for free.
- **Runtime control**: live and concurrency-safe level changes, transport hot-swap, plugin add/remove, mute toggles.
- **Opt-in transport panic recovery**: `Config.OnTransportPanic` absorbs panics from `SendToLogger` so one bad sink can't suppress the others.
- **Defensive defaults**: CRLF / ANSI / Unicode-bidi sanitization in console and pretty, cross-host redirect refusal on the HTTP transport, redacted Datadog API key, cycle-safe `maputil.Cloner`.

For Go users coming from the TypeScript [`loglayer`](https://loglayer.dev) library, see [For TypeScript Developers](/for-typescript-developers) for the API mapping.
