# go.loglayer.dev/transports/phuslu

[![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/phuslu/v2.svg)](https://pkg.go.dev/go.loglayer.dev/transports/phuslu/v2)

LogLayer transport that wraps a `*phuslu/log.Logger`. Map metadata becomes individual phuslu fields; struct metadata lands under a configurable key. **Fatal entries always exit the process** since phuslu has no public hook to suppress its `os.Exit` (every other wrapper honors `Config.DisableFatalExit`).

## Install

```sh
go get go.loglayer.dev/transports/phuslu
```

## Documentation

Full reference and examples: <https://go.loglayer.dev/transports/phuslu>

Main library: <https://go.loglayer.dev>
