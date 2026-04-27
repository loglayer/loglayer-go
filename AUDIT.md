# Code & Documentation Audit

A working list from a senior-Go-dev pass of the repo on 2026-04-27. Items
are prioritized by tier and tracked with `[ ]` (todo) / `[~]` (in progress)
/ `[x]` (done) / `[-]` (decided not to do, with reason).

Discuss and tackle gradually. Refer to items by number ("let's tackle #3").

---

## Tier 1 — substantive issues

### 1. `Fields`, `Data`, `Metadata` are all `= map[string]any` type aliases

- Status: `[x]` done — see commit `83d8055`. Distinct named types now;
  `transport.MetadataAsRootMap` helper handles the dual-case detection
  in transports. No user-visible breakage in the repo's own tests.

### 2. Constructor pattern is inconsistent across transports

- Status: `[x]` done — see commit `83d8055`. Documented the rule in
  `.claude/rules/go-style.md`: "Build exists when the constructor can
  fail." Wrapper transports that take a pre-built logger ship only `New`;
  network/SDK-bound transports ship the pair. Sentinels named
  `Err<X>Required`.

### 3. `phuslu` can't honor `DisableFatalExit`

- Status: `[ ]` not yet addressed.
- Where: `transports/phuslu/phuslu.go`,
  `docs/src/transports/_partials/transport-list.md`
- Problem: the contract is "core decides exit." phuslu is the only
  transport that breaks it. The warning is buried.
- Fix: visible callout in the partials table (a row or icon for
  "exits-on-fatal-anyway"). Surfaced once now in the level-mapping
  blurb on `transports/index.md` (in commit `fc36146`) but the
  catalog row itself doesn't flag it.

### 4. `WithPrefix` directly mutates `child.config.Prefix`

- Status: `[x]` done — see commit `83d8055`. Hoisted `Prefix` to a
  top-level `LogLayer.prefix` field alongside fields/boundCtx/
  assignedGroups. Same lifecycle: set on a fresh child before publish,
  never mutated post-publish.

### 5. CI tests only Go 1.26

- Status: `[x]` done — see commit `83d8055`. CI matrix tests Go 1.25
  and 1.26 (the actual floor turned out to be 1.25, driven by
  `golang.org/x/exp` via `charmbracelet/log` and `golang.org/x/sys`,
  not by OTel as originally suspected).

### 6. Pre-commit and CI enforcement diverge

- Status: `[x]` done — see commit `83d8055`. `lefthook.yml` now runs
  `staticcheck` pre-commit (skips with a hint when the binary isn't
  installed). AGENTS.md updated.

---

## Tier 2 — worth doing

### 7. `WithFields` and friends silently no-op on missing assignment

- Status: `[x]` done — see commit `fc36146`. New `/common-pitfalls`
  doc page covers this and several other footguns (mutating maps after
  binding, plugin panics requiring OnError, fatal-exit divergence,
  ctx-binding traps, group-routing surprises).

### 8. No coverage reporting in CI

- Status: `[ ]` deferred (user did not request).
- Two-line addition: `go test -coverprofile=...` + codecov upload.

### 9. `panicError` loses typed-error inspection

- Status: `[x]` done — see commit `83d8055`. New `RecoveredPanicError`
  type with `Hook`, `Value`, and `Unwrap()` so `OnError` consumers can
  `errors.As` to it and reach the original panic value (whether it
  was an `error` or any other type).

### 10. Documentation gaps

- Status: `[x]` mostly done.
  - `[x]` 10a: Migration guides for slog, zerolog, zap (commit pending).
  - `[x]` 10b: Transport selection decision tree on `transports/index.md`.
  - `[x]` 10c: Wire format comparison — re-scoped per user feedback.
    Replaced "growing master table" approach with brief callout
    pointing readers to per-transport doc pages.
  - `[x]` 10d: Fields/Metadata/Data terminology consolidated in
    `/concepts/data-shapes`. Cross-linked from `fields.md` and
    `metadata.md`.
  - `[x]` 10e: Groups worked example — multi-service routing scenario
    with concrete table of what each call dispatches to.

---

## Tier 3 — style and nice-to-haves

### 11. `MetadataFieldName` is inconsistent across transports

- Status: `[x]` done — audit verified all 7 wrapper transports
  (slog/zerolog/zap/phuslu/logrus/charmlog/otellog) are already
  consistent. No code change needed.

### 12. No goroutine-leak detection in tests

- Status: `[x]` done — `go.uber.org/goleak.VerifyTestMain` added to
  the root package, `transports/http`, and `transports/datadog`.
  Catches transport-side goroutine leaks (the HTTP/Datadog workers).

### 13. Trace-level mapping comparison table

- Status: `[x]` done — initial big table reverted per user feedback
  (would grow with every new transport). Replaced with a brief
  deviation callout on `transports/index.md` listing only the cases
  that differ from straight pass-through (Trace→Debug in
  zap/slog/charmlog; phuslu always exits on Fatal).

### 14. Examples coverage gaps

- Status: `[x]` done.
  - `examples/custom-plugin/`: from-scratch plugin demo covering
    `OnBeforeDataOut`, `OnMetadataCalled`, and `ShouldSend`.
  - `examples/otel-end-to-end/`: combined `transports/otellog` +
    `plugins/oteltrace` demo against a real OTel SDK with stdout
    exporters. Lives in its own go.mod (mirroring the OTel split).
  - Skipped a separate `datadogtrace` example: the per-page docs and
    the livetest module already cover the documented pattern; an
    additional example would just duplicate.

### 15. Test-setup duplication acknowledged but kept

- Status: `[-]` decided not to address.
- `setup(t, plugin)` exists in 4 test files. Lifting it across module
  boundaries (datadogtrace livetest is a separate go.mod, otellog and
  oteltrace are now also separate go.mods) would create more friction
  than it saves. Two prior `/simplify` passes converged on the same
  conclusion.

---

## Module-structure work (not in original audit)

- Status: `[x]` done — see commit `ba3bfdc`.
- Split `transports/otellog` and `plugins/oteltrace` into their own Go
  modules so the OpenTelemetry SDK's transitive deps don't bind users
  who don't import OTel. Main module's go.sum dropped from 85 → 60
  lines. The 1.25 floor itself didn't drop because other deps
  (`golang.org/x/exp`, `golang.org/x/sys`) still demand it; the user
  decided not to chase a lower floor through more dep surgery.
- Documented the splitting policy and per-module Go-version handling
  in `AGENTS.md`, `.claude/rules/documentation.md`, the README, the
  partial lists, and per-page docs.

---

## Done from this audit

- `[x]` #1 — distinct Fields/Data/Metadata named types
- `[x]` #2 — Build constructor rule documented in coding style
- `[x]` #4 — WithPrefix mutation removed
- `[x]` #5 — CI matrix on 1.25 + 1.26
- `[x]` #6 — staticcheck in lefthook for CI parity
- `[x]` #7 — `/common-pitfalls` doc page
- `[x]` #9 — `RecoveredPanicError` typed inspection
- `[x]` #10a — three migration guides
- `[x]` #10b — transport selection guide
- `[x]` #10c — wire-format pointer
- `[x]` #10d — Fields/Metadata/Data concept page
- `[x]` #10e — groups worked example
- `[x]` #11 — MetadataFieldName audit (was already consistent)
- `[x]` #12 — goleak in tests
- `[x]` #13 — level-mapping deviation callout
- `[x]` #14 — examples for new transports/plugins + plugin authoring
- `[x]` Module split — OTel pieces in their own modules

## Still open

- `[ ]` #3 — phuslu fatal-exit visibility in the catalog table.
- `[ ]` #8 — coverage reporting in CI (deferred).

---

## Round 2 audit (2026-04-27)

### Tier 1 — contract tightening

- `[x]` #A1 — Transport error-handling contract documented in
  `creating-transports.md` (sync renderers print to stderr; async
  transports expose `OnError`; wrappers forward through the underlying
  library).
- `[x]` #A2 — Documented why `OnFieldsCalled`/`OnMetadataCalled` don't
  receive ctx (chain order is non-deterministic, ctx-aware behavior
  belongs in dispatch-time hooks).
- `[x]` #A3 — Added "Testing your plugin" / "Testing your transport"
  sections to creating-plugins.md / creating-transports.md pointing at
  `transports/testing` as the canonical capture path. Decided NOT to
  promote `internal/transporttest` (only 20 lines of JSON-specific
  helpers; `transports/testing` covers the broader case).

### Tier 2 — correctness + coverage

- `[x]` #A4 — `dispatch_edge_test.go` covers all-ShouldSend-false drops
  everywhere, disabled-group does not fall through (vs undefined
  group does), ErrorSerializer returning nil drops the err key
  entirely (was a real fix: was previously storing nil under err),
  multiple TransformLogLevel plugins (last ok=true wins).
- `[x]` #A5 — `TestConcurrentPluginMutation` in concurrency_test.go
  exercises `AddPlugin`/`RemovePlugin` against concurrent emission
  with all hook types firing.
- `[x]` #A6 — `loghttp.Config.ShouldStartLog` lets services sample
  start-line emission per-request (e.g. only when X-Debug header
  present, or for 1% of traffic).
- `[x]` #A7 — added `BenchmarkLoglayer_WithError_CustomSerializer` and
  `BenchmarkLoglayer_PluginPipeline` to bench_test.go.

### Tier 3 — polish

- `[x]` #A8 — Cheatsheet calls out `MetadataOnly`/`ErrorOnly` are
  terminal, not builders.
- `[x]` #A9 — `maputil.Cloner` doc names the canonical use case (redact
  plugin) so future readers don't think it's load-bearing infra.
- `[x]` #A10 — Cheatsheet links to creating-plugins.md, examples/,
  and the eight-rule routing precedence on the groups page.

### Surprise finding (fixed, not in original audit)

- ErrorSerializer returning nil used to store `data["err"] = nil`
  rather than dropping the key. Now drops the key entirely, matching
  plugin-hook nil-drop semantics. Documented on the type.
