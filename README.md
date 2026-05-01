<p align="center">
  <a href="https://go.loglayer.dev" title="LogLayer for Go">
    <img src="docs/src/public/images/loglayer.png" alt="LogLayer logo by Akshaya Madhavan" width="200">
  </a>
</p>

# LogLayer for Go

<p align="center">
  <a href="https://github.com/loglayer/loglayer-go/releases"><img src="https://img.shields.io/github/v/tag/loglayer/loglayer-go?filter=v*&sort=date&label=version&style=flat-square&color=blue" alt="Latest version"></a>
  <a href="https://pkg.go.dev/go.loglayer.dev"><img src="https://pkg.go.dev/badge/go.loglayer.dev.svg" alt="Go Reference"></a>
  <a href="https://github.com/loglayer/loglayer-go/actions/workflows/ci.yml"><img src="https://github.com/loglayer/loglayer-go/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License: MIT"></a>
</p>

`loglayer-go` is a unified logger that routes logs to various logging libraries, cloud providers, files, terminals, and OpenTelemetry while providing a fluent API for specifying log messages, fields, metadata, and errors.

Requires **Go 1.25+** for the main module.

For full documentation, read the [docs](https://go.loglayer.dev).

```go
// Example using the Structured (JSON) transport.
// You can start out with one transport and swap to another later
// without touching application code.
import (
    "errors"

    "go.loglayer.dev"
    "go.loglayer.dev/transports/structured"
)

log := loglayer.New(loglayer.Config{
    Transport: structured.New(structured.Config{}),
    // Put fields under "context" and metadata under "metadata"
    // (defaults are flattened to the root).
    FieldsKey:         "context",
    MetadataFieldName: "metadata",
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
  "metadata": {
    "some": "data"
  },
  "err": {
    "message": "test"
  }
}
```

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
## Table of contents

- [Install](#install)
- [Documentation](#documentation)
- [TypeScript counterpart](#typescript-counterpart)
- [Contributing](#contributing)
- [Issues and questions](#issues-and-questions)
- [License](#license)

<!-- END doctoc -->

## Install

```sh
go get go.loglayer.dev
```

## Documentation

For detailed documentation, visit [go.loglayer.dev](https://go.loglayer.dev).

## TypeScript counterpart

Coming from [loglayer for TypeScript](https://loglayer.dev)? See [For TypeScript Developers](https://go.loglayer.dev/for-typescript-developers) for the API mapping and Go-specific differences.

## Contributing

This is a multi-module repo: the framework core lives at the root (`go.loglayer.dev`); every transport, plugin, and integration ships as its own independently-versioned Go module under `transports/`, `plugins/`, and `integrations/`.

- **Dev-loop on-ramp** (prerequisites, hooks, make targets, commits, tests, docs, releases via [monorel](https://monorel.disaresta.com)): [CONTRIBUTING.md](CONTRIBUTING.md).
- **Architectural context** (multi-module split, thread-safety contract, performance log, release flow internals): [AGENTS.md](AGENTS.md).

`transports/blank/` is the copyable template for adding a new transport, plugin, or integration; the full recipe is in [AGENTS.md → Adding a new transport, plugin, or integration](AGENTS.md).

## Issues and questions

Bug reports, feature requests, and architectural questions go in [GitHub Issues](https://github.com/loglayer/loglayer-go/issues).

## License

[MIT](LICENSE)

Made with ❤️ by [Theo Gravity](https://suteki.nu) / [Disaresta](https://disaresta.com). Logo by [Akshaya Madhavan](https://www.linkedin.com/in/akshaya-madhavan).
