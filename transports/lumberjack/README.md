# go.loglayer.dev/transports/lumberjack

[![Go Reference](https://pkg.go.dev/badge/go.loglayer.dev/transports/lumberjack/v2.svg)](https://pkg.go.dev/go.loglayer.dev/transports/lumberjack/v2)

File transport for LogLayer with built-in rotation. Writes one JSON object per log entry; rotation is delegated to [lumberjack.v2](https://github.com/natefinch/lumberjack) (size-triggered rollover, backup retention, age-based cleanup, optional gzip compression). On-disk format matches `transports/structured`.

The package name shadows the upstream `gopkg.in/natefinch/lumberjack.v2`. Use an import alias if you import both:

```go
import (
    lltransport "go.loglayer.dev/transports/lumberjack"
    "gopkg.in/natefinch/lumberjack.v2"
)
```

## Install

```sh
go get go.loglayer.dev/transports/lumberjack
```

## Documentation

Full reference and examples: <https://go.loglayer.dev/transports/lumberjack>

Main library: <https://go.loglayer.dev>
