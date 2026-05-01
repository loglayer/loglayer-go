# Contributing

Thanks for considering a contribution. The essentials:

## Setup

### Prerequisites

| Tool | Why | Install |
|------|-----|---------|
| Go 1.25+ | Main module floor. CI matrix tests against 1.25 and 1.26. | <https://go.dev/dl/> |
| lefthook | Drives the pre-commit, commit-msg, and pre-push git hooks. | `go install github.com/evilmartians/lefthook@latest` |
| staticcheck | Pre-commit lint that mirrors CI. Hook hard-fails without it. | `go install honnef.co/go/tools/cmd/staticcheck@latest` |
| Bun | Runs the conventional-commit linter and builds the docs site. The commit-msg hook hard-fails without `bun` + `node_modules`. | <https://bun.sh/> |
| govulncheck (optional) | Advisory vuln scan; the SessionStart hook surfaces findings as session context. | `go install golang.org/x/vuln/cmd/govulncheck@latest` |

`go install` puts binaries in `$(go env GOPATH)/bin` (default `~/go/bin`). Make sure that directory is on your `PATH`, otherwise the git hooks silently skip when they can't find `lefthook` / `staticcheck`. If only `~/.local/bin` is on your `PATH`, symlink:

```sh
ln -sf ~/go/bin/lefthook ~/.local/bin/lefthook
```

### Clone and wire up hooks

```sh
git clone https://github.com/loglayer/loglayer-go
cd loglayer-go

make hooks       # wire up git hooks via lefthook
bun install      # install commit-msg lint deps
make build       # warm the module cache for every sub-module
```

Run `make help` for the full list of targets. The ones you'll use most: `make ci` (full pre-push gauntlet), `make test-race` (multi-module race tests), `make test` (fast main-module-only inner loop), `make lint` (vet + gofmt-check + staticcheck), `make fmt` (apply gofmt), `make docs-dev` (live-reload docs preview).

## Commits

Use [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/)
with the package as the scope. The description should be lowercased.

Allowed types: `feat`, `fix`, `perf`, `revert`, `refactor`, `deps`, `docs`,
`chore`, `style`, `test`, `build`, `ci`. Examples:

- `feat(transports/datadog): add eu1 site support`
- `fix(integrations/loghttp): preserve trailers when wrapping ResponseWriter`
- `docs: clarify Fields vs Metadata`

The `commit-msg` git hook lints every message with
`@conventional-commits/parser` for git-history hygiene. CI re-runs
the same check on every PR and on push to main. Run `bun install`
once after cloning to enable the local hook; without it the hook
skips and CI is the only safety net. Note: releases are driven by
`.changeset/*.md` files, not commit messages — the lint catches
malformed messages without affecting whether a release fires.

## Tests

Add tests for any code change. Run them with:

```sh
make test          # go test ./...
make test-race     # go test -race ./... (what CI runs)
```

Use [`internal/transporttest`](internal/transporttest) helpers (`ParseJSONLine`,
`MessageContains`) where they fit; reach for new helpers there if a pattern
shows up across 3+ test files.

## Lint and Format

```sh
make fmt           # apply gofmt
make lint          # gofmt-check + go vet
make ci            # lint + race tests (mirror CI locally before pushing)
```

The pre-commit hook runs `make fmt-check` + `vet` automatically; pre-push
runs the race tests. Don't `--no-verify` past failures; fix the underlying
issue.

## Docs

User-visible changes need:

1. Update the relevant page under `docs/src/` (per-method, per-transport,
   configuration, cheatsheet).
2. Add an entry to `docs/src/whats-new.md` under today's date.
3. Add a changeset describing the change — see [Releases](#releases) below.
   The per-package `CHANGELOG.md` entry is written automatically by
   `monorel release`; don't hand-edit it.
4. If a new transport: update `docs/src/transports/_partials/transport-list.md`
   and the sidebar in `docs/.vitepress/config.ts`.

Verify docs build locally:

```sh
make docs
```

The codebase is designed to work well with AI coding assistants. The
[`docs/src/llms.md`](docs/src/llms.md) page documents the `/llms.txt` and
`/llms-full.txt` references we ship for that purpose.

## Releases

Releases are managed by [monorel](https://monorel.disaresta.com). The
release signal is explicit: every release-affecting PR includes a
`.changeset/<name>.md` file declaring which packages release at what
bump level. The framework core (`go.loglayer.dev`) and each
transport / plugin / integration version independently.

### When to add a changeset

- **Add one** for any PR that should produce a new tag for at least one
  package: features, fixes, performance improvements that change
  observable behavior, doc fixes that correct a documented API,
  dependency bumps that affect consumers.
- **Skip the changeset** for pure-internal changes: refactors with no
  behavior change, test-only updates, doc-typo fixes, CI / tooling
  tweaks. The PR still needs conventional-commit hygiene for the
  history; it just doesn't trigger a release.

### Authoring a changeset

The interactive flow:

```sh
monorel add
```

Or non-interactively:

```sh
monorel add \
  --package "transports/zerolog:minor" \
  --message "Adds Lazy() helper for deferred field evaluation."
```

A single changeset can name multiple packages with different bump
levels:

```sh
monorel add \
  --package "transports/zerolog:major" \
  --package "go.loglayer.dev:patch" \
  --message "Reshape the zerolog Config; pass-through fix in the root."
```

This writes `.changeset/<two-word-name>.md` like:

```markdown
---
"transports/zerolog": major
"go.loglayer.dev": patch
---

Reshape the zerolog Config; pass-through fix in the root.
```

The frontmatter keys are `monorel.toml` package names: `go.loglayer.dev`
for the root, `<path>` for sub-modules (e.g. `transports/zerolog`,
`plugins/oteltrace`, `integrations/loghttp`). Bump levels are `major`,
`minor`, or `patch`. The body becomes the rendered changelog entry for
every package the changeset names.

Hand-writing the file directly works too; the `monorel add` command is
just a generator.

### What happens after merge

1. On every push to `main`, the `release-pr` workflow updates the
   always-open release PR with the cumulative plan across all pending
   changesets.
2. The release PR's body shows the proposed `from` → `to` versions per
   package plus the rendered changelog body for each.
3. When a maintainer merges the release PR, the `release` workflow
   writes per-package `CHANGELOG.md` entries, deletes the consumed
   `.changeset/*.md` files, creates per-package tags, pushes, and
   creates one GitHub Release per tag.

Don't `git tag` manually. The full policy lives in
[AGENTS.md → Versioning and Changelog](AGENTS.md).

### Common pitfalls

- **Forgetting the changeset.** Reviewers should treat "release-affecting
  PR with no changeset" the way they used to treat "release-affecting
  PR with the wrong commit-type prefix": ask the author to add one.
- **Wrong package key.** Frontmatter keys must match `monorel.toml`
  exactly. `transports/zerolog` is right; `go.loglayer.dev/transports/zerolog`
  (the import path) is not. `monorel plan` errors loudly on unknown keys.
- **Multiple PRs touching the same package without coordinating bumps.**
  monorel takes the **maximum** bump across all changesets naming a
  package, so two `:patch` changesets and one `:minor` produce a
  single `:minor` release. That's usually what you want; just be aware.

## Performance Changes

If a change is a performance improvement, follow [`.claude/rules/benchmarking.md`](.claude/rules/benchmarking.md):
capture baseline + after numbers with `benchstat -count=10`. Rejected
attempts go in `AGENTS.md` under "Performance: Attempted and Rejected" so
they don't get re-attempted.

## Architecture Context

Deeper context (project structure, design decisions, thread-safety contract,
versioning policy, transport-split policy) lives in [`AGENTS.md`](AGENTS.md).
Project-wide style rules live under [`.claude/rules/`](.claude/rules/).

## Pull Requests

PRs must:

- Pass CI (build, vet, race tests).
- Include tests for code changes.
- Update relevant docs.
- Use conventional commit messages.

Made with ❤️ by contributors. Logo by [Akshaya Madhavan](https://www.linkedin.com/in/akshaya-madhavan).
