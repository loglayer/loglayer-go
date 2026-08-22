### Renderers

Self-contained transports that format the entry and write it to an `io.Writer`. Pick one of these when you want LogLayer to do the rendering itself.

<div class="module-list-table">

| Name | Version | Go Reference | Description |
|------|---------|--------------|-------------|
| [Blank](/transports/blank) | [![Version](https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=transports/blank/v*&sort=date&label=version&style=flat-square&color=blue)](https://github.com/loglayer/loglayer-go/releases?q=transports/blank/&expanded=true) | [![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/blank/v3.svg)](https://pkg.go.dev/go.loglayer.dev/transports/blank/v3) | Delegates dispatch to a user-supplied function. For prototyping or one-off integrations. |
| [CLI](/transports/cli) | [![Version](https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=transports/cli/v*&sort=date&label=version&style=flat-square&color=blue)](https://github.com/loglayer/loglayer-go/releases?q=transports/cli/&expanded=true) | [![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/cli/v3.svg)](https://pkg.go.dev/go.loglayer.dev/transports/cli/v3) | Tuned for CLI apps: short level prefixes, stdout/stderr routing, TTY-detected color, no timestamps. |
| [Console](/transports/console) | [![Version](https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=transports/console/v*&sort=date&label=version&style=flat-square&color=blue)](https://github.com/loglayer/loglayer-go/releases?q=transports/console/&expanded=true) | [![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/console/v3.svg)](https://pkg.go.dev/go.loglayer.dev/transports/console/v3) | Plain `fmt.Println`-style output to stdout/stderr; minimal formatting. |
| [Pretty](/transports/pretty) | [![Version](https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=transports/pretty/v*&sort=date&label=version&style=flat-square&color=blue)](https://github.com/loglayer/loglayer-go/releases?q=transports/pretty/&expanded=true) | [![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/pretty/v3.svg)](https://pkg.go.dev/go.loglayer.dev/transports/pretty/v3) | Colorized, theme-aware terminal output. **Recommended for local dev.** |
| [Structured](/transports/structured) | [![Version](https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=transports/structured/v*&sort=date&label=version&style=flat-square&color=blue)](https://github.com/loglayer/loglayer-go/releases?q=transports/structured/&expanded=true) | [![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/structured/v3.svg)](https://pkg.go.dev/go.loglayer.dev/transports/structured/v3) | One JSON object per log entry. Recommended for production. |
| [Testing](/transports/testing) | [![Version](https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=transports/testing/v*&sort=date&label=version&style=flat-square&color=blue)](https://github.com/loglayer/loglayer-go/releases?q=transports/testing/&expanded=true) | [![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/testing/v3.svg)](https://pkg.go.dev/go.loglayer.dev/transports/testing/v3) | Captures entries in memory for tests. |

</div>

### Cloud

Managed log services. Async + batched by default; site-aware where applicable.

<div class="module-list-table">

| Name | Version | Go Reference | Description |
|------|---------|--------------|-------------|
| [Axiom](/transports/axiom) | [![Version](https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=transports/axiom/v*&sort=date&label=version&style=flat-square&color=blue)](https://github.com/loglayer/loglayer-go/releases?q=transports/axiom/&expanded=true) | [![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/axiom/v3.svg)](https://pkg.go.dev/go.loglayer.dev/transports/axiom/v3) | Ships logs to Axiom via caller-supplied `*axiom.Client`. NDJSON ingestion with configurable message field. |
| [Better Stack](/transports/betterstack) | [![Version](https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=transports/betterstack/v*&sort=date&label=version&style=flat-square&color=blue)](https://github.com/loglayer/loglayer-go/releases?q=transports/betterstack/&expanded=true) | [![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/betterstack.svg)](https://pkg.go.dev/go.loglayer.dev/transports/betterstack) | Ships logs to Better Stack via HTTP intake. Source token auth, configurable timestamp field. |
| [Datadog](/transports/datadog) | [![Version](https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=transports/datadog/v*&sort=date&label=version&style=flat-square&color=blue)](https://github.com/loglayer/loglayer-go/releases?q=transports/datadog/&expanded=true) | [![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/datadog/v2.svg)](https://pkg.go.dev/go.loglayer.dev/transports/datadog/v2) | Datadog Logs HTTP intake. Site-aware URL, DD-API-KEY header, status mapping. |
| [Google Cloud Logging](/transports/gcplogging) | [![Version](https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=transports/gcplogging/v*&sort=date&label=version&style=flat-square&color=blue)](https://github.com/loglayer/loglayer-go/releases?q=transports/gcplogging/&expanded=true) | [![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/gcplogging/v3.svg)](https://pkg.go.dev/go.loglayer.dev/transports/gcplogging/v3) | Forwards entries to a caller-supplied `*logging.Logger` from `cloud.google.com/go/logging`. Severity mapping, root-level Entry skeleton, async + sync dispatch. |
| [New Relic](/transports/newrelic) | [![Version](https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=transports/newrelic/v*&sort=date&label=version&style=flat-square&color=blue)](https://github.com/loglayer/loglayer-go/releases?q=transports/newrelic/&expanded=true) | [![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/newrelic.svg)](https://pkg.go.dev/go.loglayer.dev/transports/newrelic) | New Relic Log Ingest API. Site-aware URL, api-key header, LogEvent encoding. |
| [Sentry](/transports/sentry) | [![Version](https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=transports/sentry/v*&sort=date&label=version&style=flat-square&color=blue)](https://github.com/loglayer/loglayer-go/releases?q=transports/sentry/&expanded=true) | [![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/sentry/v3.svg)](https://pkg.go.dev/go.loglayer.dev/transports/sentry/v3) | Forwards entries to a `sentry.Logger`. Routes fatal/panic through `LFatal` so loglayer's core controls termination. |

</div>

### Other Transports

Generic shippers and on-disk sinks.

<div class="module-list-table">

| Name | Version | Go Reference | Description |
|------|---------|--------------|-------------|
| [File (Lumberjack)](/transports/lumberjack) | [![Version](https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=transports/lumberjack/v*&sort=date&label=version&style=flat-square&color=blue)](https://github.com/loglayer/loglayer-go/releases?q=transports/lumberjack/&expanded=true) | [![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/lumberjack/v3.svg)](https://pkg.go.dev/go.loglayer.dev/transports/lumberjack/v3) | One JSON object per line written to a rotating file. Backed by `lumberjack.v2`. |
| [HTTP](/transports/http) | [![Version](https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=transports/http/v*&sort=date&label=version&style=flat-square&color=blue)](https://github.com/loglayer/loglayer-go/releases?q=transports/http/&expanded=true) | [![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/http/v3.svg)](https://pkg.go.dev/go.loglayer.dev/transports/http/v3) | Generic batched HTTP POST to any endpoint. Pluggable Encoder. |
| [OpenTelemetry Logs](/transports/otellog) | [![Version](https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=transports/otellog/v*&sort=date&label=version&style=flat-square&color=blue)](https://github.com/loglayer/loglayer-go/releases?q=transports/otellog/&expanded=true) | [![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/otellog/v3.svg)](https://pkg.go.dev/go.loglayer.dev/transports/otellog/v3) | Emits to an OTel `log.Logger`. Forwards `WithContext` so SDK processors can correlate with the active span. |

</div>

### Supported Loggers

Transports that hand the entry off to an existing third-party logger you already configure. Pick one of these when you have an established logging stack and want LogLayer's API on top.

<div class="module-list-table">

| Name | Version | Go Reference | Description |
|------|---------|--------------|-------------|
| [charmbracelet/log](/transports/charmlog) | [![Version](https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=transports/charmlog/v*&sort=date&label=version&style=flat-square&color=blue)](https://github.com/loglayer/loglayer-go/releases?q=transports/charmlog/&expanded=true) | [![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/charmlog/v3.svg)](https://pkg.go.dev/go.loglayer.dev/transports/charmlog/v3) | Pretty terminal-friendly logger from Charm |
| [log/slog](/transports/slog) | [![Version](https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=transports/slog/v*&sort=date&label=version&style=flat-square&color=blue)](https://github.com/loglayer/loglayer-go/releases?q=transports/slog/&expanded=true) | [![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/slog/v3.svg)](https://pkg.go.dev/go.loglayer.dev/transports/slog/v3) | Wraps a stdlib `*slog.Logger`. Forwards `WithContext` to handlers. |
| [logrus](/transports/logrus) | [![Version](https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=transports/logrus/v*&sort=date&label=version&style=flat-square&color=blue)](https://github.com/loglayer/loglayer-go/releases?q=transports/logrus/&expanded=true) | [![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/logrus/v3.svg)](https://pkg.go.dev/go.loglayer.dev/transports/logrus/v3) | The classic structured logger |
| [phuslu/log](/transports/phuslu) | [![Version](https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=transports/phuslu/v*&sort=date&label=version&style=flat-square&color=blue)](https://github.com/loglayer/loglayer-go/releases?q=transports/phuslu/&expanded=true) | [![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/phuslu/v3.svg)](https://pkg.go.dev/go.loglayer.dev/transports/phuslu/v3) | High-performance zero-alloc JSON logger. Always exits on fatal. |
| [Zap](/transports/zap) | [![Version](https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=transports/zap/v*&sort=date&label=version&style=flat-square&color=blue)](https://github.com/loglayer/loglayer-go/releases?q=transports/zap/&expanded=true) | [![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/zap/v3.svg)](https://pkg.go.dev/go.loglayer.dev/transports/zap/v3) | Wraps a `*zap.Logger` |
| [Zerolog](/transports/zerolog) | [![Version](https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=transports/zerolog/v*&sort=date&label=version&style=flat-square&color=blue)](https://github.com/loglayer/loglayer-go/releases?q=transports/zerolog/&expanded=true) | [![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/zerolog/v3.svg)](https://pkg.go.dev/go.loglayer.dev/transports/zerolog/v3) | Wraps a `*zerolog.Logger` |

</div>
