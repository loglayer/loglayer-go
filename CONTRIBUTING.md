# Contributing

Thanks for considering a contribution. The essentials:

## Setup

```sh
git clone https://github.com/loglayer/loglayer-go
cd loglayer-go
go install github.com/evilmartians/lefthook@latest
lefthook install   # wires up pre-commit and pre-push hooks
```

`go install` puts binaries in `$(go env GOPATH)/bin` (default `~/go/bin`). If
that's not on your `PATH`, symlink the binary somewhere that is, otherwise
the git hooks will silently skip:

```sh
ln -sf ~/go/bin/lefthook ~/.local/bin/lefthook
```

## Commits

Use [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/)
with the package as the scope. The description should be lowercased.

Allowed types: `feat`, `fix`, `perf`, `revert`, `refactor`, `deps`, `docs`,
`chore`, `style`, `test`, `build`, `ci`. Examples:

- `feat(transports/datadog): add eu1 site support`
- `fix(integrations/loghttp): preserve trailers when wrapping ResponseWriter`
- `docs: clarify Fields vs Metadata`

The `commit-msg` git hook lints every message with the same parser
release-please uses (`@conventional-commits/parser`). CI re-runs the
same check on every PR and on push to main. Run `bun install` once
after cloning to enable the local hook; without it the hook skips and
CI is the only safety net.

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
3. Add an entry to `CHANGELOG.md` under `## [Unreleased]` in the right
   component section.
4. If a new transport: update `docs/src/transports/_partials/transport-list.md`
   and the sidebar in `docs/.vitepress/config.ts`.

Verify docs build locally:

```sh
make docs
```

The codebase is designed to work well with AI coding assistants. The
[`docs/src/llms.md`](docs/src/llms.md) page documents the `/llms.txt` and
`/llms-full.txt` references we ship for that purpose.

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
