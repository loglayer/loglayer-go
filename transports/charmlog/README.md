# go.loglayer.dev/transports/charmlog

[![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/charmlog/v2.svg)](https://pkg.go.dev/go.loglayer.dev/transports/charmlog/v2)

LogLayer transport that wraps a `*charmbracelet/log.Logger`. Map metadata becomes individual key/value attrs; struct metadata lands under a configurable key. The package name is `charmlog` to avoid colliding with the stdlib `log`.

## Install

```sh
go get go.loglayer.dev/transports/charmlog
```

## Documentation

Full reference and examples: <https://go.loglayer.dev/transports/charmlog>

Main library: <https://go.loglayer.dev>
