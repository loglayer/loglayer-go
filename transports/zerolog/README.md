# go.loglayer.dev/transports/zerolog

[![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/zerolog/v2.svg)](https://pkg.go.dev/go.loglayer.dev/transports/zerolog/v2)

LogLayer transport that wraps a `*zerolog.Logger`. Map metadata becomes individual zerolog fields; struct metadata lands under a configurable key. Fatal-level entries skip zerolog's default `os.Exit` so loglayer's `DisableFatalExit` is honored.

## Install

```sh
go get go.loglayer.dev/transports/zerolog
```

## Documentation

Full reference and examples: <https://go.loglayer.dev/transports/zerolog>

Main library: <https://go.loglayer.dev>
