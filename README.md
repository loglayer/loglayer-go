<p align="center">
  <a href="https://go.loglayer.dev" title="LogLayer for Go">
    <img src="docs/src/public/images/loglayer.png" alt="LogLayer logo by Akshaya Madhavan" width="200">
  </a>
</p>

# LogLayer for Go

<p align="center">
  <a href="https://github.com/loglayer/loglayer-go/releases"><img src="https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=v*&sort=semver&label=version&style=flat-square" alt="Latest version"></a>
  <a href="https://pkg.go.dev/go.loglayer.dev"><img src="https://pkg.go.dev/badge/go.loglayer.dev.svg" alt="Go Reference"></a>
  <a href="https://github.com/loglayer/loglayer-go/actions/workflows/ci.yml"><img src="https://github.com/loglayer/loglayer-go/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License: MIT"></a>
</p>

`loglayer-go` is a unified logger that routes logs to various logging libraries, cloud providers, files, terminals, and OpenTelemetry while providing a fluent API for specifying log messages, fields, metadata, and errors.

Requires **Go 1.25+** for the main module.

For full documentation, read the [docs](https://go.loglayer.dev).

```go
// Example using the Pretty terminal transport.
// You can start out with one transport and swap to another later
// without touching application code.
import (
    "errors"

    "go.loglayer.dev"
    "go.loglayer.dev/transports/pretty"
)

log := loglayer.New(loglayer.Config{
    Transport: pretty.New(pretty.Config{}),
    // Put fields under a specific key (default is flattened)
    FieldsKey: "context",
})

// Persistent fields that appear on every subsequent log
log = log.WithFields(loglayer.Fields{
    "path":  "/",
    "reqId": "1234",
})

log.WithPrefix("[my-app]").
    WithError(errors.New("test")).
    // Data attached to this log entry only
    WithMetadata(loglayer.Metadata{"some": "data"}).
    Info("my message")
```

```json
{
  "level": "info",
  "time": "2026-04-26T12:00:00Z",
  "msg": "[my-app] my message",
  "context": {
    "path": "/",
    "reqId": "1234"
  },
  "some": "data",
  "err": {
    "message": "test"
  }
}
```

## Install

```sh
go get go.loglayer.dev
```

## Documentation

For detailed documentation, visit [go.loglayer.dev](https://go.loglayer.dev).

## TypeScript counterpart

Coming from [loglayer for TypeScript](https://loglayer.dev)? See [For TypeScript Developers](https://go.loglayer.dev/for-typescript-developers) for the API mapping and Go-specific differences.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for setup, commit conventions, and PR requirements. Architecture context lives in [AGENTS.md](AGENTS.md).

## License

[MIT](LICENSE)

Made with ❤️ by [Theo Gravity](https://suteki.nu) / [Disaresta](https://disaresta.com). Logo by [Akshaya Madhavan](https://www.linkedin.com/in/akshaya-madhavan).
