# go.loglayer.dev/transports/sentry

[![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/sentry.svg)](https://pkg.go.dev/go.loglayer.dev/transports/sentry)

LogLayer transport that forwards entries to a caller-supplied `sentry.Logger` (Sentry's structured-logs API). The user owns Sentry initialization; this transport just hands each entry off via the chain-builder API. Fatal-level entries skip Sentry's `Fatal()` / `Panic()` (which would terminate the process) so loglayer's `DisableFatalExit` is honored.

## Install

```sh
go get go.loglayer.dev/transports/sentry
```

## Documentation

Full reference and examples: <https://go.loglayer.dev/transports/sentry>

Main library: <https://go.loglayer.dev>
