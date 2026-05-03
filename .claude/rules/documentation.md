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

### Density: prefer sub-sections to dense bullets

If a section is a bulleted list where each bullet is its own multi-sentence concept, break it into named `###` sub-sections. The reader should be able to scan headings to find the strategy that matches their situation. Heuristic: 2+ bullets that each combine a name, a technique, caveats, and a link → split.

Single-sentence bullets are fine and don't need this treatment.

### Casual users vs implementers

Pages aimed at *callers* (logging-api/, transports/index, plugins/index, configuration, getting-started, introduction) should not bleed implementation details that only a transport or plugin author needs. That includes:

- The `TransportParams` struct shape and its fields.
- Plugin hook signatures and lifecycle phase names beyond what a *user installing* a plugin needs to understand timing.
- Internal helpers (`MetadataAsRootMap`, `MergeFieldsAndMetadata`, `BaseTransport`, etc.).
- Typed errors used only by hooks (e.g. recovered-panic types).

Implementer-only material lives in `creating-transports.md`, `creating-plugins.md`, and the relevant per-implementation page. Casual-user pages can mention that those creator pages exist (one-liner pointer is fine), but should not paraphrase their content.

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
├── plugins/
│   ├── index.md                Overview + catalog only
│   ├── configuration.md        Construction-time wiring (Config.Plugins, IDs)
│   ├── management.md           Runtime mutation (AddPlugin / RemovePlugin / etc.)
│   ├── creating-plugins.md     Authoring guide
│   └── _partials/
└── transports/
    ├── index.md                Overview + catalog only
    ├── configuration.md        Construction-time wiring (Config.Transport(s), BaseConfig)
    ├── management.md           Runtime mutation (AddTransport / etc.)
    ├── multiple-transports.md  Fan-out semantics
    ├── creating-transports.md  Authoring guide
    └── _partials/
```

**The four-page pattern for transports and plugins.** Overview / Configuration / Management / Creating. The split mirrors the TS loglayer site and keeps each page narrow:

- **Overview** (`index.md`): "what is this" + the catalog partial. Don't put management or configuration prose here.
- **Configuration**: construction-time wiring. `Config` fields the user sets, ID semantics, defaults.
- **Management**: runtime mutation. `AddX`, `RemoveX`, `GetX`, replace-by-ID semantics, concurrency.
- **Creating**: authoring guide for users implementing the interface. All implementation-side details live here (helpers, hooks, lifecycle).

Each pair (Configuration / Management) cross-references the other in its first paragraph.

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

## Go Version Floors

The main `go.loglayer.dev` module's Go floor is whatever the highest dep in its tree demands. Today that's **1.25** (driven by `golang.org/x/exp` via `charmbracelet/log` and `golang.org/x/sys`). Sub-modules — `transports/otellog`, `plugins/oteltrace`, `plugins/datadogtrace/livetest` — have their own go.mod files and their own floors.

When adding a transport, plugin, or integration:

1. **If your dep would raise the main module's floor**, first ask whether splitting your code into its own go.mod would isolate the bump. Heavy SDK bindings (OpenTelemetry, vendor APIs) are good candidates for splitting; small libraries that nudge the floor by one minor version usually aren't.

2. **If you split**, mirror the structure used by `transports/otellog/`: own `go.mod` with `module go.loglayer.dev/<path>`, `replace go.loglayer.dev => ../...` for development, a placeholder `require go.loglayer.dev v0.0.0-...` line that the replace directive overrides. Add a CI step in `.github/workflows/ci.yml` that `cd`s into the new module and runs tests. Update the `Mostly single Go module` bullet in AGENTS.md "Key Design Decisions" with the new module path.

3. **If you don't split and the floor moves**, update `go.mod`, the matrix in `.github/workflows/ci.yml`, and the version statements in `README.md`, `docs/src/getting-started.md`, and `AGENTS.md`. Add a `.changeset/*.md` for the affected module(s) at the appropriate bump level and note the floor change in `docs/src/whats-new.md`.

4. **Per-transport/plugin pages** for split modules need an `::: info Separate module` block at the top stating the import path and floor. Pages for sub-packages of the main module default to "inherits the module's floor" without restating the number; only call it out when the floor differs from the main module.

The catalog partials (`transports/_partials/transport-list.md`, `plugins/_partials/plugin-list.md`) deliberately do not restate the Go floor: per-transport/plugin pages own that information. Don't reintroduce a Go-version block to the catalog pages.

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

The `BaseConfig.ID` field on a transport, and the `id` argument to a plugin's `New*Hook` constructor (or whatever the plugin's `ID()` returns), is optional and auto-generated when empty. Include it in an example **only** when the example specifically demonstrates management (`RemoveTransport`, `GetLoggerInstance`, `RemovePlugin`, `GetPlugin`, replace-by-ID). Bare construction examples should leave it off.

## Configuration Tables

When documenting a config struct, prefer a table for quick scanning:

```markdown
| Field | Type | Default | Description |
|-------|------|---------|-------------|
```

Use code blocks for the full struct shape only when the type/default columns would push line length too far.

## Pitfalls / traps: inline, not centralized

Don't create a "Common Pitfalls" or similar page that aggregates every footgun in one place. Embed warnings inline on the page that owns the API the trap relates to, using a `::: warning` callout near the relevant API description. Keep the callout short: name the trap, show the ❌ and ✅ in side-by-side code if it helps, link to a deeper page only if there's already a good target.

Why: a centralized pitfalls page bit-rots, gets read once and forgotten, and means readers have to round-trip when they hit a snag. Inline warnings show up exactly when the reader is making the call that could trigger the trap.

Confirmed pitfalls already inlined:

- `WithFields` / `WithContext` returns a new logger → `fields.md`, `go-context.md`.
- Maps passed to `WithFields` / `WithMetadata` aren't deep-copied → `fields.md`, `metadata.md`.
- Mute toggles can interleave under concurrency → `fields.md`, `metadata.md` (in their Muting sections).
- `MetadataFieldName` only applies to non-map metadata → `metadata.md`.
- Phuslu can't suppress fatal exit → `transports/phuslu.md` (`:::danger` block).
- OTel transport silent-drops without a provider → `transports/otellog.md`.
- oteltrace plugin needs propagator across service boundaries → `plugins/oteltrace.md`.

When you ship a new trap, add a `::: warning` (or `:::danger` for breakage) on the owning page. Don't create a sibling reference.

## Testing helpers for transport / plugin authors

Two public helper packages back the testing sections of the creator docs:

- `transport/transporttest` (package `transporttest`): `RunContract` runs the 14-test wrapper-transport contract suite against any wrapper-shaped transport that produces JSON output. Also exports `ParseJSONLine` and `MessageContains`. Used by every built-in wrapper test.
- `plugins/plugintest` (package `plugintest`): `Install` wires a plugin into a fresh logger backed by the testing transport. `AssertNoMutation` proves an input-side hook doesn't mutate caller-owned input. `AssertPanicRecovered` drives a panicking hook and confirms `*loglayer.RecoveredPanicError` reaches `OnError`. Used by every built-in plugin test.

The doc pages that document these helpers are `transports/testing-transports.md` and `plugins/testing-plugins.md`, respectively. The matching `creating-*` pages have a one-paragraph "Testing" section that links to them; don't duplicate worked examples between the creating- and testing- pairs.

When you change a helper, update the testing-* page in the same change. The pattern in built-in tests is the canonical reference.

## Examples for strategy / policy choices

When a doc section explains *which* strategy to pick (e.g. "Picking a metadata placement policy"), back each strategy with a runnable example under `examples/<name>/`. Keep the doc text tight (intro + helper + caveats); let the example carry the full implementation. Link to the example on GitHub from the doc with a `[`examples/foo`](https://github.com/loglayer/loglayer-go/blob/main/examples/foo/main.go)` line.

Current pattern:

- `examples/custom-transport`: renderer / "flatten" policy via `MergeFieldsAndMetadata`.
- `examples/custom-transport-attribute`: wrapper / "attribute-forwarding" policy via `MetadataAsRootMap`.

Add to this set when a new policy emerges; don't try to teach the policy entirely in prose.

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

- "first-party" / "First-party": every transport, plugin, and integration in the loglayer-go module is part of the same module; calling them "first-party" implies a tier that doesn't exist. Just say "built-in" if a qualifier is needed at all, or drop the qualifier entirely (e.g. "the redact plugin", not "the first-party redact plugin").
- Em dashes anywhere.

## "What's new" page format (`docs/src/whats-new.md`)

Mirrors the [TypeScript `loglayer` whats-new page](https://loglayer.dev/whats-new.html). Distinct from the auto-generated root `CHANGELOG.md` (monorel writes that on release from `.changeset/*.md` files); maintained manually.

```markdown
## MMM DD, YYYY

`module-or-version`:

- **Title**: brief description of the change. See [Doc Page](/path).
```

Rules:

- **Top of page**: a one-bullet intro pointing at the root `CHANGELOG.md`. Nothing else.
- **`## MMM DD, YYYY`** date sections (e.g. `## Apr 29, 2026`), reverse chronological. The date is when the entries landed on `main`. If the date already exists, add to the existing section.
- Inside a date, group bullets by **scope** as a backticked plain paragraph followed by a colon (not a heading):
  - For the main module's API changes: `` `loglayer`: ``.
  - For sub-module changes: the import path in backticks: `` `transports/lumberjack`: ``, `` `plugins/fmtlog`: ``.
  - For cross-cutting work tied to a specific release: `` `vX.Y.Z`: `` (e.g. `` `v1.1.0`: ``, `` `v1.0.0`: ``).
- **Bullets only when listing multiple items.** A scope with a single change reads as a plain paragraph (or a few paragraphs) under the `scope:` line, not as a one-bullet list. Bullets are for genuine enumerations.
- **For initial releases of a new transport, plugin, or integration**, the whats-new entry is one short paragraph: what it does and a link to the doc page. Don't enumerate config knobs, severity mappings, or implementation details ("forwards entries to a caller-supplied X.Logger", "writes JSON-per-line via Y", etc.); that's the doc page's job.
- **Make the link text name the project's component, not the upstream product.** `[Sentry transport](/transports/sentry)` (clear: this links to our transport docs), not `[Sentry](/transports/sentry)` (ambiguous: looks like it should go to sentry.io). Same for plugins (`[redact plugin]`) and integrations (`[loghttp integration]`). The inline link replaces a trailing `See [...](...)` link; don't include both.
- **Lead with the conclusion.** The first sentence (or for multi-item bulleted scopes, the title before the colon) names the change in plain terms. Don't paraphrase the doc page; link to it.
- **Length: pick the shortest form that fits the change.** Most fixes and minor features are one or two sentences. For substantial changes (multi-aspect features, anything where forcing it into a single sentence produces a parenthetical wall), break the elaboration into paragraphs separated by blank lines so the reader can scan rather than parse. One idea per paragraph.
- Code or diff blocks belong attached to the entry: indented two spaces under a bullet, or as their own block separated by blank lines under a paragraph entry.
- Apply the project's general docs-style rules: no em dashes, no comma splices, no filler.

### Examples of each shape

**Single-item scope, brief (initial transport release):**

```markdown
`transports/foo`:

Initial release. New [Foo transport](/transports/foo).
```

**Multi-paragraph entry (substantial feature):**

```markdown
`loglayer`:

**`MetadataFieldName` is now a core `Config` knob.** Set it once and metadata nests under the configured key uniformly across every transport. Joins `FieldsKey` and `ErrorFieldName` as the third assembly-shape knob. See [`MetadataFieldName`](/configuration#metadatafieldname).

The resolved assembly shape is also published as `loglayer.Schema` on `TransportParams` and on the dispatch-time plugin hook param structs, so plugins can navigate `params.Data` without guessing keys.

The per-transport `Config.MetadataFieldName` field is removed from every wrapper; set the core knob instead.
```

**Multi-item bulleted scope:**

```markdown
`transports/pretty`:

- **Column-aligned YAML in expanded mode**: same-level scalar keys pad to the longest sibling so values line up.
- **Nested rendering for keyed metadata**: when `Config.MetadataFieldName` is set, the metadata value renders as YAML under the configured key.
```

## When You Add a New Feature or Make an API Change

Update all of these:

1. **`docs/src/cheatsheet.md`**: add the new method/field to the quick reference.
2. **`docs/src/whats-new.md`**: add a bullet under today's `## MMM DD, YYYY` date section, beneath the appropriate `` `module-or-version`: `` paragraph (creating the section and/or paragraph if they don't exist). Format per the rules above.
3. **`docs/src/public/llms.txt`**: concise LLM-facing reference. Add a link or bullet for the new surface.
4. **`docs/src/public/llms-full.txt`**: comprehensive LLM-facing reference. Add a section or bullet describing the new surface.
5. **`.changeset/<name>.md`**: add a changeset naming the affected package(s) and bump level (`:major` / `:minor` / `:patch`) so the release pipeline picks it up. Use `monorel add --package "<key>:<level>" --message "..."` or hand-roll the file. `CHANGELOG.md` files (root + per-package) are written from these by `monorel release`; do not edit them by hand.
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

The repo is multi-module: every transport, plugin, and integration ships as its own Go module with its own `CHANGELOG.md`. Both the root and per-package `CHANGELOG.md` files are written from `.changeset/*.md` files by `monorel release` at release time; do not edit them by hand. See `AGENTS.md` for the full versioning policy and the changeset workflow.
