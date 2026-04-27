# Code & Documentation Audit

A working list from a senior-Go-dev pass of the repo on 2026-04-27. Items
are prioritized by tier and tracked with `[ ]` (todo) / `[~]` (in progress)
/ `[x]` (done) / `[-]` (decided not to do, with reason).

Discuss and tackle gradually. Refer to items by number ("let's tackle #3").

---

## Tier 1 — substantive issues

### 1. `Fields`, `Data`, `Metadata` are all `= map[string]any` type aliases

- Status: `[ ]`
- Where: `types.go:8,12,17`
- Problem: documented as conceptually distinct but interchangeable to the
  type system. A user can pass `loglayer.Data` to `WithFields` and the
  compiler won't catch it. The doc reviewer flagged this as the most
  confusing terminology in the docs. Compounds with the `Plugin.OnBeforeDataOut`
  return type also being `Data`.
- Fix: drop the `=` to make them distinct named types
  (`type Fields map[string]any`). Small migration cost, removes a class of
  bugs, clarifies docs. Need to audit every transport/plugin call site to
  verify they use the correct type.

### 2. Constructor pattern is inconsistent across transports

- Status: `[ ]`
- Where: `transports/{http,datadog,otellog}/` (have `Build`) vs
  `transports/{zerolog,zap,slog,phuslu,logrus,charmlog}/` (no `Build`)
- Problem: Only network/SDK-bound transports got `Build`. Older wrapper
  transports panic on misconfig with no error-returning sibling. Pattern
  is accidental rather than principled.
- Fix: pick a rule and apply it uniformly. Either every transport gets a
  `Build` (even if it can't fail today) or document explicitly that
  `Build` exists only when there's runtime-loaded config. I lean toward
  the latter — uniform `Build` for non-failing constructors is just noise.

### 3. `phuslu` can't honor `DisableFatalExit`

- Status: `[ ]`
- Where: `transports/phuslu/phuslu.go`,
  `docs/src/transports/_partials/transport-list.md`
- Problem: the contract is "core decides exit." phuslu is the only
  transport that breaks it (upstream phuslu always calls `os.Exit` on
  Fatal). The warning is buried in the per-transport doc page; the
  partials list doesn't flag it. Newcomers picking a transport from the
  catalog won't notice.
- Fix: visible callout in the partials table (a row or icon for
  "exits-on-fatal-anyway"), or move phuslu to a "use only if you've read
  the caveats" subsection. Consider whether the phuslu transport earns
  its keep at all — it's the only one with this footgun.

### 4. `WithPrefix` directly mutates `child.config.Prefix`

- Status: `[ ]`
- Where: `loglayer.go:216`
- Problem: only place in the codebase that breaks the immutable-by-copy
  pattern (`assignedGroups`, `boundCtx`, atomic snapshots). Safe today
  because `Child()` returns a fresh logger, but the asymmetry will decay
  into a bug when a future refactor changes how `Child()` shares state.
- Fix: either deep-copy the `Config` in `Child()` (explicit, small cost),
  or wrap `Prefix` in the atomic-snapshot pattern used by transport sets.

### 5. CI tests only Go 1.26

- Status: `[ ]`
- Where: `.github/workflows/ci.yml:24` (`matrix.go: ['1.26']`)
- Problem: library is pre-1.0 and integrates with several major loggers.
  Single-version matrix can't detect Go-version compatibility regressions.
- Fix: add 1.24 and 1.25 to the matrix.

### 6. Pre-commit and CI enforcement diverge

- Status: `[ ]`
- Where: `lefthook.yml` (gofmt + vet) vs `.github/workflows/ci.yml`
  (also runs `staticcheck`)
- Problem: local commits pass clean and then CI fails on staticcheck.
  Same drift for tests: pre-push runs `-race ./...` but a developer who
  commits without pushing for a while never runs the suite locally.
- Fix: move `staticcheck` into `lefthook.yml` pre-commit. Decide whether
  to also run `go test -short ./...` pre-commit.

---

## Tier 2 — worth doing

### 7. `WithFields` and friends silently no-op on missing assignment

- Status: `[ ]`
- Where: `loglayer.go` / `fields.go` (every "returns new logger" method)
- Problem: `log.WithFields(...)` (without `log = `) is valid Go that
  silently discards. Documented but not enforced — the canonical Go
  footgun for this pattern.
- Fix: add a "Common Pitfalls" doc page that names this and the
  metadata-mutation trap. Code-level mitigation isn't realistic.

### 8. No coverage reporting in CI

- Status: `[ ]`
- Where: `.github/workflows/ci.yml`
- Problem: `go test` runs but nothing uploads coverage. Reviewers can't
  see if a PR drops coverage; no threshold enforcement.
- Fix: add a coverage step (`go test -coverprofile=...`) and upload to
  codecov or similar. Two-line change.

### 9. `panicError` loses typed-error inspection

- Status: `[ ]`
- Where: `plugin.go:184-189`
- Problem: when the recovered value isn't already an `error`, the
  fallback `fmt.Errorf("...: %v", r)` produces an untyped error, so
  `OnError` consumers can't `errors.As` to detect specific failure modes.
- Fix: define an unexported `panicValueError` struct that holds the
  recovered value and exposes it via `RecoveredValue() any`. Marginal — only
  matters if `OnError` becomes a real observability surface.

### 10. Documentation gaps

- Status: `[ ]`
- Where: `docs/src/`
- Problems (each is a candidate for its own doc page):
  - No "migrating from `log/slog`" / `zerolog` / `zap` migration guide.
    TypeScript users get a dedicated page; Go users coming from stdlib
    don't.
  - No "which transport should I use?" decision tree. The catalog lists
    them all but doesn't help readers pick.
  - No side-by-side wire-format comparison. `structured` flattens metadata;
    `zerolog` nests under a key. The inconsistency isn't called out
    anywhere. A single page with the same input run through every
    transport, showing JSON output, would be invaluable.
  - "Fields vs Metadata vs Data" trio is introduced casually but not
    consolidated. Compounds with #1.
  - Groups page is technically complete but the 8-rule routing precedence
    is intimidating; needs a worked multi-service example or flowchart.
  - "Common pitfalls" page (see #7).

---

## Tier 3 — style and nice-to-haves

### 11. `MetadataFieldName` is inconsistent across transports

- Status: `[ ]`
- Some transports have it, some don't, semantics differ slightly. Audit
  for uniformity.

### 12. No goroutine-leak detection in tests

- Status: `[ ]`
- Add a `goleak.VerifyNone` (or hand-rolled equivalent) in a `TestMain`
  to catch transport-side leaks.

### 13. Trace-level mapping table

- Status: `[ ]`
- zap/slog/charmlog collapse Trace to Debug; zerolog/phuslu/logrus
  preserve it. Documented per-transport but no single comparison table
  on the transports overview.

### 14. Examples coverage gaps

- Status: `[ ]`
- The new transports/plugins (otellog, oteltrace, datadogtrace) have docs
  but no `examples/` entry. Plugin authoring also has no runnable example.

### 15. Test-setup duplication acknowledged but kept

- Status: `[-]` decided not to address
- `setup(t, plugin)` exists in 4 test files. Lifting it across module
  boundaries (datadogtrace livetest is a separate go.mod) would create
  more friction than it saves. Two prior `/simplify` passes converged on
  the same conclusion.

---

## Done from this audit

(empty for now; move items here as we tackle them)
