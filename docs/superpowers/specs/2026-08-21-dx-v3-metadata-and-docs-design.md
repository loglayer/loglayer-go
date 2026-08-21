# DX v3: uniform metadata nesting + docs sweep

Date: 2026-08-21
Status: approved design (brainstorming session)

## Problem

Feedback from the hmn-cli migration (LOGLAYER.md, four open concerns) plus two
code gaps found during design:

1. **#7 Metadata shape depends on the value type and the transport class.**
   With `Config.MetadataFieldName` empty (the v2 default), map metadata flattens
   to root attributes in every transport, while non-map metadata (structs,
   scalars) nests under a hardcoded `"metadata"` key in wrapper transports and
   JSON-roundtrips to root in renderers. Same `WithMetadata` call, different
   JSON shape depending on what type you pass and which transport renders it.
2. **ANSI gap.** `transports/structured` writes message and metadata values
   through `encoding/json` directly, which escapes C0 controls (ESC becomes
   ``, CR/LF escaped) but passes bidi and zero-width characters through
   raw. The terminal renderers (console/pretty/cli) sanitize those; structured
   does not.
3. **#2 New-vs-Build discoverability.** `Build` is the right constructor for
   runtime config, but the docs present `New` as primary and `Build` as an
   afterthought (the hmn complaint).
4. **#3 Fatal exit foot-gun.** Default stays `os.Exit(1)` (matches
   log.Fatal/zerolog/zap). Docs need to make the `DisableFatalExit` escape
   hatch and the shutdown-flush behavior prominent and discoverable.
5. **#6 Transport IDs discoverability.** Setting an ID requires
   `transport.BaseConfig{ID: ...}` plus an import; nothing on the
   `*LogLayer` API or the transport pages makes that discoverable before
   management methods (`AddTransport` / `RemoveTransport`) silently misbehave.

Decisions from the brainstorming session:

- Ship the metadata default flip as a **v3 core** (`go.loglayer.dev/v3`).
- Keep the `Fatal` exit default; harden docs only.
- No new `KV()` method; `MetadataOnly` is the KV idiom; docs say so.
- **#5 F/M aliases and #10 empty-msg/KV are out of scope** for this design.
- Every sub-module that imports the core moves with it to v3.

## Goals / non-goals

Goals:

- Uniform metadata placement across value types and transport classes by
  default, with a one-line opt-out for the v2 shape.
- ANSI/bidi sanitization in structured output.
- A docs sweep that is mechanically checked for accuracy (each example
  compiles against the post-change code, each referenced symbol exists).
- v3 path migration for the whole module tree.

Non-goals:

- Removing the F/M aliases (out of scope per decision).
- A `KV()` first-class method (out of scope).
- Changing `New` / `Build` signatures (docs-only) — but we make `Build` the
  documented default for runtime-config construction.
- Changing `DisableFatalExit` semantics.

## Design

### Core v3: uniform metadata nesting (breaking default)

`Config.MetadataFieldName` keeps its semantics: when non-empty, the entry's
metadata nests under that key uniformly (map and non-map) across every
transport. The v2 default (`""`) meant "transport decides": renderers flatten,
wrappers type-depend. The v3 flip:

- **New default**: when `MetadataFieldName == ""`, `build()` resolves it to
  `"metadata"` so every transport's existing `if key != ""` branch nests
  uniformly. This is a **breaking behavior change** (existing code that relied
  on map flattening at root changes its JSON schema).
- **New escape hatch**: `Config.FlattenMetadata bool`. When true, `build()`
  leaves `MetadataFieldName` empty (v2 shape). Ignored when
  `MetadataFieldName` is explicitly set (explicit key always wins).
- `Schema.MetadataFieldName` published on `TransportParams` reflects the
  resolved value (after defaulting), so transports need **zero code changes**
  — their existing `key != ""` polarity already implements the new default.

Migration: any caller that wants v2 output sets
`FlattenMetadata: true` (one line). The design doc for this change lives in
`docs/src/migrating-to-v3.md`.

### Core v3: `Child()` propagation

`Child()` copies config wholesale, so the resolved `MetadataFieldName`
propagates. `FlattenMetadata` is part of the config copy. Add a test that a
child logger nests metadata identically to its parent.

### Structured v3: ANSI sanitization + empty-msg omission

`transports/structured` (go.loglayer.dev/transports/structured/v3):

1. **Sanitize at the top level** before JSON-encoding: join the message and
   run `sanitize.Message`; run `sanitize.Message` on metadata and Data keys
   and on string-typed top-level values before `writeKeyValue`. This closes
   the ESC/bidi/CRLF hole for the values structured actually renders. Deep
   struct fields (rendered by the underlying JSON encoder, not by us) remain
   the encoder's domain; document that limit (it matches console/pretty/cli).
2. **Omit empty `msg`**: when the joined message is empty (`Info("")`), the
   structured transport omits the message field entirely, so
   `WithFields(...)` + `Info("")` renders
   `{"level":"info","time":...,"fields...}` with no `msg` key. `msg:""` was a
   wart for fields-only callers (LOGLAYER.md #10). `MetadataOnly` remains the
   documented KV idiom; this only cleans up the empty-string case.

### Docs sweep (accuracy-checked)

A dedicated sweep over the docs that reference the changed surfaces. Mechanic:
every Go example is compiled (or at minimum symbol-checked) against the
post-change tree; every heading link and cross-reference is checked; the
`_partials/` includes render. Concretely the sweep covers:

- `docs/src/migrating-to-v3.md` (new): the v3 story, `FlattenMetadata` opt-out,
  the module-path migration for the whole tree.
- `docs/src/configuration.md`: `MetadataFieldName` + `FlattenMetadata` rows;
  `Build` vs `New` guidance; `DisableFatalExit` note.
- `docs/src/logging-api/metadata.md`: uniform-nesting rule + `FlattenMetadata`
  opt-out + pointer to `MetadataOnly` for KV.
- `docs/src/logging-api/mocking.md` (or wherever `NewMock` / capture pattern
  lives): the `NewMockWithWriter`-style capture pattern note; confirm the
  existing `::: tip` block already covers it.
- `docs/src/logging-api/basic-logging.md`: `Fatal` + `os.Exit` + deffered-flush
  note; `New` vs `Build` pointer.
- `docs/src/transports/*.md` (all transport pages): `MetadataFieldName` default
  statements; `Fatal Behavior` sections; anything that pins the old
  flatten-at-root shape.
- `docs/src/transports/_partials/transport-list.md`: nothing changes here (no
  new transports), but re-check the catalog.
- `docs/src/whats-new.md`: v3 date-section entry.
- `docs/src/public/llms.txt` + `llms-full.txt`: new surface entries.
- `docs/src/cheatsheet.md`: new config fields on the quick reference.

Accuracy mechanics: for each Go code block, `go doc` / symbol-exists check on
the referenced names; for each link, verify the target path + fragment exists
in the built site (`bun run docs:build` and grep the `dist` index); for each
config table row, verify the field name against `types.go`.

### Module path migration (the v3 sweep)

Per AGENTS.md multi-module policy: every module that imports
`go.loglayer.dev/v2` moves to `go.loglayer.dev/v3`:

- Core: `module go.loglayer.dev/v3`, `go.mod` bump, `monorel.toml`
  `[packages."go.loglayer.dev"]` gets the v3 path.
- `transport/` (go.loglayer.dev/transport): shared helpers; does it import the
  core? It imports `go.loglayer.dev/v2` types (TransportParams,
  loglayer.Metadata). So it bumps to v3 too.
- `utils/*` (sanitize, idgen, maputil): same — those import core types; bump.
- Every sub-module that imports `go.loglayer.dev/v2` (or `transport` /
  `utils`): bump its go.mod `require go.loglayer.dev/v3` (and the matching
  `replace`). Their own majors only change if their *own* API breaks
  (wrapper transports that re-export `TransportParams` types do; renderers
  usually don't).
- `go.work` gets the v3 `use` entries for every bumped module.
- `scripts/foreach-module.sh` module lists.
- `transports/structured` bumps to `v3` because it re-exports
  `loglayer.TransportParams` in its API (and gets the sanitize/msg changes).
- `internal/lltest`, `examples/`, `bench_test.go`, all tests: import-path
  updates.
- `monorel.toml` change blocks for the bumped packages.

This is the bulk of the mechanical work. CI runs `scripts/foreach-module.sh`
so the sweep is verifiable locally.

### Out of scope (explicitly)

- F/M aliases (#5): keep as-is.
- KV() method (#10): `MetadataOnly` is the KV idiom.
- `New` / `Build` signature changes: docs-only.
- `DisableFatalExit` default flip: docs-only hardening.

## Testing

- Core: unit tests for the `MetadataFieldName` defaulting (`build()`),
  `FlattenMetadata` opt-out, `Child()` propagation, and the
  `Schema` value on `TransportParams`.
- Structured: sanitization tests (ESC, bidi, CR/LF in message, metadata keys,
  string values) and empty-msg omission test.
- `transporttest.RunContract` (the 14-test shared contract): update any assertion
  that pins `msg` presence/absence or exact metadata placement.
- Docs: `cd docs && bun run docs:build` clean; a mechanical example-accuracy
  check (compile each Go block against the post-change tree where feasible).
- Full tree: `scripts/foreach-module.sh test` (CI parity).

## Risks / open items

- The multi-module v3 sweep is large and mechanical; doing it in one PR keeps
  the tree consistent (buildable at every commit). Doing it in stages leaves
  the tree in a mixed v2/v3 state.
- The wrapper transports' `event.Fields` flatten path still exists for
  `FlattenMetadata: true` users; that path is now opt-in, so its behavior
  needs one contract test to avoid regressions for that cohort.
- The `MetadataFieldName == ""` + `FlattenMetadata == true` case is the v2
  exact shape; document it as "the v2 compatibility mode".

## Status

Approved by user (Theo) in brainstorming session. Awaiting implementation
plan (writing-plans).
