# Changesets

This directory holds pending release intents for [monorel](https://monorel.disaresta.com).

## What's a changeset?

A `.changeset/<name>.md` file declares which packages should release at what bump level when the next release lands, plus the changelog body to use for each release.

Example:

```markdown
---
"transports/zerolog": minor
"go.loglayer.dev": patch
---

Adds Lazy() helper for deferred field evaluation. Pass-through fix in the root.
```

The frontmatter keys are `monorel.toml` package names (`go.loglayer.dev` for the root, `transports/<name>` / `plugins/<name>` / `integrations/<name>` for sub-modules). Bump levels are `major`, `minor`, or `patch`. The body becomes the rendered changelog entry for every package the changeset names.

## Authoring

Run `monorel add` for the interactive flow, or write the file directly:

```sh
monorel add \
  --package "transports/zerolog:minor" \
  --message "Adds Lazy() helper."
```

Filenames are auto-generated (two-word, e.g. `quick-otter.md`). Multiple changesets per PR are fine; monorel rolls them up at release time.

## Lifecycle

1. PR includes a `.changeset/<name>.md` file describing the intended release(s).
2. On every push to `main`, the `release-pr` workflow updates the always-open release PR with the cumulative plan.
3. Merging the release PR runs the `release` workflow: writes the per-package CHANGELOG entries, creates the per-package tags, deletes the consumed `.changeset/*.md` files, and creates the GitHub Releases.

## Reference

- [monorel docs](https://monorel.disaresta.com)
- [Changeset file format](https://monorel.disaresta.com/changesets)
- [`monorel.toml`](../monorel.toml) — the canonical list of releasable packages.
