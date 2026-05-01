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
| Bot anti-recursion for docs deploy | `release-please.yml` calls `docs.yml` via `workflow_call` | `release.yml` calls `docs.yml` via `workflow_call` — same `GITHUB_TOKEN` anti-recursion constraint applies to monorel-created Releases. monorel itself doesn't exercise this pattern in its own repo (its `docs.yml` deploys on every push to main); loglayer-go is the first to verify it. Listed in "Open questions / future work" as a smoke-test item. |
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

The `if` filter mirrors monorel's own `release-pr.yml` — skip the workflow on the release PR's merge commit so we don't churn the just-merged PR. Known limitation: a hand-authored PR titled `chore(release): ...` (e.g. a doc cleanup) would also skip, since the message-prefix filter is a heuristic. Acceptable for this repo (the `chore(release):` prefix is monorel-bot's by convention; we don't use it for hand-authored commits). If a false-skip ever bites, switch to filtering by `github.event.head_commit.author.username == 'monorel-bot[automation]'` per monorel's own author config.

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

- **Update AGENTS.md:58** ("Key Design Decisions" → "Multi-module layout" bullet): change "`.release-please-manifest.json` is the canonical list" to "`monorel.toml`'s `[packages]` map is the canonical list".

- **Update AGENTS.md:138** (Git Hooks lefthook section, commit-msg hook description): drop the "the same parser release-please uses" framing. The hook keeps working unchanged; only the framing comment changes. Same applies to the lefthook config file (line 9) — covered separately under "lefthook + scripts cleanup."

- **Keep unchanged:** the thread-safety contract, the "Performance: Attempted and Rejected" log, the "Currently Out of Scope" list, "Vulnerability scanning". The "CI / Release Workflows" overview gets a workflow-name swap (`release-please.yml` → `release-pr.yml` + `release.yml`); the "Git Hooks (lefthook)" section gets the `release-please-state` hook removed and the commit-msg framing fixed per the bullets above.

#### README.md

Find any `.release-please-manifest.json` link and replace with `monorel.toml`. Update the "to cut a release" snippet from "merge the release-please PR" to "merge the always-open release PR".

#### docs/src/whats-new.md (line 46)

Swap the `.release-please-manifest.json` link to `monorel.toml`. The historical "Multi-module split" entry stays — it documents what happened at the time, just with a refreshed link target.

#### docs/src/public/llms-full.txt (lines 53, 1113)

Same link swap.

#### CONTRIBUTING.md (line 35)

Drop the "release-please uses" framing on the conventional-commits parser description. The hygiene value of the parser stands on its own.

#### package.json (description string)

The package description references the parser as "the same parser release-please uses." Rewrite to describe the linter on its own terms (e.g. "Conventional-commit linter for `loglayer-go` git hooks and CI").

#### `.claude/rules/documentation.md` (line 299)

Rewrite the parenthetical "(release-please owns that)" framing for `CHANGELOG.md` ownership. With monorel, the CHANGELOG is still auto-maintained, just by `monorel release` instead of release-please. The agent rule about not hand-editing CHANGELOG entries below `[Unreleased]` stays the same.

#### `scripts/lint-commit.mjs` (lines 3, 8, 86)

Three comments frame the parser as release-please's. Rewrite to describe the parser independently (it's just `@conventional-commits/parser`).

#### Per-package CHANGELOG.md preambles

13 of the 26 CHANGELOG.md files (root + the older sub-modules whose preamble we hand-wrote) start with a multi-paragraph "Releases are managed by Release Please…" preamble that references release-please by name and links to `.release-please-manifest.json`. The other 13 (the ones release-please created with no preamble; just `# Changelog` and entries) need no preamble change.

For the 13 that have a preamble, replace the preamble with a monorel-equivalent paragraph. Body of the new preamble (root and sub-modules differ slightly):

> *Root* (`./CHANGELOG.md`): "All notable changes to this project are documented here. Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versioning follows [SemVer](https://semver.org/spec/v2.0.0.html). `go.loglayer.dev` is the main module — every transport, plugin, and integration ships as its own sub-module under its own tag (`<path>/v<X.Y.Z>`); the canonical list lives in `monorel.toml`. See `AGENTS.md` for the layout and release flow. From v1.0.0 forward, this file is maintained automatically by [monorel](https://monorel.disaresta.com)."

> *Sub-module* (e.g. `transports/zerolog/CHANGELOG.md`): "All notable changes to `go.loglayer.dev/transports/zerolog` are documented here. Format follows Keep a Changelog; versioning follows SemVer. Tags use the prefixed form `transports/zerolog/v<X.Y.Z>`. Maintained automatically by [monorel](https://monorel.disaresta.com)."

The historical version entries below the preamble stay verbatim. The implementation plan owns the per-file diff; the spec only specifies the new preamble shape.

The 13 preamble-less CHANGELOGs (release-please-generated) get no edits in this PR. They'll naturally accumulate Keep-a-Changelog-shaped monorel entries above the existing release-please-shaped entries on the next release.

### 5. Migration PR commit shape

**Single commit, single PR.** Subject: `chore: migrate from release-please to monorel`.

The PR includes no `.changeset/*.md` files. After merge, the `release-pr` workflow runs once on the merge commit, sees an empty changeset set, and either no-ops (no open release PR yet) or closes a stale one (none exists). The first real changeset lands in the next feature PR.

The `.changeset/` directory itself ships as part of the migration PR with a `README.md` explaining what it is — see [the canonical changesets format docs](https://monorel.disaresta.com/changesets) for content reference.

### 6. Validation

#### Pre-merge (local)

1. `go install monorel.disaresta.com/cmd/monorel@v0.1.0` — verify the binary installs and runs.
2. `monorel plan` — must print "No pending changesets. Nothing to release." (no error, no spurious package release).
3. For all 26 packages, scripted-check that the latest tag matches the value in `.release-please-manifest.json`. The glob is `<prefix>/v*` for prefixed packages and bare `v*` for the root (where `tag_prefix = ""`); concretely:
   - Root: `git tag --list 'v*' | grep -E '^v[0-9]' | sort -V | tail -1` — compare to manifest `.`.
   - Sub-module: `git tag --list "<path>/v*" | sort -V | tail -1` — compare to manifest `<path>` (e.g. `transports/zerolog`).

   The check exits non-zero on any mismatch. A mismatch can mean two things:
   - **Manifest is stale, latest tag is correct** (most common — happens when release-please errored and a tag was published anyway, or when bootstrapping a sub-module manually). monorel will pick the correct tag-derived version on the first `monorel plan` run, which is the right behavior. Migration is safe to proceed; flag in the PR description so reviewers know.
   - **Manifest is correct, latest tag is wrong** (rare — would require a manually-pushed tag that release-please refused). monorel will pick the spurious tag. Either delete the spurious tag from origin first, or document the intent and bump intentionally.

   The implementation plan owns the exact script and the decision matrix for each finding.
4. `monorel preview` (no `--upsert`) — should render an empty plan markdown.
5. `grep -rIE "release-please|release_please|releasePlease" --exclude-dir=.git --exclude-dir=node_modules --exclude-dir=docs/.vitepress/cache .` — any hit outside of:
   - per-package `CHANGELOG.md` historical version entries (the entries below the preamble stay verbatim; the preambles themselves get rewritten per section 4);
   - `docs/src/whats-new.md` "Multi-module split" entry (historical context, with the link target swapped to `monorel.toml`);
   - `docs/src/public/llms-full.txt` historical references (link targets swapped);
   - this design spec itself (`docs/superpowers/specs/2026-05-01-monorel-migration-design.md`) — references release-please by name throughout, by design.

   …is a stale reference and gets removed.

#### Post-merge (CI + follow-up PR)

1. The `release-pr` workflow fires once on the merge commit, completes green, opens no spurious release PR.
2. In a follow-up PR, add a trivial `.changeset/<name>.md` targeting **`transports/blank` at `:patch`** with a body line documenting the migration (e.g. "Smoke test of the monorel migration."). `transports/blank` is the canonical no-op transport (used as a template / placeholder); a `:patch` bump from `v1.6.1` → `v1.6.2` on it is the lowest-stakes real release the repo can produce. Avoid using a heavily-imported transport (zerolog, zap) for the smoke test, since the version bump propagates to dependents on pkg.go.dev with no functional change.
3. After the follow-up PR merges, confirm the always-open release PR opens correctly.
4. Step 4 is a deliberate trade-off; pick one explicitly:
   - **Merge the release PR** — cuts a real `transports/blank/v1.6.2` tag and GitHub Release. Validates the full pipeline end-to-end including: release commit, tag push, `monorel publish` Release creation, downstream `build-release-binaries`-style workflows (none for loglayer-go itself, but the `deploy-docs` `workflow_call` chain is exercised). Cost: a real version bump on `transports/blank` with no functional change.
   - **Close the release PR without merging** — validates only the orchestrator's close path and the `release-pr` workflow. Does NOT exercise: `monorel release`, the tag push, `monorel publish`, the `release.yml` workflow, or the `deploy-docs` chain. Cost: lower confidence in the full pipeline; first real release after migration is the first time `release.yml` runs in this repo.

   Recommendation: **merge**, accepting the no-op version bump as the cost of validating the full chain. The alternative defers risk to whenever the next real changeset lands, which could be days or weeks later under different conditions.

5. **If the merge path is chosen**, additionally verify after `release.yml` completes: (a) the tag `transports/blank/v1.6.2` exists on origin, (b) a corresponding GitHub Release was created with the rendered changelog body, (c) the `deploy-docs` job ran and updated GitHub Pages (if any docs changes were in the PR; if not, the build job runs but produces no visible diff). Failure at (c) means the `workflow_call` anti-recursion claim from the audit table doesn't hold and needs a separate fix (likely a Personal Access Token instead of `GITHUB_TOKEN` for the `monorel publish` step).

### 7. Rollback

Rollback complexity depends on whether monorel cut any real tags before the rollback fires. Two distinct cases:

#### Case A: Rollback before any monorel-driven release fires (cheap, non-destructive)

If the migration PR merges but is reverted before the post-merge follow-up smoke-test PR cuts a real tag, rollback is non-destructive:

1. Revert the migration PR.
2. The revert restores `.release-please-config.json`, `.release-please-manifest.json`, and the two release-please workflows.
3. The auto-opened release PR (if any was opened by monorel from the migration PR alone, which it shouldn't be — there are no changesets — but defensively) gets closed manually.
4. release-please's next run on the next push to main picks up where it left off, since neither tags nor manifest have moved.

Existing tags, existing CHANGELOGs, existing GitHub Releases are all preserved.

#### Case B: Rollback after monorel cut a tag (requires manual reconciliation)

If monorel-driven releases have already produced new tags (e.g. `transports/zerolog/v1.7.0`), rollback is **not** non-destructive. The reconciliation steps:

1. Identify every package whose tag advanced during the monorel period: `git diff <pre-migration-sha>..HEAD -- '*/CHANGELOG.md' CHANGELOG.md` and cross-reference with `git tag --list` for tags created since the pre-migration SHA.

2. For each such package, **hand-edit `.release-please-manifest.json`** to match the new tag's version. This is required because release-please's source of truth is the manifest, not git tags. Without this step, release-please will read the manifest's stale value, compute commit ranges from there, and either:
   - Try to re-create a tag that already exists (fails with a CI error you'll have to manually intervene on), or
   - Compute a smaller bump than the existing tag and silently produce a divergent tag namespace (`v1.6.2` co-existing with `v1.7.0` on the same module).

3. Commit the manifest reconciliation alongside the migration revert (single PR is fine).

4. Optionally close any GitHub Releases monorel created if they should be hidden — but the underlying tags must stay (deleting them would break `go get` for anyone who's already pinned).

5. release-please picks up from the reconciled manifest on the next push.

The "no destructive operations" property of the migration PR itself stands; what's destructive is the **interaction** of (monorel-cut tags) + (release-please reading a stale manifest) — and that requires the manual manifest patch to resolve.

#### Rollback decision gate

Before reverting, run: `git tag --list --contains <pre-migration-sha>` to enumerate any tags created since migration. If the list is empty, Case A applies. Otherwise, Case B.

## Open questions / future work

- **`@v0.1.0` exact pin** keeps loglayer-go on a specific monorel version. Each monorel patch release requires a manual loglayer-go workflow bump. Once monorel ships a moving `@v1` (post-v1.0.0) or `@v0` major-track tag, switch loglayer-go's pin.
- **Pre-release windows** (`monorel pre enter rc`) aren't used by this migration but are available. Document the workflow when first needed.
- **CHANGELOG format hard-cut** means the per-package CHANGELOGs will visually have two formats: Keep-a-Changelog at the top (new entries), release-please format below (historical). Both render correctly on GitHub. If consistency matters more than preservation, a one-time format conversion could happen later as a separate PR; not in scope here.
- **`workflow_call` docs-deploy verification.** monorel's own repo doesn't exercise the `release.yml` → `docs.yml` `workflow_call` chain (its `docs.yml` deploys on every push). loglayer-go is the first to verify that monorel's `GITHUB_TOKEN`-created Releases trigger the same anti-recursion behavior release-please-created Releases do. Smoke-tested in section 6 post-merge step 5; if it fails, the fix is likely a Personal Access Token for the `monorel publish` step.
- **monorel-side recipe update.** `monorel/docs/src/recipes/loglayer-go.md` currently has a `::: warning Pending` block — it's a placeholder for the actual loglayer-go migration. Once this migration lands and the smoke-test release is cut, the recipe should be updated with real commit-by-commit details (PR link, manifest-vs-tag mismatch findings if any, smoke-test outcome, deploy-docs verification result). This is a follow-up PR on the monorel repo, not this one.
- **Initial-vs-stale manifest classification (section 6 step 3).** The decision matrix between "manifest is stale" and "tag is wrong" is currently described in prose. If the spot-check script finds many mismatches at implementation time, codify the decision tree as part of the script's output (e.g. emit the recommended action per mismatch).

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
- `AGENTS.md` (rewrite "Versioning and Changelog" + "Adding a new transport, plugin, or integration"; delete "Release-please gotchas"; update line 58 in "Key Design Decisions"; update line 138 in "Git Hooks (lefthook)" framing; workflow-name swap in "CI / Release Workflows" overview)
- `lefthook.yml` (remove `release-please-state` hook + update line 9 framing comment)
- `.github/workflows/docs.yml` (header comment only)
- `.github/workflows/pr-title.yml` (header comment only)
- `.github/workflows/commit-lint.yml` (header comment only)
- `README.md` (link + release procedure)
- `CONTRIBUTING.md` (line 35 framing)
- `package.json` (description string)
- `.claude/rules/documentation.md` (line 299 framing)
- `scripts/lint-commit.mjs` (lines 3, 8, 86 framing)
- `docs/src/whats-new.md` (link)
- `docs/src/public/llms-full.txt` (link)
- 13 per-package CHANGELOG.md preamble rewrites (root + the older sub-modules with hand-written preambles): `./CHANGELOG.md`, `transports/{zerolog,otellog,lumberjack,zap,http,datadog,charmlog,phuslu,pretty,logrus,gcplogging}/CHANGELOG.md`, `plugins/oteltrace/CHANGELOG.md`. The other 13 CHANGELOGs (release-please-generated, no preamble) stay untouched.

Total: 4 added files, 5 deleted files, 25 modified files (12 prose + 13 CHANGELOG preambles).
