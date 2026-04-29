# go.loglayer.dev/transports/http

[![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/http.svg)](https://pkg.go.dev/go.loglayer.dev/transports/http)

Generic batched HTTP POST transport for LogLayer. Pluggable encoder, async worker, configurable batching and retries. Default `Client` refuses cross-host redirects so an `Authorization` header (or any other credential you set on the request) can't leak to a redirected host. Use it directly to talk to any log-ingestion API; the Datadog transport is built on top of it.

## Install

```sh
go get go.loglayer.dev/transports/http
```

## Documentation

Full reference and examples: <https://go.loglayer.dev/transports/http>

Main library: <https://go.loglayer.dev>
