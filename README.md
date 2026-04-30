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

## Install

```sh
go get go.loglayer.dev
```

## Documentation

For detailed documentation, visit [go.loglayer.dev](https://go.loglayer.dev).

## TypeScript counterpart

Coming from [loglayer for TypeScript](https://loglayer.dev)? See [For TypeScript Developers](https://go.loglayer.dev/for-typescript-developers) for the API mapping and Go-specific differences.

## Local development setup

You need:

| Tool | Why | Install |
|------|-----|---------|
| **Go 1.25+** | The main module's floor (`go.mod`). Every sub-module's CI matrix tests against it. | <https://go.dev/dl/> |
| **lefthook** | Manages the pre-commit, commit-msg, and pre-push git hooks (formatting, vet, conventional-commit lint, race tests). | `go install github.com/evilmartians/lefthook@latest` |
| **staticcheck** | Pre-commit lint that mirrors what CI runs. Hook fails open if not installed; CI catches anything missed. | `go install honnef.co/go/tools/cmd/staticcheck@latest` |
| **govulncheck** | Advisory vuln scan (CI runs the same one). Optional; the agent-vulncheck hook prints a hint if it isn't installed. | `go install golang.org/x/vuln/cmd/govulncheck@latest` |
| **Bun** | Runs `scripts/lint-commit.mjs` (the conventional-commit parser check used by the commit-msg hook) and builds the VitePress docs site under `docs/`. Without bun, the local hook skips and CI catches it. | <https://bun.sh/> |

After cloning:

```sh
git clone https://github.com/loglayer/loglayer-go
cd loglayer-go

# Wire up the git hooks (one-time per clone).
lefthook install

# Install the deps used by the commit-msg hook.
bun install

# Build everything once to fetch sub-module deps.
go build ./...
```

Make sure `$(go env GOPATH)/bin` (default `~/go/bin`) is on your `PATH` so the hooks can find `lefthook` / `staticcheck` / `govulncheck`. If only `~/.local/bin` is on `PATH`, symlink:

```sh
ln -sf ~/go/bin/lefthook ~/.local/bin/lefthook
```

Without the symlink (or `PATH` entry), lefthook silently fails open and the hooks won't run.

For docs work:

```sh
cd docs
bun install
bun run docs:build      # validate
bun run docs:dev        # local preview
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for setup, commit conventions, and PR requirements. Architecture context lives in [AGENTS.md](AGENTS.md).

## License

[MIT](LICENSE)

Made with ❤️ by [Theo Gravity](https://suteki.nu) / [Disaresta](https://disaresta.com). Logo by [Akshaya Madhavan](https://www.linkedin.com/in/akshaya-madhavan).
