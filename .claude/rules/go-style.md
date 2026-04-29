# Go Style Rules

Project-specific Go conventions. General Go style (effective Go, gofmt, golint) is assumed; this file only records patterns that came up in this codebase.

## Naming

### Avoid stdlib name collisions

Don't pick exported identifiers that shadow widely-used stdlib types or packages, even if our type is technically distinct. Users will read it as the stdlib type.

The persistent key/value bag was originally called `loglayer.Context` (a `map[string]any`). Every reader assumed it was related to `context.Context`. We renamed it to `loglayer.Fields`. Same shape, no collision.

Watch for: `Context`, `Error` (use `Err` for fields), `Request`, `Response`, `Reader`, `Writer`, `Time`, `Map`, `String` (these all have stdlib meanings or are interface names users expect).

If the natural name collides, prefix with the package's role (`LogLevel`, not `Level`) or pick a synonym (`Fields`, not `Context`).

### Package name vs import path

When the import path's last element collides with a stdlib package, the package name diverges from the directory. Example: `transports/charmlog/` (directory) but `package charmlog` (avoids clashing with stdlib `log`). Document this at the top of the package's main file with a one-line comment so users importing it understand why.

## API Surface

### Three-way data separation

There are three distinct ways to attach data to a log call. Don't blur them.

| API | Type | Scope | Use for |
|-----|------|-------|---------|
| `WithFields(loglayer.Fields)` | `map[string]any` | Persistent on the logger | Request IDs, user info, anything that should appear on every subsequent log |
| `WithMetadata(any)` | any value | Single log call only | Per-event payload — counters, durations, structs |
| `WithContext(context.Context)` | `context.Context` | Single log call only | Trace IDs, deadlines, cancellation; for transports that forward to OTel/slog |

When adding new logging surface, fit it into one of these three. Do not introduce a fourth bag.

### Concurrent-safety classes

Every method on `*LogLayer` falls into one of three categories. Document which in the GoDoc comment.

1. **Read-only**: emission methods (`Info`, `WithMetadata`, `Raw`, ...), getters. Always safe.
2. **Returns-new**: `WithFields`, `WithoutFields`, `Child`, `WithPrefix`. Always safe; receiver untouched.
3. **Atomic-mutate**: level mutators (atomic.Uint32 bitmap), transport mutators (atomic.Pointer + mutex on the slow path). Always safe.
4. **Setup-only mutate-in-place**: mute toggles. NOT safe concurrently with emission. GoDoc must say `// Setup-only:` explicitly.

Default to (2) or (3) for new methods. Use (4) only when the operation is genuinely admin-rare and the cost of going atomic isn't justified. If you add a setup-only method, mark it in GoDoc and add a note in AGENTS.md "Thread Safety" so the contract stays comprehensive.

`concurrency_test.go` covers each class with `-race` runs, including a runtime-level-toggle test (validates atomic levels) and a transport hot-swap test (validates atomic transports). Add to that file when you ship a new method that needs concurrency coverage.

### Constructor pair: `New` panics, `Build` returns error

`loglayer.New(Config)` panics on misconfiguration; `loglayer.Build(Config) (*LogLayer, error)` returns an error. Both report `loglayer.ErrNoTransport`. When adding a constructor that can fail at construction time, follow the same pattern: a panicking primary entry point and a `Build`-style sibling, not just one or the other.

**The rule is "Build exists when the constructor can fail."** Wrapper transports that take a pre-built `*zerolog.Logger`/`*zap.Logger`/etc. and have nothing to validate ship only `New`. Network transports (`http`, `datadog`) and SDK-binding transports (`otellog`) that need a URL/API key/instrumentation name have both `New` and `Build`, plus a sentinel error in `errors.go` (e.g. `ErrURLRequired`, `ErrAPIKeyRequired`, `ErrNameRequired`) callers compare with `errors.Is`. Keep these sentinels named `ErrXRequired` for consistency.

When you add a new transport, ask: "can this constructor fail with a runtime-loaded value (env var, config file, secret manager)?" If yes, ship the pair. If no, `New` alone is honest — adding a `Build` that always returns nil error is just noise.

## Code Reuse

### Lift duplicated defaulting into `transport/`

When N transports share the same boilerplate (config defaulting, conversion, setup), put the helper in `transport/helpers.go` and have each transport import it. Three concrete examples already in there:

- `transport.JoinMessages([]any) string`: assembling the message string with a single-string fast path.
- `transport.MetadataAsMap(any) map[string]any`: collapsing arbitrary metadata via JSON roundtrip.
- `transport.WriterOrStderr(io.Writer) / WriterOrStdout(io.Writer)`: returning a default writer when the caller didn't supply one.

The `WriterOrStderr` extraction was triggered when 5 wrapper transports each had `if w == nil { w = os.Stderr }`. Threshold heuristic: **3+ verbatim copies** of the same 4+ line pattern is the moment to lift.

**Don't lift if the policy diverges.** `transports/console` picks stdout vs stderr per log level (debug→stdout, error→stderr). It does not use `WriterOrStderr` and shouldn't, even though the function names look applicable. Same shape, different intent.

### Don't generalize across transports past the shared helpers

Every wrapper transport has a roughly cookie-cutter `Config { BaseConfig; Logger; Writer; MetadataFieldName }` shape. Don't try to abstract this into a generic via type parameters or interface. The destination logger types (`*zap.Logger`, `*zerolog.Logger`, `*slog.Logger`) have no common interface, and the level mappings are intentionally per-library. The repetition is honest; abstraction would just move it.

## File Organization

### Split by responsibility, not by line count alone

A 400+ LOC file that does one thing is fine. A 200 LOC file that mixes two unrelated concerns is not.

The pretty transport was split when `pretty.go` reached 407 LOC because it mixed:

- **Setup and dispatch**: Config, Transport struct, New, SendToLogger, header rendering (timestamp/level/logID).
- **Data rendering**: combineData, inline format, expanded format, scalar formatters.

Those are two responsibilities with nearly disjoint helper functions, so they became `pretty.go` (config + dispatch) and `render.go` (data formatting). Themes were already in `themes.go`.

**Soft signal to look for a split**: a file approaches ~300 LOC AND you can name two distinct responsibilities AND the helper functions cluster cleanly into two non-overlapping sets. If any of those three is missing, leave it alone.

### Co-locate by package, split by file

Don't fragment a single concept across packages just because the file got large. Within a package, splitting into multiple files is cheap (Go treats them as one compilation unit) and doesn't affect callers. Across packages it's a public API decision that affects every importer.

The core split (`loglayer.go`, `dispatch.go`, `fields.go`, `levels.go`, `transports.go`, `log.go`, `errors.go`, `from_context.go`, `builder.go`) all live in `package loglayer`. Callers see one package; maintainers see focused files.

## Performance

### Don't fight the compiler

Recent Go (1.22+) inlines aggressively and stack-allocates small structs. Before adding `sync.Pool`, manual escape-analysis tricks, or other allocation-elimination strategies:

1. Measure baseline with `-benchmem`.
2. Apply the change.
3. Measure again with `benchstat` (10 runs).
4. If allocs/op didn't decrease, revert.

The history of attempted-and-reverted perf changes lives in `AGENTS.md` under "Performance: Attempted and Rejected". Read it before reinventing.

### Pointer to embedded config in hot paths

In `processLog` we do `cfg := &l.config` instead of value-copying the `Config` struct. This is a measured ~5% win on `BenchmarkLoglayer_SimpleMessage` because `Config` carries function pointers. Apply the same pattern in any other hot path that would otherwise copy a struct with embedded function fields.

## Errors

### Sentinel errors over typed errors

Public errors are sentinels declared in `errors.go`: `var ErrNoTransport = errors.New(...)`. Callers compare with `errors.Is`. Don't introduce error types unless the caller genuinely needs to extract structured data from the error.

### Don't validate at internal boundaries

Validate at the public API boundary (`New`, `Build`, transport constructors), then trust the input through internal code. Don't re-check `nil` on every internal hop. The dispatch path assumes its inputs are valid.

## Comments

### Setup-only methods get a one-line marker

```go
// WithFields merges fields into the logger's persistent bag.
//
// Setup-only: mutates l.fields without locking.
func (l *LogLayer) WithFields(fields Fields) *LogLayer { ... }
```

### Don't narrate the change

No comments referring to "added for X", "used by Y", "renamed from Z". Those belong in commit messages and CHANGELOG.md, not source.

### Justify non-obvious choices

A comment like `// fields is read concurrently; only mutate at setup` is keeper because the why is non-obvious. A comment like `// loop over transports` is noise.
