# LogLayer for Go: Agent Guidelines

> **Note:** Always-applicable rules (documentation, code patterns) live in `.claude/rules/`.
> This file is high-level project context.

## Project Overview

LogLayer for Go is a Go port of the TypeScript loglayer library: a transport-agnostic
structured logging library with a fluent API for messages, fields, metadata, and errors.

**Module path:** `go.loglayer.dev`
**GitHub:** `github.com/loglayer/loglayer-go`
**Docs:** VitePress site under `docs/`

## Project Structure

```
loglayer-go/
├── docs/                       VitePress documentation site
│   ├── .vitepress/             VitePress config (sidebar lives here)
│   └── src/                    Markdown source
│       ├── logging-api/        Per-method API guides
│       ├── transports/         Per-transport guides + _partials/
│       └── ...                 Top-level pages (index, intro, configuration, etc.)
├── transport/                  BaseTransport / BaseConfig
├── transports/                 Built-in transports
│   ├── console/                Plain fmt.Println-style
│   ├── pretty/                 Colorized terminal output (uses fatih/color)
│   ├── structured/             JSON-per-line
│   ├── testing/                In-memory capture for tests
│   ├── zerolog/                Wraps github.com/rs/zerolog
│   ├── zap/                    Wraps go.uber.org/zap
│   ├── phuslu/                 Wraps github.com/phuslu/log (always exits on Fatal)
│   ├── logrus/                 Wraps github.com/sirupsen/logrus
│   └── charmlog/               Wraps github.com/charmbracelet/log
├── loglayer.go                 LogLayer struct + processLog dispatcher
├── builder.go                  LogBuilder fluent chain
├── level.go                    LogLevel + per-level state
├── types.go                    Config, Fields, Data, Metadata aliases
├── fields.go                   WithFields / WithoutFields / mute methods
├── from_context.go             NewContext / FromContext / MustFromContext
├── mock.go                     loglayer.NewMock for silent mocking
└── *_test.go                   Tests + benchmarks
```

## Key Design Decisions

- **Metadata is `any`**: `WithMetadata` accepts any value; the transport decides serialization.
  `loglayer.Metadata` is a type alias for `map[string]any` for the common case.
- **Fields is `map[string]any`**: not `any`, because fields support keyed mutation,
  inheritance, and nesting that only make sense over keys. (Renamed from "Context"
  to avoid clashing with Go's `context.Context`.)
- **Fatal exits by default**: matches Go convention (stdlib `log.Fatal`, zerolog, zap, etc.).
  Set `Config.DisableFatalExit: true` to opt out. `loglayer.NewMock()` does this automatically.
- **No `Logger` interface in core**: Go convention is "consumer defines the interface."
  Application code accepts the concrete `*loglayer.LogLayer`; `loglayer.NewMock()` returns
  the same type for test injection.
- **Multi-module layout**: the main `go.loglayer.dev` module hosts only the framework core (loglayer/builder/dispatch/plugin/level/etc.), the shared `transport/` package (BaseTransport, helpers, transporttest, benchtest), the `utils/*` helpers, and `internal/lltest` (a private capture transport so main's own tests don't have to require `transports/testing`). Every transport, plugin, and integration ships as its own independently-versioned Go module; `monorel.toml`'s `[packages]` map is the canonical list.

  The split is the whole point: a breaking change in any single sub-module bumps only that sub-module's major version, so `go.loglayer.dev` itself never has to migrate to `/v2` because of a downstream rename. Pre-split, every breaking refactor cascaded into a main-module path migration.

  Each sub-module has its own `go.mod` with a `replace go.loglayer.dev => ../..` (and any relevant sibling) directive for development. A `go.work` at the repo root lets `gopls` and `go test ./...` see every module from one place; CI uses `scripts/foreach-module.sh` which runs each module in isolation and is unaffected.

## Verification

After any code change:

```sh
go build ./...
go test ./...
```

For benchmarks:

```sh
go test -bench=. -benchmem -run=^$ ./...
```

For docs:

```sh
cd docs && bun run docs:build
```

For livetests (integration tests against real third-party SDKs):

```sh
# OTel: build-tag-gated, lives in the main module (cheap deps).
go test -tags=livetest -race ./transports/otellog/ ./plugins/oteltrace/

# Datadog dd-trace-go: lives in its own module so dd-trace-go's heavy
# transitive closure doesn't pollute the main module's dependency graph.
cd plugins/datadogtrace/livetest && go test -race ./...
```

Two patterns are in use, picked by dependency weight:

- **Build-tag gating in the main module** (`//go:build livetest`): used
  when the SDK adds only a handful of indirect deps. The OTel SDK fits
  here (~4 added go.sum lines on top of the OTel API we already need).
- **Separate test module**: used when the SDK pulls in a heavy transitive
  closure that we don't want exposed to users of the plugin. The Datadog
  dd-trace-go SDK fits here (would have added 250+ go.sum lines to the
  main module). The test module imports the parent via a `replace`
  directive and is opt-in by `cd`-ing into it.

Add new livetests for any package that talks to a third-party SDK whose
contract you need to verify in-process. Use the build-tag pattern when
the dep cost is small; use a separate module when it isn't.

## Git Hooks (lefthook)

Pre-commit and pre-push hooks are managed by [lefthook](https://github.com/evilmartians/lefthook).
Config lives in `lefthook.yml` at the repo root.

Install once after cloning:

```sh
go install github.com/evilmartians/lefthook@latest
lefthook install
```

`go install` puts binaries in `$(go env GOPATH)/bin` (default `~/go/bin`).
Make sure that directory is on your `PATH` so git hooks can find `lefthook`
when they fire. If you only have `~/.local/bin` or similar on `PATH`, a
symlink works too:

```sh
ln -sf ~/go/bin/lefthook ~/.local/bin/lefthook
```

Without this, the hook script will print `Can't find lefthook in PATH` and
exit 0, silently skipping the checks (lefthook intentionally fails open so a
missing install doesn't break commits).

What runs:

- **commit-msg**: lints the commit message against `@conventional-commits/parser`
  for git-history hygiene. Hard-fails if `bun` isn't on PATH or
  `node_modules` is missing; install bun (https://bun.sh) and run `bun install`.
- **pre-commit** (parallel): `gofmt -l` on staged Go files (fails if anything
  needs formatting; run `gofmt -w <file>` to fix), `go vet ./...`, and
  `staticcheck ./...`. Hard-fails if `staticcheck` isn't on PATH;
  install once with `go install honnef.co/go/tools/cmd/staticcheck@latest`.
- **pre-push**: `go test -race -count=1 ./...` across every module via
  `scripts/foreach-module.sh test`. The script parallelizes per-module
  test runs across CPUs; override with `PARALLEL=1` for serial output.

Hook commands fail closed: missing tooling blocks the commit/push so the
local checks actually run. Skip a single commit/push with `--no-verify` if
you genuinely need to. (Note: this only applies to the steps inside
lefthook.yml. The git hook script lefthook generates still fails open if
lefthook itself isn't installed; the line below explains why and how to
recover.)

Skip a hook for one command with `git commit --no-verify` or
`git push --no-verify`. Don't make this a habit; the hooks are deliberately
fast. The pre-push test step parallelizes the per-module test runs across
CPUs (override with `PARALLEL=1` to force serial when debugging a single
module's output), so a typical run finishes in well under 10 seconds on a
multi-core box.

## Versioning and Changelog

Releases are managed by [monorel](https://monorel.disaresta.com), a changesets-style release tool built specifically for the layout this repo uses (bare `vX.Y.Z` for the root, `<path>/vX.Y.Z` for sub-modules). The release signal is explicit per-PR: `.changeset/<name>.md` files declare which packages release at what bump level. Don't `git tag` manually.

- **Main module** tags as `v1.X.Y`. Sub-modules tag as
  `transports/otellog/v1.X.Y`, `plugins/oteltrace/v1.X.Y` (Go module
  convention). Configured in `monorel.toml` at the repo root.
- Standard SemVer: `:major` bump, `:minor` bump, `:patch` bump per changeset frontmatter. The maximum bump across all changesets naming a package wins.
- Per-package `CHANGELOG.md` is **maintained by `monorel release`** from
  v1.0.0 forward. New entries land above the first `## ` heading in
  Keep-a-Changelog format; existing entries (release-please-formatted,
  for the historical period before the migration) stay verbatim.
- User-facing release notes also land in `docs/src/whats-new.md` for
  the docs site. Currently maintained manually; follow the Keep a
  Changelog shape.
- A changeset can name multiple packages with different bumps. Use
  `monorel add --package "<name>:<level>"` to write one (or hand-roll
  the file). Package keys match the `[packages."<key>"]` lines in
  `monorel.toml`: `go.loglayer.dev` for the root, `<path>` for
  sub-modules (e.g. `transports/zerolog`, `plugins/oteltrace`).
- Conventional commits are still required for PR titles + commit
  messages, but **only as a hygiene check**. `pr-title.yml` and the
  lefthook commit-msg hook validate the format; the parser doesn't
  drive any release decision. A release happens iff a PR contributes
  a `.changeset/*.md` file to main.

### Adding a new transport, plugin, or integration

Every transport, plugin, and integration ships as its own Go module. There is no "bundle it in main first, split later" path — split from day one so the eventual breaking change is local to that module's tag namespace.

To add `<path>` (e.g. `transports/foo` or `plugins/bar`):

1. Create the directory and code as usual.
2. Add `<path>/go.mod` with:

   ```
   module go.loglayer.dev/<path>

   go 1.25.0

   replace go.loglayer.dev => ../..

   require go.loglayer.dev v0.0.0-00010101000000-000000000000
   ```

   Adjust the `replace` depth (`../..` for `transports/foo`, `../../..` for `plugins/foo/livetest`, etc.). If the package depends on other split sub-modules (e.g. `plugins/plugintest`), add corresponding `replace` and `require` lines following the existing siblings as a template.
3. Register the module in `monorel.toml` with a `[packages."<path>"]` block following the existing siblings as a template (`tag_prefix`, `path`, `changelog` all set to the path-derived values).
4. Add the path to `scripts/foreach-module.sh` (`ALL_MODULES`, `SHIPPED_MODULES`, and the `test` op's hardcoded list).
5. Add the path to `go.work`'s `use` block.
6. Run `bash scripts/foreach-module.sh tidy` to settle indirect deps and `bash scripts/foreach-module.sh test` to confirm.
7. Open the PR. **No release happens from this PR** — `monorel.toml` registers the package but registration alone doesn't trigger a release.
8. Cut the first release in a follow-up PR by adding a changeset:

   ```sh
   monorel add --package "<path>:major" --message "Initial release."
   ```

   When that follow-up PR merges, the always-open release PR updates with `<path>/v1.0.0` (or `:minor` → `v0.1.0`, `:patch` → `v0.0.1`); merging that release PR creates the tag.

Tags are created by `monorel release` from changesets scoped to the module. The post-merge tag form is `<path>/v<X.Y.Z>`. CI runs `go test` and the other checks per-module via `scripts/foreach-module.sh`. Adding the new module to that script's arrays in step 4 is what makes CI / pre-push pick it up.

The release-please-era gotchas (`release-as` initial-version dance, squash-merge stripping `Release-As:` footers, `exclude-paths` completeness) are gone. Changesets are explicit per-PR file artifacts; nothing is inferred from commit messages, so none of those failure modes can recur.

## CI / Release Workflows

`.github/workflows/`:

- **ci.yml**: build, gofmt, vet, test -race, staticcheck, govulncheck.
  Matrix tests Go 1.25 and 1.26. Calls `scripts/foreach-module.sh`
  for the per-module operations so the same checks run locally.
- **docs.yml**: build vitepress docs on PR (verify clean), deploy to
  GitHub Pages on main push, also called by `release.yml` after a
  monorel-driven release (workflow_call sidesteps GitHub's anti-recursion
  for `GITHUB_TOKEN`-created releases).
- **release-pr.yml**: triggered on push to `main`. Runs `monorel preview --upsert`
  to maintain the always-open release PR.
- **release.yml**: triggered on push to `main` for the `chore(release):`
  merge commit (or via `workflow_dispatch`). Runs the release pipeline:
  `monorel release` → `git push --follow-tags` → `monorel publish`.
  Then calls `docs.yml` via `workflow_call`.
- **pr-title.yml**: validates that PR titles follow Conventional Commits
  for git-history hygiene. Allowed types match the scoped-commit
  convention above.

To cut a release:

1. Land changes on `main` via PRs that include a `.changeset/<name>.md`
   file when a release is desired. Use `monorel add --package "<name>:<level>"`
   to author one, or hand-roll the file.
2. The release-pr workflow updates the always-open release PR after each
   push to `main`. The PR body shows the rendered plan (per-package
   `from`/`to` versions plus the changelog body for each).
3. Edit `docs/src/whats-new.md` to add the user-facing summary if
   relevant — monorel doesn't touch this file.
4. Merge the release PR. The release.yml workflow runs the pipeline:
   writes per-package CHANGELOG entries, deletes the consumed
   `.changeset/*.md` files, creates per-package tags, pushes, and
   creates one GitHub Release per tag.

## Vulnerability scanning (advisory, not gating)

`scripts/agent-vulncheck.sh` runs `govulncheck` across all modules and
emits a compact summary of findings. It's wired up as a Claude Code
`SessionStart` hook in `.claude/settings.json`, so any agent working
in the repo sees current findings as session context. CI runs the same
scan as a separate job (`govulncheck` in `ci.yml`).

Findings come in three flavors and only some are addressable here:

1. **Standard library vulns** (`crypto/x509@go1.25` → `go1.25.9`):
   fixed by the operator upgrading their Go install. The repo can bump
   the `go` directive in go.mod to require a patched version, which
   forces downstream users onto it. Worth doing when patches accumulate
   but trades adoption for urgency. Don't bump on every advisory.
2. **Direct dependency vulns**: usually fixable with `go get -u
   <module>` followed by `go mod tidy`. Pre-push tests catch
   regressions.
3. **Indirect / unreachable vulns**: govulncheck reports these as
   "imports" but doesn't show a call path. Typically false positives
   for our use case; ignore unless they migrate to the "Your code is
   affected" section on a future scan.

The hook never blocks. If a finding warrants action, an agent or
human surfaces it in a PR; otherwise the noise floor stays advisory.

## Thread Safety

Every method on `*LogLayer` is safe to call from any goroutine, including
concurrently with emission. There is no setup-only category.

How each class achieves safety:

- **Emission methods** (`Info`, `Warn`, `Error`, `Debug`, `Fatal`,
  `WithMetadata`, `WithError`, `WithContext`, `Raw`, `MetadataOnly`, `ErrorOnly`):
  read-only on logger state.
- **Returns-new** (`WithFields`, `WithoutFields`, `Child`, `WithPrefix`,
  `WithGroup` on `*LogLayer`, `WithContext` on `*LogLayer`): build a new
  logger; receiver untouched.
- **Read-only** (`GetFields`, `GetLoggerInstance`, `IsLevelEnabled`): no state
  change.
- **Level mutators** (`SetLevel`, `EnableLevel`, `DisableLevel`,
  `EnableLogging`, `DisableLogging`): backed by an `atomic.Uint32` bitmap.
  Mirrors `zap.AtomicLevel`. Designed for live runtime toggling (SIGUSR1,
  admin endpoints flipping debug on, etc.).
- **Transport mutators** (`AddTransport`, `RemoveTransport`,
  `SetTransports`): publish a new immutable transport set via
  `atomic.Pointer[transportSet]`. Concurrent mutators on the same logger
  serialize via an internal mutex (slow path); the dispatch hot path only
  loads the pointer.
- **Mute toggles** (`MuteFields`, `UnmuteFields`, `MuteMetadata`,
  `UnmuteMetadata`): backed by `atomic.Bool` state on `*LogLayer`. Construction
  values come from `Config.MuteFields` / `Config.MuteMetadata` and are latched
  into the atomic state in `build()`.
- **Plugin mutators** (`AddPlugin`, `RemovePlugin`): publish a new
  immutable `pluginSet` via `atomic.Pointer[pluginSet]`. Same pattern as
  transports; serialized by `pluginMu`. The dispatch hot path only loads.
- **Group mutators** (`AddGroup`, `RemoveGroup`, `EnableGroup`,
  `DisableGroup`, `SetGroupLevel`, `SetActiveGroups`,
  `ClearActiveGroups`): publish a new immutable `groupSet` via
  `atomic.Pointer[groupSet]`. Same pattern, serialized by `groupMu`.

The contract above is verified by `concurrency_test.go` under `-race`,
including a runtime-level-toggle test and a transport hot-swap test.

## Performance: Attempted and Rejected

This section records architectural perf changes that were tried, measured, and
reverted. Don't redo them without new evidence.

### sync.Pool for LogBuilder (rejected)

**Hypothesis:** pooling `*LogBuilder` would eliminate one allocation per
`WithMetadata` / `WithError` call.

**Result:** net negative. Map metadata went from 200 ns / 3 allocs to 217 ns /
3 allocs, struct metadata 75 ns / 2 allocs to 87 ns / 2 allocs.

**Why:** the Go compiler already inlines `newLogBuilder` (cost 5) into the
caller, and escape analysis stack-allocates the resulting `*LogBuilder` when
the chain is consumed inline. There was no allocation to save. `sync.Pool.Get`
plus the `defer pool.Put` added overhead with zero alloc savings.

**Don't try again unless** Go's compiler regresses on inlining or the LogBuilder
shape grows beyond what fits a stack frame. Verify by running:

```sh
go build -gcflags='-m' . | grep -E "newLogBuilder|LogBuilder.*moved to heap"
```

The builder should NOT appear in the "moved to heap" output for a
`log.WithMetadata(...).Info(...)` call site.

### Pool the Data map (rejected, never shipped)

**Hypothesis:** pooling the assembled `Data` map saves an alloc per log with
fields or error.

**Result:** would force a contract change. Transports today are free to retain
`params.Data` (the testing transport stores it in `LogLine.Data` without
copying; future async/batching transports would do the same). Pooling means
"transports must not retain or must clone first," which silently breaks every
custom transport.

**Don't try again unless** there's a clean contract migration path (e.g. a
`params.RetainData()` helper transports must call before they retain).

### singleTransport cached field (removed)

A cached pointer to `transports[0]` for the single-transport fast path. Saved
the loop-iteration cost (~2 ns) but had to be re-synced across `New`, `Child`,
`AddTransport`, `RemoveTransport`, `SetTransports`. Marginal speedup
didn't justify the maintenance burden.

**Don't reintroduce** unless single-transport dispatch becomes a measured
bottleneck.

### Verified wins (kept)

- **Level state as `[6]bool` array** instead of `map[LogLevel]bool`: ~10%
  faster on `BenchmarkLoglayer_SimpleMessage` (42 → 38 ns) and improves
  thread-safety story.
- **Lazy `Data` allocation in `processLog`**: skip the `make(Data, ...)` when
  there are no fields and no error. SimpleMessage went from 2 allocs to 1.
- **Sized `Data` map** (`make(Data, len(fields)+1)` instead of a fixed `2`):
  trivial, avoids map-grow on fields-heavy logs.
- **`cfg := &l.config` instead of value copy** in `processLog`: avoids copying
  the entire Config struct (multiple fields incl. function pointers) on every
  log call. ~5% faster on SimpleMessage (38 → 36 ns) and proportional wins on
  metadata paths.

## Currently Out of Scope

These exist in upstream loglayer but are not in the Go v1:

- Field managers (TS calls these "context managers": Linked / Isolated / Default. Fields behave as a flat copied map here)
- Log level managers (LinkedLogLevelManager / etc.; level state is per-instance, copied on `Child()`)
- Async lazy evaluation (the sync form ships as `loglayer.Lazy`; async has no Go equivalent for `Promise<T>`)
- Mixins (the `useLogLayerMixin` augmentation pattern)

If you're asked to add one of these, propose the design first, do not silently start implementing.
