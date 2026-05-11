# Better Stack transport for loglayer

Send structured logs to [Better Stack](https://betterstack.com) from your Go applications.

## Installation

```bash
go get go.loglayer.dev/transports/betterstack
```

## Quick start

```go
package main

import (
    "log"

    "go.loglayer.dev/v2"
    bs "go.loglayer.dev/transports/betterstack"
)

func main() {
    logger := loglayer.New(loglayer.Config{
        Transport: bs.New(bs.Config{
            SourceToken: "<your-source-token>",
        }),
    })

    logger.Info("Application started")
}
```

## Configuration

```go
bs.New(bs.Config{
    SourceToken:     "<your-source-token>",  // Required
    URL:             "https://in.logs.betterstack.com",  // Optional, defaults to Better Stack's default endpoint
    TimestampField:  "dt",                   // Optional, custom timestamp field name
})
```

## Payload format

Each log entry is sent as a JSON object with:

- `message`: The log message
- `level`: The log level (trace, debug, info, warn, error, fatal, panic)
- `dt`: ISO 8601 timestamp (configurable via TimestampField)
- All fields and metadata you add via `WithFields()` and `WithMetadata()`

## Examples

See [example_test.go](https://github.com/loglayer/loglayer-go/blob/main/transports/betterstack/example_test.go) for more examples.
