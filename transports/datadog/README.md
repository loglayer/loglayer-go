# go.loglayer.dev/transports/datadog

[![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/datadog.svg)](https://pkg.go.dev/go.loglayer.dev/transports/datadog)

Datadog Logs HTTP intake transport for LogLayer. Built on `transports/http` with a Datadog-specific encoder, site-aware URL, and `DD-API-KEY` header. Rejects non-https URLs by default; opt-in via `Config.AllowInsecureURL` for on-prem TLS-terminating proxies. The API key is redacted in `String()` output and tagged `json:"-"` to keep it out of accidental config dumps.

## Install

```sh
go get go.loglayer.dev/transports/datadog
```

## Documentation

Full reference and examples: <https://go.loglayer.dev/transports/datadog>

Main library: <https://go.loglayer.dev>
