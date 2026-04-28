# go.loglayer.dev/transports/datadog

Datadog Logs HTTP intake transport for LogLayer. Built on
`transports/http` with a Datadog-specific encoder, site-aware URL, and
`DD-API-KEY` header. Rejects non-https URLs by default; opt-in via
`Config.AllowInsecureURL` for on-prem TLS-terminating proxies.

## Install

```sh
go get go.loglayer.dev/transports/datadog
```

## Documentation

Full reference and examples: <https://go.loglayer.dev/transports/datadog>

The framework itself: <https://go.loglayer.dev>
