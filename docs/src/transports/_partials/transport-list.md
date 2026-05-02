### Renderers

Self-contained transports that format the entry and write it to an `io.Writer`. Pick one of these when you want LogLayer to do the rendering itself.

<div class="module-list-table">

| Name | Version | Go Reference | Description |
|------|---------|--------------|-------------|
| [Pretty](/transports/pretty) | [![Version](https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=transports/pretty/v*&sort=date&label=version&style=flat-square&color=blue)](https://github.com/loglayer/loglayer-go/releases?q=transports/pretty/&expanded=true) | [![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/pretty/v2.svg)](https://pkg.go.dev/go.loglayer.dev/transports/pretty/v2) | Colorized, theme-aware terminal output. **Recommended for local dev.** |
| [Structured](/transports/structured) | [![Version](https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=transports/structured/v*&sort=date&label=version&style=flat-square&color=blue)](https://github.com/loglayer/loglayer-go/releases?q=transports/structured/&expanded=true) | [![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/structured/v2.svg)](https://pkg.go.dev/go.loglayer.dev/transports/structured/v2) | One JSON object per log entry. Recommended for production. |
| [Console](/transports/console) | [![Version](https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=transports/console/v*&sort=date&label=version&style=flat-square&color=blue)](https://github.com/loglayer/loglayer-go/releases?q=transports/console/&expanded=true) | [![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/console/v2.svg)](https://pkg.go.dev/go.loglayer.dev/transports/console/v2) | Plain `fmt.Println`-style output to stdout/stderr; minimal formatting. |
| [CLI](/transports/cli) | [![Version](https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=transports/cli/v*&sort=date&label=version&style=flat-square&color=blue)](https://github.com/loglayer/loglayer-go/releases?q=transports/cli/&expanded=true) | [![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/cli/v2.svg)](https://pkg.go.dev/go.loglayer.dev/transports/cli/v2) | Tuned for CLI apps: short level prefixes, stdout/stderr routing, TTY-detected color, no timestamps. |
| [Testing](/transports/testing) | [![Version](https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=transports/testing/v*&sort=date&label=version&style=flat-square&color=blue)](https://github.com/loglayer/loglayer-go/releases?q=transports/testing/&expanded=true) | [![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/testing/v2.svg)](https://pkg.go.dev/go.loglayer.dev/transports/testing/v2) | Captures entries in memory for tests. |
| [Blank](/transports/blank) | [![Version](https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=transports/blank/v*&sort=date&label=version&style=flat-square&color=blue)](https://github.com/loglayer/loglayer-go/releases?q=transports/blank/&expanded=true) | [![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/blank/v2.svg)](https://pkg.go.dev/go.loglayer.dev/transports/blank/v2) | Delegates dispatch to a user-supplied function. For prototyping or one-off integrations. |

</div>

### Cloud

Managed log services. Async + batched by default; site-aware where applicable.

<div class="module-list-table">

| Name | Version | Go Reference | Description |
|------|---------|--------------|-------------|
| [Datadog](/transports/datadog) | [![Version](https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=transports/datadog/v*&sort=date&label=version&style=flat-square&color=blue)](https://github.com/loglayer/loglayer-go/releases?q=transports/datadog/&expanded=true) | [![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/datadog/v2.svg)](https://pkg.go.dev/go.loglayer.dev/transports/datadog/v2) | Datadog Logs HTTP intake. Site-aware URL, DD-API-KEY header, status mapping. |
| [Google Cloud Logging](/transports/gcplogging) | [![Version](https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=transports/gcplogging/v*&sort=date&label=version&style=flat-square&color=blue)](https://github.com/loglayer/loglayer-go/releases?q=transports/gcplogging/&expanded=true) | [![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/gcplogging/v2.svg)](https://pkg.go.dev/go.loglayer.dev/transports/gcplogging/v2) | Forwards entries to a caller-supplied `*logging.Logger` from `cloud.google.com/go/logging`. Severity mapping, root-level Entry skeleton, async + sync dispatch. |
| [Sentry](/transports/sentry) | [![Version](https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=transports/sentry/v*&sort=date&label=version&style=flat-square&color=blue)](https://github.com/loglayer/loglayer-go/releases?q=transports/sentry/&expanded=true) | [![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/sentry/v2.svg)](https://pkg.go.dev/go.loglayer.dev/transports/sentry/v2) | Forwards entries to a `sentry.Logger`. Routes fatal/panic through `LFatal` so loglayer's core controls termination. |

</div>

### Other Transports

Generic shippers and on-disk sinks.

<div class="module-list-table">

| Name | Version | Go Reference | Description |
|------|---------|--------------|-------------|
| [HTTP](/transports/http) | [![Version](https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=transports/http/v*&sort=date&label=version&style=flat-square&color=blue)](https://github.com/loglayer/loglayer-go/releases?q=transports/http/&expanded=true) | [![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/http/v2.svg)](https://pkg.go.dev/go.loglayer.dev/transports/http/v2) | Generic batched HTTP POST to any endpoint. Pluggable Encoder. |
| [File (Lumberjack)](/transports/lumberjack) | [![Version](https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=transports/lumberjack/v*&sort=date&label=version&style=flat-square&color=blue)](https://github.com/loglayer/loglayer-go/releases?q=transports/lumberjack/&expanded=true) | [![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/lumberjack/v2.svg)](https://pkg.go.dev/go.loglayer.dev/transports/lumberjack/v2) | One JSON object per line written to a rotating file. Backed by `lumberjack.v2`. |
| [OpenTelemetry Logs](/transports/otellog) | [![Version](https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=transports/otellog/v*&sort=date&label=version&style=flat-square&color=blue)](https://github.com/loglayer/loglayer-go/releases?q=transports/otellog/&expanded=true) | [![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/otellog/v2.svg)](https://pkg.go.dev/go.loglayer.dev/transports/otellog/v2) | Emits to an OTel `log.Logger`. Forwards `WithContext` so SDK processors can correlate with the active span. |

</div>

### Supported Loggers

Transports that hand the entry off to an existing third-party logger you already configure. Pick one of these when you have an established logging stack and want LogLayer's API on top.

<div class="module-list-table">

| Name | Version | Go Reference | Description |
|------|---------|--------------|-------------|
| [Zerolog](/transports/zerolog) | [![Version](https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=transports/zerolog/v*&sort=date&label=version&style=flat-square&color=blue)](https://github.com/loglayer/loglayer-go/releases?q=transports/zerolog/&expanded=true) | [![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/zerolog/v2.svg)](https://pkg.go.dev/go.loglayer.dev/transports/zerolog/v2) | Wraps a `*zerolog.Logger` |
| [Zap](/transports/zap) | [![Version](https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=transports/zap/v*&sort=date&label=version&style=flat-square&color=blue)](https://github.com/loglayer/loglayer-go/releases?q=transports/zap/&expanded=true) | [![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/zap/v2.svg)](https://pkg.go.dev/go.loglayer.dev/transports/zap/v2) | Wraps a `*zap.Logger` |
| [log/slog](/transports/slog) | [![Version](https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=transports/slog/v*&sort=date&label=version&style=flat-square&color=blue)](https://github.com/loglayer/loglayer-go/releases?q=transports/slog/&expanded=true) | [![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/slog/v2.svg)](https://pkg.go.dev/go.loglayer.dev/transports/slog/v2) | Wraps a stdlib `*slog.Logger`. Forwards `WithContext` to handlers. |
| [phuslu/log](/transports/phuslu) | [![Version](https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=transports/phuslu/v*&sort=date&label=version&style=flat-square&color=blue)](https://github.com/loglayer/loglayer-go/releases?q=transports/phuslu/&expanded=true) | [![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/phuslu/v2.svg)](https://pkg.go.dev/go.loglayer.dev/transports/phuslu/v2) | High-performance zero-alloc JSON logger. Always exits on fatal. |
| [logrus](/transports/logrus) | [![Version](https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=transports/logrus/v*&sort=date&label=version&style=flat-square&color=blue)](https://github.com/loglayer/loglayer-go/releases?q=transports/logrus/&expanded=true) | [![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/logrus/v2.svg)](https://pkg.go.dev/go.loglayer.dev/transports/logrus/v2) | The classic structured logger |
| [charmbracelet/log](/transports/charmlog) | [![Version](https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=transports/charmlog/v*&sort=date&label=version&style=flat-square&color=blue)](https://github.com/loglayer/loglayer-go/releases?q=transports/charmlog/&expanded=true) | [![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/charmlog/v2.svg)](https://pkg.go.dev/go.loglayer.dev/transports/charmlog/v2) | Pretty terminal-friendly logger from Charm |

</div>
