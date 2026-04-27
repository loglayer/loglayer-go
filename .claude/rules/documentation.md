# Documentation Rules

## Prose Style

### No em dashes. Ever.

Use the right replacement based on what the em dash was doing. **A bare comma is almost always wrong** because it usually creates a comma splice (two independent clauses joined by a comma).

| Em-dash pattern | Replacement |
|-----------------|-------------|
| `X — Y` where Y is a noun phrase that defines/elaborates X | colon: `X: Y` |
| `X — Y` where Y is a complete clause | period: `X. Y.` |
| `X — Y` where X and Y are tightly related independent clauses | semicolon: `X; Y` |
| `X — Y, Z, A — B` (mid-sentence parenthetical aside) | parens: `X (Y, Z, A) B` |
| `## Heading X — Y` (heading split into two parts) | colon or parens: `## Heading X: Y` or `## Heading X (Y)` |
| Bullet list separator: `- name — description` | colon: `- name: description` |

```markdown
✅ "Fields are keyed data: request IDs, user info, session data."  (colon for elaboration)
✅ "WithFields merges. It does not replace."  (period for two clauses)
✅ "WithFields merges; WithoutFields removes by key."  (semicolon for tight pair)
✅ "Use ApplicationFn (or override per-call) to customize."  (parens for aside)
✅ "## loglayer.Metadata: the canonical map shape"  (heading colon)
✅ "- WithFields: persists across logs."  (bullet colon)

❌ "Fields are keyed data, request IDs, user info, session data."  (comma splice / list confusion)
❌ "WithFields merges, it does not replace."  (comma splice)
❌ "## loglayer.Metadata, the canonical map shape"  (reads as a list)
```

When in doubt: split into two sentences. Period is always safe.

### Heading patterns

- Two-part headings use a colon: `## Silent Mock: NewMock()`, not `## Silent Mock — NewMock()` and not `## Silent Mock, NewMock()`.
- A heading should not look like a comma-separated list unless it actually is one. `## Replacing, Not Merging` reads as two items; prefer `## Replacement, Not Merge` or `## WithMetadata Replaces (Doesn't Merge)`.

### Frontmatter gotcha

If a frontmatter `description:` value contains a `:`, the value must be quoted, otherwise the YAML parser fails:

```yaml
✅ description: "Per-log structured data: maps, structs, or any value."
❌ description: Per-log structured data: maps, structs, or any value.
```

### Cross-reference link text

When linking to a sub-heading, use just the heading title:

```markdown
✅ See [Fatal Exits the Process](/logging-api/basic-logging#fatal-exits-the-process).
❌ See [Basic Logging, Fatal Exits the Process](/logging-api/basic-logging#fatal-exits-the-process).
❌ See [Basic Logging — Fatal Exits the Process](/logging-api/basic-logging#fatal-exits-the-process).
```

### General rules

- Lead with the conclusion, then explain.
- Default to short sentences; one idea per sentence.
- Don't write paragraphs that exist only to introduce the next paragraph.
- Avoid "let's", "we'll", "you'll find that" filler.

This applies to every doc page, README, code comment, commit message, PR description, and chat response.

## README Requirements

The repo root `README.md` should be minimal:

1. Project name and one-line description.
2. Install command (`go get go.loglayer.dev`).
3. Minimal usage example (one transport, three to five lines of Go).
4. Link to the docs site for everything else.

Reserve deeper details for the VitePress site. Do not duplicate full reference content in the README.

## Site Documentation Structure

Docs live under `docs/src/`. Layout:

```
docs/src/
├── index.md                    Homepage (hero + quick example + transport list partial)
├── introduction.md             "Why use LogLayer?"
├── getting-started.md          Install + minimal example
├── configuration.md            Every Config field
├── cheatsheet.md               One-page API reference
├── logging-api/                Per-method guides
└── transports/                 Per-transport guides
    └── _partials/              Shared markdown fragments
```

Required elements per page:

- Frontmatter with `title` and `description` (used for OG meta).
- One H1 matching the title.
- Concrete code examples in Go syntax-highlighted blocks (` ```go `).
- Output samples in matching syntax (` ```json ` for structured, plain ` ``` ` for terminal).

## Partial Files

Lists that appear in multiple places live as a single partial in `_partials/`, included via:

```markdown
<!--@include: ./_partials/transport-list.md-->
```

Currently maintained partials:

- `docs/src/transports/_partials/transport-list.md`: catalog of all transports with category, link, description, dependency.
- `docs/src/logging-api/_partials/combining-example.md`: shared Go + JSON snippet showing `WithFields` + `WithMetadata` + `WithError` chained together. Included from `fields.md` and `metadata.md`.

When adding a new transport, update the transport-list partial. Both the homepage and the transports overview render it.

## When Adding a New Transport

1. Add an entry to `docs/src/transports/_partials/transport-list.md` under the right category (Renderers or Logger Wrappers).
2. Create a new `docs/src/transports/<name>.md` page matching the existing pattern: install, basic usage, config, fatal behavior, metadata handling, level mapping, GetLoggerInstance.
3. Add a sidebar entry in `docs/.vitepress/config.ts`.
4. Run `cd docs && bun run docs:build` and confirm clean.

## When Adding a New API or Config Field

1. Update `docs/src/configuration.md` (if a Config field) or the relevant `logging-api/` page.
2. Update `docs/src/cheatsheet.md` to include the new method or field in the quick reference.
3. Update `docs/src/introduction.md` only if the change is a real selling point, not implementation detail.

## Code Examples in Docs

Use `loglayer.Metadata` (the alias) as the canonical map shape, not `map[string]any`:

```go
// ✅ idiomatic in docs
log.WithMetadata(loglayer.Metadata{"userId": 42}).Info("user")

// ❌ verbose
log.WithMetadata(map[string]any{"userId": 42}).Info("user")
```

Exception: `ErrorSerializer` signatures and transport implementation code in `creating-transports.md` use `map[string]any` because they're showing the underlying type at a system boundary.

Show the call shape correctly: chain `WithMetadata`/`WithError` *before* the level method.

```go
// ✅ correct
log.WithMetadata(loglayer.Metadata{"k": "v"}).Info("hello")

// ❌ wrong: LogLayer doesn't accept metadata as a positional arg
log.Info("hello", loglayer.Metadata{"k": "v"})
```

The `BaseConfig.ID` field on a transport is optional in most examples. Include it only when the example needs it (multi-transport, `RemoveTransport`, `GetLoggerInstance`).

## Configuration Tables

When documenting a config struct, prefer a table for quick scanning:

```markdown
| Field | Type | Default | Description |
|-------|------|---------|-------------|
```

Use code blocks for the full struct shape only when the type/default columns would push line length too far.

## Custom Containers (VitePress)

```markdown
::: info Title
Informational note.
:::

::: tip Title
Recommended approach or shortcut.
:::

::: warning Title
Behavior to be aware of; default workflow still works.
:::

::: danger Title
MUST-type instructions or behavior that breaks expectations.
:::

::: details Title
Collapsible block for optional lengthy information.
:::
```

Use `:::danger` sparingly. Reserve it for cases like "this transport will exit the process even when DisableFatalExit is set."

## Fatal Behavior Section in Transport Docs

Every transport doc page must have a `## Fatal Behavior` section. The shape:

- For renderers (console, pretty, structured, testing): "writes the entry; core decides whether `os.Exit(1)` is called via `Config.DisableFatalExit`."
- For wrappers that defer to core (zerolog, zap, charmlog, logrus): describe how the underlying library's Fatal is bypassed (specific mechanism: `WithLevel`, custom `CheckWriteHook`, `Log()` method, no-op `ExitFunc`).
- For wrappers that can't bypass (phuslu): use a `:::danger` block. The user needs to know before they pick the transport.

## Words to Avoid

- "first-party" / "First-party" — every transport, plugin, and integration in the loglayer-golang module is part of the same module; calling them "first-party" implies a tier that doesn't exist. Just say "built-in" if a qualifier is needed at all, or drop the qualifier entirely (e.g. "the redact plugin", not "the first-party redact plugin").
- Em dashes anywhere.

## When You Add a New Feature or Make an API Change

Update all of these:

1. **`docs/src/cheatsheet.md`**: add the new method/field to the quick reference.
2. **`docs/src/whats-new.md`**: add a bullet under the current date heading. If the date doesn't exist yet, add it at the top.
3. **`docs/src/public/llms.txt`**: concise LLM-facing reference. Add a link or bullet for the new surface.
4. **`docs/src/public/llms-full.txt`**: comprehensive LLM-facing reference. Add a section or bullet describing the new surface.
5. **`CHANGELOG.md`** (repo root): add an entry under `## [Unreleased]` in the appropriate component subsection. Format follows [Keep a Changelog](https://keepachangelog.com).
6. The relevant doc page (e.g. `configuration.md`, the `logging-api/` page, or the `transports/<name>.md` page).

For a brand-new transport, also see "When Adding a New Transport" above.

## When You Make a Perf Change

If a change is intended to improve performance:

1. Run benchmarks before and after with `benchstat` (see `.claude/rules/benchmarking.md`).
2. If the change is rejected (no improvement, or slower), record it in `AGENTS.md` under "Performance: Attempted and Rejected" so it doesn't get re-attempted.
3. If the change is kept, update the numbers in `docs/src/benchmarks.md`.

## Thread-Safety Statements

If you add or change a method on `*LogLayer` or a transport, classify it:

- **Safe to call concurrently with emission**: read-only or atomic.
- **Setup-only**: mutates state, callers must coordinate.

Document this in the method's GoDoc comment. The current contract is summarized in `AGENTS.md` under "Thread Safety".

## Versioning Note

The repo is currently a single Go module. Do not create per-package `CHANGELOG.md` files. All entries go in the root `CHANGELOG.md`, grouped by component under each release. See `AGENTS.md` for the full versioning policy.
