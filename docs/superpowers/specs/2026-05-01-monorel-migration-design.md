# Migrate loglayer-go from release-please to monorel

**Date:** 2026-05-01
**Scope:** Replace release-please with monorel as loglayer-go's release tool. Full rewrite of release-tooling configuration, workflows, lefthook hooks, scripts, and AGENTS.md release-related prose. Single PR.

## Context

loglayer-go currently uses [release-please](https://github.com/googleapis/release-please) to manage versions, tags, changelogs, and the release PR for 26 Go modules (one main module at the repo root, 25 sub-modules under `transports/`, `plugins/`, `integrations/`).

The release-please saga on this repo (squash-merge stripping `Release-As:` footers, full-history scans leaking footers into new packages, `exclude-paths` not catching footer-based attribution, `bootstrap-sha` not applying mid-life, manual tag/manifest interventions) prompted the creation of [monorel](https://monorel.disaresta.com), a changesets-style Go-native release tool. monorel shipped v0.1.0 and is now self-hosted on its own repo. loglayer-go is the first external consumer.

## Goals

- Cut over the release flow with no loss of tag history, no rewrite of historical CHANGELOG entries, and no manual tag/manifest interventions.
- Replace every release-please-coupled file in the tree (config, manifest, workflows, scripts, lefthook hooks).
- Rewrite AGENTS.md release-related prose so future agent sessions don't drift back to release-please semantics.
- Fix any monorel gap that surfaces during the migration. monorel exists to solve loglayer-go's problems; gaps get fixed in monorel.

## Non-goals

- Rewriting historical CHANGELOG entries from release-please format to Keep-a-Changelog. Hard cut: existing entries stay verbatim, new entries use Keep-a-Changelog format above them.
- Migrating the tag scheme. Bare `vX.Y.Z` for the root module and `<path>/vX.Y.Z` for sub-modules stays.
- Rewriting historical commits. Squash and force-push are out of scope.
- Pushing the migration PR through to a real release in this PR. The first real release after merge happens in a separate follow-up PR with a single test changeset.

## Audit: monorel covers loglayer-go's needs

Confirmed every release-please feature loglayer-go uses has a monorel equivalent:

| Concern | Today (release-please) | After (monorel) |
|---------|------------------------|-----------------|
| Bare `vX.Y.Z` for root module | `include-component-in-tag: false` per-package override | `tag_prefix = ""` per-package |
| `<path>/vX.Y.Z` for sub-modules | `include-component-in-tag: true` + `tag-separator: "/"` | `tag_prefix = "<path>"` per-package |
| Per-sub-module CHANGELOG.md | `changelog-path` per-package | `changelog` per-package |
| `exclude-paths` for the root | Required (commits to `.claude/`, `.github/`, `docs/`, etc. shouldn't bump root) | Not needed: changesets are the only release signal |
| `release-as: "1.0.0"` for new packages | One-shot override per package, must remove later | Not needed: monorel reads existing tags. New packages with no tag history use initial-release rules (major→1.0.0, minor→0.1.0, patch→0.0.1) |
| Bot anti-recursion for docs deploy | `release-please.yml` calls `docs.yml` via `workflow_call` | `release.yml` calls `docs.yml` via `workflow_call` (same mechanism) |
| Always-open release PR | release-please-action's permanent `release-please--branches--main` | `monorel preview --upsert` orchestrator |
| Pre-release windows (rc / beta) | `prerelease: true` in config + commit footers | `monorel pre enter <channel>` / `pre exit` (not used by this migration; available if needed later) |
| Conventional-commit linting | `pr-title.yml` + `commit-lint.yml` use `@conventional-commits/parser` | Same parser, kept as a hygiene tool. monorel doesn't depend on commit messages, so the lint stands on its own. |

**Conclusion:** no monorel changes needed for this migration. Any gap that surfaces during implementation is fixed in monorel as a separate change.

## Design

### 1. `monorel.toml`

Single file at repo root, 26 packages mirroring `.release-please-manifest.json` 1:1, in the same order so the migration diff is reviewable as "delete config, add toml" with the same package list.

```toml
[forge]
provider = "github"
owner    = "loglayer"
repo     = "loglayer-go"

[packages."go.loglayer.dev"]
tag_prefix = ""
path       = "."
changelog  = "CHANGELOG.md"

[packages."transports/charmlog"]
tag_prefix = "transports/charmlog"
path       = "transports/charmlog"
changelog  = "transports/charmlog/CHANGELOG.md"

# ... 24 more entries, one per release-please-manifest.json key ...
```

**Key conventions:**

- Root package key is `"go.loglayer.dev"` (the import path / module name).
- Sub-module package keys use the path (`"transports/zerolog"`), not the full import path. Matches monorel's worked-example recipe and what `monorel add --package "<key>:<level>"` accepts.
- Order mirrors `.release-please-manifest.json` for review-time diffability.

The `exclude-paths` block from release-please's config is not carried over — changesets are the only release signal in monorel, so root-level changes to `.claude/`, `.github/`, `docs/`, `Makefile`, etc. don't bump the root unless a changeset names it. The "main module stays on v1 forever" property is preserved by construction, not by an exclude list.

### 2. Workflows

#### `release-pr.yml` (new)

Fires on every push to `main`. Skips the `chore(release):` merge commit (which lands when the always-open release PR is merged) so the workflow doesn't churn the PR right after merge. Uses `disaresta-org/monorel/ci/github@v0.1.0` (pinned exactly because monorel hasn't shipped a moving `@v0` major-track tag yet; pre-1.0 convention).

```yaml
name: release-pr
on: { push: { branches: [main] } }
permissions: { contents: write, pull-requests: write }
jobs:
  release-pr:
    if: ${{ !startsWith(github.event.head_commit.message, 'chore(release):') }}
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with: { fetch-depth: 0 }
      - uses: disaresta-org/monorel/ci/github@v0.1.0
        with: { command: pr }
```

#### `release.yml` (new)

Fires on the `chore(release):` merge commit. Runs the release pipeline (release → push --follow-tags → publish) inside the action wrapper. Includes a `deploy-docs` job that calls `docs.yml` via `workflow_call`, mirroring the existing release-please-driven docs-deploy flow.

```yaml
name: release
on: { push: { branches: [main] }, workflow_dispatch: }
permissions: { contents: write }
jobs:
  release:
    if: github.event_name == 'workflow_dispatch' || startsWith(github.event.head_commit.message, 'chore(release):')
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with: { fetch-depth: 0 }
      - uses: disaresta-org/monorel/ci/github@v0.1.0
        with: { command: release }
  deploy-docs:
    needs: release
    if: ${{ needs.release.result == 'success' }}
    uses: ./.github/workflows/docs.yml
    permissions: { contents: read, pages: write, id-token: write }
```

The `workflow_call` is required because monorel's `publish` step creates GitHub Releases via `GITHUB_TOKEN`, and GitHub's anti-recursion rule blocks `release: published` events from propagating to other workflows when the release was created via that token. Same constraint release-please already worked around.

#### `release-please.yml` and `release-please-cleanup.yml` (deleted)

Both go away.

#### `docs.yml` (rewrite header comment only)

The trigger logic stays identical (`pull_request` + `release: published` + `workflow_dispatch` + `workflow_call`). Only the comment block at the top is rewritten to refer to monorel instead of release-please.

#### `pr-title.yml` and `commit-lint.yml` (rewrite header comments only)

Both keep functioning unchanged. Their leading comments currently frame the parser as "the same parser release-please uses" — rewrite to drop that framing. The conventional-commits parser is a hygiene tool that stands on its own; monorel has no opinion about commit messages.

### 3. lefthook + scripts cleanup

**`lefthook.yml`:** remove the `release-please-state` pre-push hook entry. Other hooks (commit-msg lint, gofmt + vet + staticcheck pre-commit, pre-push test runner) keep working unchanged.

**`scripts/check-release-please-state.sh`:** deleted.

**`scripts/foreach-module.sh`:** kept. Not release-please-coupled — walks `ALL_MODULES` for per-module `go test`, `gofmt`, etc.

**`scripts/lint-commit.mjs`:** kept. Same parser, same hygiene value.

### 4. Prose rewrite

#### AGENTS.md (the bulk of the diff)

Three sections rewrite, one deletes:

- **Rewrite "Versioning and Changelog"** as the monorel section. Describes: `monorel.toml` per-package config, `.changeset/<name>.md` files as the release signal, the always-open release PR pattern, the cut-a-release procedure (`monorel add ...` in feature PRs, push, merge the auto-opened release PR). Keeps the framing that "scopes that match a sub-module's component name trigger a release of that sub-module" but expresses it via `monorel add --package "<scope>:<level>"` rather than `feat(<scope>): ...`.

- **Rewrite "Adding a new transport, plugin, or integration"** to drop steps 3, 4, 8, 9 (release-please config registration with `release-as: "1.0.0"`, manifest entry, follow-up cleanup PR removing the stale `release-as`) and the `release-please-cleanup.yml` mention. New step list: create directory + `go.mod`, add to `monorel.toml`, add to `scripts/foreach-module.sh`, add to `go.work`, run tidy + test, open PR. First release for the new package happens by adding a `<path>:major` (or `:minor` / `:patch`) changeset in a follow-up PR.

- **Delete "Release-please gotchas"** entirely. All three documented gotchas (full-history `Release-As:` scan leakage, squash-merge stripping footers, `exclude-paths` completeness) are impossible by construction in monorel: the only release signal is `.changeset/*.md` files in the merged PR.

- **Keep unchanged:** the thread-safety contract, the "Performance: Attempted and Rejected" log, the "CI / Release Workflows" overview (with workflow names swapped: `release-please.yml` → `release-pr.yml` + `release.yml`), the "Currently Out of Scope" list, "Vulnerability scanning", the "Git Hooks (lefthook)" section minus the `release-please-state` reference.

#### README.md

Find any `.release-please-manifest.json` link and replace with `monorel.toml`. Update the "to cut a release" snippet from "merge the release-please PR" to "merge the always-open release PR".

#### docs/src/whats-new.md (line 46)

Swap the `.release-please-manifest.json` link to `monorel.toml`. The historical "Multi-module split" entry stays — it documents what happened at the time, just with a refreshed link target.

#### docs/src/public/llms-full.txt (lines 53, 1113)

Same link swap.

### 5. Migration PR commit shape

**Single commit, single PR.** Subject: `chore: migrate from release-please to monorel`.

The PR includes no `.changeset/*.md` files. After merge, the `release-pr` workflow runs once on the merge commit, sees an empty changeset set, and either no-ops (no open release PR yet) or closes a stale one (none exists). The first real changeset lands in the next feature PR.

The `.changeset/` directory itself ships as part of the migration PR with a `README.md` explaining what it is — see [the canonical changesets format docs](https://monorel.disaresta.com/changesets) for content reference.

### 6. Validation

#### Pre-merge (local)

1. `go install monorel.disaresta.com/cmd/monorel@v0.1.0` — verify the binary installs and runs.
2. `monorel plan` — must print "No pending changesets. Nothing to release." (no error, no spurious package release).
3. For all 26 packages, scripted-check that `git tag --list "<prefix>v*" | sort -V | tail -1` matches the value in `.release-please-manifest.json`. The check exits non-zero on any mismatch. A mismatch means monorel's planner would pick the wrong "from" version after merge — investigate before opening the PR. The implementation plan owns the exact script.
4. `monorel preview` (no `--upsert`) — should render an empty plan markdown.
5. `grep -rIE "release-please|release_please|releasePlease" --exclude-dir=.git --exclude-dir=node_modules .` — any hit outside of:
   - per-package `CHANGELOG.md` files (historical entries stay verbatim);
   - `docs/src/whats-new.md` "Multi-module split" entry (historical context);
   - `docs/src/public/llms-full.txt` historical references;

   …is a stale reference and gets removed.

#### Post-merge (CI + follow-up PR)

1. The `release-pr` workflow fires once on the merge commit, completes green, opens no spurious release PR.
2. In a follow-up PR, add a trivial `.changeset/<name>.md` targeting one transport at `:patch` with a body line documenting the migration (e.g. "Smoke test of the monorel migration."). The PR doesn't need a real code change — the changeset alone exercises the planner + orchestrator end-to-end.
3. After the follow-up PR merges, confirm the always-open release PR opens correctly.
4. Either merge the release PR (cuts a real `transports/<name>/v1.6.2` tag and GitHub Release) or close it without merging (validates the orchestrator's close path).

### 7. Rollback

If the migration PR merges and the release pipeline misbehaves on the follow-up release-PR merge, rollback is:

1. Revert the migration PR.
2. Restore `.release-please-config.json` and `.release-please-manifest.json` from the revert.
3. Restore the two release-please workflows from the revert.
4. The follow-up release PR (created by monorel) closes when its branch's `monorel.toml` no longer exists (or just close it manually).
5. Tags created by monorel before rollback stay — they're git tags, the same shape release-please would have produced. release-please reads existing tags as authoritative, so it picks up where monorel left off.

The rollback is non-destructive: existing tags, existing CHANGELOGs, existing GitHub Releases are all preserved either way.

## Open questions / future work

- **`@v0.1.0` exact pin** keeps loglayer-go on a specific monorel version. Each monorel patch release requires a manual loglayer-go workflow bump. Once monorel ships a moving `@v1` (post-v1.0.0) or `@v0` major-track tag, switch loglayer-go's pin.
- **Pre-release windows** (`monorel pre enter rc`) aren't used by this migration but are available. Document the workflow when first needed.
- **CHANGELOG format hard-cut** means the per-package CHANGELOGs will visually have two formats: Keep-a-Changelog at the top (new entries), release-please format below (historical). Both render correctly on GitHub. If consistency matters more than preservation, a one-time format conversion could happen later as a separate PR; not in scope here.

## Files changed (summary)

**Added:**
- `monorel.toml` (~80 lines, 26 packages)
- `.github/workflows/release-pr.yml` (~15 lines)
- `.github/workflows/release.yml` (~25 lines)
- `.changeset/README.md`

**Removed:**
- `.release-please-config.json`
- `.release-please-manifest.json`
- `.github/workflows/release-please.yml`
- `.github/workflows/release-please-cleanup.yml`
- `scripts/check-release-please-state.sh`

**Modified:**
- `AGENTS.md` (rewrite "Versioning and Changelog", "Adding a new transport, plugin, or integration"; delete "Release-please gotchas"; small swaps elsewhere)
- `lefthook.yml` (remove `release-please-state` hook)
- `.github/workflows/docs.yml` (header comment only)
- `.github/workflows/pr-title.yml` (header comment only)
- `.github/workflows/commit-lint.yml` (header comment only)
- `README.md` (link + release procedure)
- `docs/src/whats-new.md` (link)
- `docs/src/public/llms-full.txt` (link)

Total: 4 added files, 5 deleted files, 8 modified files.
