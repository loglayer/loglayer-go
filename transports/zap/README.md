# go.loglayer.dev/transports/zap

[![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/zap.svg)](https://pkg.go.dev/go.loglayer.dev/transports/zap)

LogLayer transport that wraps a `*zap.Logger`. Map metadata becomes individual zap fields; struct metadata lands under a configurable key. Fatal-level entries are routed through a custom `CheckWriteHook` so loglayer's `DisableFatalExit` is honored.

## Install

```sh
go get go.loglayer.dev/transports/zap
```

## Documentation

Full reference and examples: <https://go.loglayer.dev/transports/zap>

Main library: <https://go.loglayer.dev>
