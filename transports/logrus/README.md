# go.loglayer.dev/transports/logrus

[![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/logrus.svg)](https://pkg.go.dev/go.loglayer.dev/transports/logrus)

LogLayer transport that wraps a `*logrus.Logger`. Map metadata becomes individual logrus fields; struct metadata lands under a configurable key. Fatal-level entries route through `Log()` so loglayer's `DisableFatalExit` is honored.

## Install

```sh
go get go.loglayer.dev/transports/logrus
```

## Documentation

Full reference and examples: <https://go.loglayer.dev/transports/logrus>

Main library: <https://go.loglayer.dev>
