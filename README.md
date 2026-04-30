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
  - [Prerequisites](#prerequisites)
  - [One-time setup](#one-time-setup)
  - [Repository layout](#repository-layout)
  - [Common Make targets](#common-make-targets)
  - [Workflow](#workflow)
  - [Versioning and stability](#versioning-and-stability)
  - [Adding a new transport, plugin, or integration](#adding-a-new-transport-plugin-or-integration)
  - [Docs work](#docs-work)
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

The user-facing reference is the [docs site](https://go.loglayer.dev). Architectural context (multi-module split, thread-safety contract, performance log, release process) lives in [AGENTS.md](AGENTS.md). Commit conventions and PR requirements are in [CONTRIBUTING.md](CONTRIBUTING.md). This section is the on-ramp for getting a local dev loop running.

### Prerequisites

| Tool | Why | Install |
|------|-----|---------|
| Go 1.25+ | Main module floor. CI matrix tests against 1.25 and 1.26. | <https://go.dev/dl/> |
| lefthook | Drives the pre-commit, commit-msg, and pre-push git hooks. | `go install github.com/evilmartians/lefthook@latest` |
| staticcheck | Pre-commit lint that mirrors CI. Hook hard-fails without it. | `go install honnef.co/go/tools/cmd/staticcheck@latest` |
| Bun | Runs the conventional-commit linter and builds the docs site. The commit-msg hook hard-fails without `bun` + `node_modules`. | <https://bun.sh/> |
| govulncheck (optional) | Advisory vuln scan; the SessionStart hook surfaces findings as session context. | `go install golang.org/x/vuln/cmd/govulncheck@latest` |

`go install` puts binaries in `$(go env GOPATH)/bin` (default `~/go/bin`). Make sure that directory is on your `PATH` or the git hooks can't find `lefthook` / `staticcheck`. If only `~/.local/bin` is on your `PATH`, symlink:

```sh
ln -sf ~/go/bin/lefthook ~/.local/bin/lefthook
```

Without this, the generated git hook script silently fails open (lefthook intentionally exits 0 when it can't find itself) and your local hooks don't run.

### One-time setup

```sh
git clone https://github.com/loglayer/loglayer-go
cd loglayer-go

make hooks       # wire up git hooks via lefthook
bun install      # install commit-msg lint deps
make build       # warm the module cache for every sub-module
```

### Repository layout

This is a multi-module repo. The framework core lives at the root (`go.loglayer.dev`); every transport, plugin, and integration ships as its own independently-versioned Go module so a breaking change in one module bumps only that module's tag namespace, never the whole repo to /v2.

```
loglayer-go/
├── *.go                Framework core (loglayer / builder / dispatch / level / ...)
├── transport/          BaseTransport / BaseConfig + helpers + transporttest
├── transports/         Built-in transports, one Go module each
│   ├── pretty/         Colorized terminal output
│   ├── structured/     JSON-per-line
│   ├── zerolog/, zap/, charmlog/, logrus/, slog/, ...   Wrappers for popular loggers
│   ├── otellog/        OpenTelemetry log SDK (split module: heavy deps)
│   ├── datadog/, http/, sentry/, lumberjack/            Network / file destinations
│   ├── blank/          Template you can copy when adding a new transport
│   └── testing/        In-memory capture for tests
├── plugins/            Built-in plugins (redact, sampling, oteltrace, ...)
├── integrations/       Higher-level glue (loghttp, sloghandler)
├── examples/           Runnable example apps, one Go module each
├── scripts/            CI / dev helpers (foreach-module.sh, agent-vulncheck.sh, ...)
├── docs/               VitePress docs site (canonical user-facing reference)
├── go.work             Workspace stitching every sub-module together for gopls
├── AGENTS.md           Architectural reference; read before non-trivial work
├── CONTRIBUTING.md     Commit conventions, PR workflow
└── .claude/rules/      Doc / Go / benchmarking conventions
```

`go.work` is committed and stitches every sub-module together for `gopls` and root-level `go test ./...`. You don't need to run `go work sync` after cloning; everything resolves out of the box. Each sub-module's `go.mod` carries a `replace go.loglayer.dev => ../..` directive (and similar replaces for any sibling sub-modules it depends on) so local edits to the core flow into the sub-modules without publishing. release-please rewrites those replaces to real versions at release time, so you don't manage them by hand. CI runs each sub-module in isolation through `scripts/foreach-module.sh` so deps don't leak between modules.

To run an example app:

```sh
cd examples/multi-transport && go run .
```

### Common Make targets

`make help` lists everything. The ones you'll use most:

| Target | What it does |
|--------|--------------|
| `make ci` | Full CI gauntlet: tidy + lint + multi-module race tests. Run before pushing. |
| `make test-race` | Race tests across every module, parallelized across CPUs. Mirrors pre-push. |
| `make test` | Fast main-module-only tests for the inner loop. |
| `make lint` | vet + gofmt-check + staticcheck across every module. |
| `make fmt` | gofmt every Go file in place. |
| `make tidy` | `go mod tidy` every sub-module + diff check. |
| `make staticcheck` / `make vuln` | Run the matching tool across every shipped module. |
| `make bench` | Run the benchmark suite (`bench_test.go` at repo root). |
| `make docs` / `make docs-dev` | Build the VitePress docs / run dev server with live reload. |

Behind the scenes, anything multi-module routes through `scripts/foreach-module.sh <op>`. The `test` op accepts `PARALLEL=N` (defaults to `nproc`); set `PARALLEL=1` to serialize when you need clean per-module output for debugging.

### Workflow

- Branch off `main` as `<type>/<short-slug>` (e.g. `feat/zerolog-id`, `docs/getting-started-fixes`).
- Use [Conventional Commits](https://www.conventionalcommits.org/) with the package as the scope: `feat(transports/zap): add ID field`. The `commit-msg` hook lints with the same parser release-please uses.
- Hooks run on every commit and push. Don't `--no-verify` past failures; fix the underlying issue. The full pre-push (`make test-race`) finishes in well under 10 seconds on a multi-core box.

### Versioning and stability

Each module follows SemVer from `v1.0.0` onward. The framework core (`go.loglayer.dev`) and every transport / plugin / integration are versioned independently, so a breaking change in (say) `transports/zap` bumps that module's tag namespace alone (`transports/zap/v2.0.0`) and doesn't force you off the latest core. `feat:` → minor, `fix:` → patch, `feat!:` or a body containing `BREAKING CHANGE:` → major.

Releases are cut by merging the always-open release-please PR; tags + GitHub Releases happen automatically. Don't `git tag` manually. Full policy: [AGENTS.md → Versioning and Changelog](AGENTS.md).

### Adding a new transport, plugin, or integration

Every transport / plugin / integration ships as its own Go module from day one (no "bundle in main, split later"). `transports/blank/` is a copyable template. The full recipe (go.mod, replace directives, release-please registration, foreach-module updates, docs page) lives in [AGENTS.md → Adding a new transport, plugin, or integration](AGENTS.md). Doc-site conventions for the new page are in [`.claude/rules/documentation.md`](.claude/rules/documentation.md).

### Docs work

```sh
make docs-dev      # live preview, typically at http://localhost:5173
make docs          # production build (CI mirrors this)
```

Source under `docs/src/`. Prose conventions (no em dashes, lead with conclusion, four-page pattern for transports/plugins) are documented in [`.claude/rules/documentation.md`](.claude/rules/documentation.md).

## Issues and questions

Bug reports, feature requests, and architectural questions go in [GitHub Issues](https://github.com/loglayer/loglayer-go/issues).

## License

[MIT](LICENSE)

Made with ❤️ by [Theo Gravity](https://suteki.nu) / [Disaresta](https://disaresta.com). Logo by [Akshaya Madhavan](https://www.linkedin.com/in/akshaya-madhavan).
