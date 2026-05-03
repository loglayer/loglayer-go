# `loglayer.Multiline`: developer-authored multi-line messages

**Date:** 2026-05-02
**Status:** Approved (ready for implementation plan)
**Scope:** Single change in the root `go.loglayer.dev` module, plus call-site swaps in the in-tree terminal-shaped transports and the `loghttp` integration.

## Motivation

The terminal-shaped transports (`cli`, `pretty`, `console`) and the `loghttp` integration run every message string through `utils/sanitize.Message`, which drops control bytes including `\n`. The threat model is two-fold: untrusted log input must not be able to (1) forge follow-up log lines via embedded `\n` or (2) smuggle ANSI / bidi / zero-width terminal escapes. JSON-shaped sinks (`structured` plus every wrapper transport) skip this because `encoding/json` already escapes control bytes.

The result today: any genuine multi-line content authored by the developer is collapsed when it reaches a terminal renderer. Indented configuration dumps, multi-line CLI hints, list-shaped status output, and stack-trace-style content all render as a single line.

The fix is a developer-issued *token of trust*: a typed wrapper the developer constructs explicitly, that signals "I authored these line boundaries." The wrapper rides through dispatch unchanged. Terminal transports detect the type and permit `\n` between authored elements while still sanitizing escapes inside each line. Untrusted input can never become multi-line by accident — it has to be hand-packed into the wrapper's constructor.

## Public surface

A new constructor in the root `loglayer` package, sibling of `Lazy`:

```go
log.Info(loglayer.Multiline("Header:", "  port: 8080", "  host: ::1"))
log.Warn("ignoring", loglayer.Multiline("/etc/foo.conf", "/etc/bar.conf"))
```

```go
// MultilineMessage wraps a sequence of authored lines so terminal
// transports render them on separate rows. Construct with [Multiline].
//
// Token of trust: the wrapper signals that the developer authored the
// line boundaries, so terminal renderers permit \n between elements
// while still sanitizing ANSI / control bytes within each line.
type MultilineMessage struct {
    lines []string
}

// Multiline wraps lines so terminal transports render them on separate
// rows. Each argument is converted to a string via fmt.Sprintf("%v", v)
// for non-string types. Nested *MultilineMessage arguments are flattened
// at construction so a wrapper's Lines() never contains embedded "\n".
func Multiline(lines ...any) *MultilineMessage

// Lines returns the authored line list. Transport authors call this
// when rendering each line independently.
func (m *MultilineMessage) Lines() []string

// String joins the lines with "\n". Used by the fmt.Stringer fallback
// path in transports that don't special-case the type (JSON sinks and
// every wrapper transport).
func (m *MultilineMessage) String() string
```

The wrapper is read-only after construction. No locks; concurrency story is identical to `*LazyValue`.

## Architecture

The wrapper is **not** flattened in core. It rides through `params.Messages` as one element of `[]any`. Each transport interprets it.

This is the load-bearing decision. If we flattened to a `\n`-joined string in core or in `JoinMessages`, the sanitizer in cli/pretty/console would strip the `\n` back out — the transport wouldn't know it came from authored content. Keeping the typed value intact through dispatch is what makes the trust signal survive to the rendering boundary.

### No dispatch changes

Core's `processLog`, the plugin pipeline, level filtering, group routing, and the `*LogLayer` thread-safety contract are all unchanged. `*MultilineMessage` is just another value type that can appear in `Messages`, alongside `string`, `int`, `*LazyValue`, etc.

Plugin hooks with a Messages signature see the wrapper as-is. Hooks that just want a flat string can call `transport.JoinMessages(params.Messages)` and let `Stringer` flatten transparently. Hooks that want to mutate per-line can type-assert to `*MultilineMessage` and call `Lines()`. Most hooks won't need to care.

### Two paths for transport authors

**JSON sinks and wrapper transports** (`structured`, `zerolog`, `zap`, `slog`, `logrus`, `charmlog`, `phuslu`): zero code change. They call `transport.JoinMessages(params.Messages)`, which falls through to `fmt.Sprintf("%v", v)` for non-string elements, which calls `*MultilineMessage.String()`, which joins with `\n`. The underlying logger writes the joined string; JSON encoders escape `\n` to `\\n`; non-JSON wrappers just hand the string to the writer. No threat from `\n` in these sinks because they don't render to a TTY.

**Sanitizing terminal transports** (`cli`, `pretty`, `console`, `loghttp`): a new helper, `transport.AssembleMessage`, replaces the existing per-transport sanitize-then-join (or join-then-sanitize) pattern.

### `transport.AssembleMessage`

```go
// AssembleMessage flattens a message slice into a single string,
// applying sanitize to every authored chunk while preserving line
// boundaries inside *MultilineMessage values.
//
// For each element in messages:
//   - string s              -> sanitize(s)
//   - *MultilineMessage m   -> per-line sanitize, joined with "\n"
//   - any other v           -> sanitize(fmt.Sprintf("%v", v))
//
// Adjacent elements are joined with " ". Nil messages are skipped.
//
// Used by terminal-style transports (cli, pretty, console) and the
// loghttp integration. Wrapper transports and JSON sinks call
// JoinMessages instead; *MultilineMessage's String method handles
// flattening transparently for them.
func AssembleMessage(messages []any, sanitize func(string) string) string
```

The CLI transport's existing `sanitizeMessages` private helper is removed; `AssembleMessage` covers its job, narrower-cased.

## Per-transport call-site swaps

| Caller | Today | After |
|---|---|---|
| `transports/cli/cli.go:256` | `transport.JoinMessages(sanitizeMessages(params.Messages))` | `transport.AssembleMessage(params.Messages, sanitize.Message)` |
| `transports/pretty/pretty.go:127` | `sanitize.Message(transport.JoinMessages(params.Messages))` | `transport.AssembleMessage(params.Messages, sanitize.Message)` |
| `transports/console/console.go:118` | per-element sanitize loop, then `JoinMessages` | `transport.AssembleMessage(params.Messages, sanitize.Message)` |
| `integrations/loghttp` | inline sanitize at the message-build site | `transport.AssembleMessage` at the same site |

No other behavior changes in the call sites — color, level prefixes, table rendering, expanded YAML, etc. all keep working. Each rendered line tints with the level color in cli (same as today's single-line body); pretty and console render the multi-line block as one body where their existing layout already works.

## Rendering by transport class

| Transport | `Multiline("a","b")` alone | Mixed: `"Header:", Multiline("a","b")` |
|---|---|---|
| **cli** | `a\nb` (level-colored, on the level's writer) | `Header: a\nb` |
| **pretty** | `[ts] [INFO] a\nb` | `[ts] [INFO] Header: a\nb` |
| **console** (JSON) | `{"msg":"a\nb",...}` | `{"msg":"Header: a\nb",...}` |
| **structured** | `{"msg":"a\nb",...}` | `{"msg":"Header: a\nb",...}` |
| **zerolog/zap/slog/logrus/charmlog/phuslu** | underlying logger receives `"a\nb"` | underlying logger receives `"Header: a\nb"` |
| **loghttp** | `a\nb` in the rendered headline | `Header: a\nb` |

## Edge cases (explicit decisions)

| Case | Behavior | Rationale |
|---|---|---|
| `Multiline()` (zero args) | Empty string. Transport's existing "empty body skip" logic handles whether to print at all. | Symmetric with `log.Info("")`. |
| `Multiline("")` (one empty arg) | Empty string (one empty line, no `\n`). | A single line, even an empty one, isn't multi-line. |
| `Multiline("a", "")` (trailing empty) | `"a\n"` — preserves the authored boundary. | The developer asked for it. |
| `Multiline(Multiline("a","b"), "c")` (nested) | `"a\nb\nc"`. The constructor flattens nested wrappers so `Lines()` == `["a","b","c"]`. | Without flattening, JSON sinks would render `"a\nb\nc"` (via `Stringer`) but terminal sinks would render `"ab\nc"` (per-line sanitize would strip the inner `\n`). Flattening at construction makes both paths agree. |
| `nil` element (`Multiline("a", nil, "b")`) | `"a\n<nil>\nb"` via `fmt.Sprintf("%v", nil)`. | Matches today's `JoinMessages` behavior. |
| `*MultilineMessage` inside `Fields` or `Metadata` | Untouched by `AssembleMessage`. CLI/pretty/console still sanitize via `writeValue` / cell-render and strip `\n`. | Documented v1 gap. |

## Threat model preserved

`sanitize.Message` runs unchanged on every authored chunk:

- A bare `string` element with embedded `\n` (e.g., from untrusted user input) still has the `\n` stripped. No wrapper, no trust.
- Each line *inside* a `Multiline` is still sanitized, so ANSI ESC, CR, bidi controls (U+202E "Trojan Source"), and zero-width joiners are still stripped. The wrapper does not unlock escape smuggling.
- The only thing the wrapper unlocks is `\n` *between* the authored elements.

Regression-guard tests confirm this in cli/pretty/console (a bare `"a\nb"` string still renders as one line; `Multiline("a","b")` renders as two).

## v1 scope: messages only

`Multiline` only applies when it appears in the message slice. It is **not** honored inside `Fields` or `Metadata`.

The argument for extending: CLI's `ShowFields=true` mode also sanitizes field values via `writeValue`, so a stack-trace-shaped field value gets its `\n` stripped today.

The argument against (and the v1 decision):

1. **Rendering shapes get awkward fast.** Multi-line inside `key=value key=value` logfmt either embeds literal newlines (breaks logfmt parsers) or escapes `\n` to `\\n` (no readability win).
2. **Multi-line cells in CLI's table renderer are broken by design.** A `tabwriter`-aligned table can't span rows for one cell.
3. **Pretty's expanded-YAML mode is the right place** for multi-line per-key rendering, and YAML already has multi-line scalars as a first-class concept. The right v2 work there is "make pretty render embedded `\n` in YAML scalars properly," not "honor the same wrapper."
4. **Stack-trace fields already work in JSON sinks.** The gap is only CLI's `ShowFields` mode and pretty's expanded mode — diagnostic shapes, not the user-facing path most callers care about.
5. **Plugin-author surface stays narrow.** Constraining the wrapper to messages means hooks for `Fields`/`Metadata` don't have to know it exists.

The gap is documented as a `::: warning Messages-only in v1` callout on the doc page, so it isn't a silent omission. If the field-value case becomes a felt need, address it then with a clearer design.

## Testing

**Core (`multiline_test.go`):**
- `Multiline("a","b","c").Lines()` returns the expected slice.
- `String()` joins with `\n`.
- Non-string args (`Multiline(42, true, nil)`) get `%v`-formatted.
- Empty (`Multiline()`) returns empty `Lines()` and `""` from `String()`.
- Nested `Multiline` inside `Multiline`: constructor flattens, so `Multiline(Multiline("a","b"), "c").Lines()` returns `["a","b","c"]` and `String()` returns `"a\nb\nc"`.

**Helper (`transport/helpers_test.go`, extending the existing file):**
- `AssembleMessage` on plain strings matches the existing `JoinMessages` + per-element sanitize behavior.
- `AssembleMessage` on a single `*MultilineMessage` produces per-line-sanitized `\n`-joined output.
- `AssembleMessage` mixed (`"Header:", Multiline("a","b")`) produces `"Header: a\nb"` (sanitized).
- Per-line sanitization within Multiline: `Multiline("clean", "evil\x1b[31mred")` -> `"clean\nevilred"`.
- Bare string with `\n` still gets stripped: `AssembleMessage([]any{"a\nb"}, sanitize.Message)` -> `"ab"`.

**Per terminal transport (`cli`, `pretty`, `console`):**
- One golden-output test each for `Multiline` rendering.
- One regression-guard test confirming a bare `\n`-containing string still gets stripped.
- One mixed-args test (`log.Info("Header:", Multiline("a","b"))`).

**Wrapper transports (zerolog/zap/slog/logrus/charmlog/phuslu):**
- A new `Multiline` scenario added to `transport/transporttest`'s `RunContract`. Every wrapper that already calls `RunContract` picks it up automatically. The scenario asserts the captured output contains `\n` between the authored lines.

**`loghttp` integration:** one test confirming the message-body assembly site honors `Multiline`.

## Documentation

Per `.claude/rules/documentation.md`:

1. **`docs/src/cheatsheet.md`** — add `loglayer.Multiline(lines...)` to the quick reference (alongside `Lazy`).
2. **`docs/src/whats-new.md`** — entry under today's date in the `` `loglayer`: `` paragraph: a paragraph naming the new wrapper, the `Stringer` fallback contract, and the messages-only scope. Linked as `[Multiline](/logging-api/multiline)`.
3. **`docs/src/public/llms.txt`** + **`llms-full.txt`** — add the surface.
4. **New page `docs/src/logging-api/multiline.md`**:
   - Lead-with-the-conclusion intro.
   - Quickstart code block.
   - Threat model: why bare `\n` gets stripped, what the wrapper unlocks, what's still sanitized inside each line.
   - `::: warning Messages-only in v1` callout naming the metadata/fields gap.
   - Per-transport behavior table (mirroring the table above).
5. **Sidebar entry** in `docs/.vitepress/config.ts` under the logging-api section.
6. **GoDoc Examples** per `.claude/rules/godoc-examples.md`:
   - One `ExampleMultiline` in main module's `example_test.go` showing a multi-line headline. Uses `internal/lltest` (or the structured transport with a fixed timestamp hook) for `// Output:` determinism.
   - One Example in `transports/cli`'s `example_test.go` showing the rendered terminal output. cli has TTY-defeated color in tests, so deterministic output is feasible.

## Changeset

Single root-module entry, `:minor` bump:

```
.changeset/multiline-message.md
---
"go.loglayer.dev": minor
---

Add loglayer.Multiline(lines...) for authoring multi-line message
content that survives terminal-renderer sanitization. The wrapper
is messages-only in v1; field/metadata values are still sanitized
to a single line. See https://go.loglayer.dev/logging-api/multiline.
```

No sub-module bumps. The cli/pretty/console/loghttp call-site swaps land in the same PR; the in-tree `replace` directives mean local builds and CI use the new code immediately. Sub-modules ship the call-site swap as part of their own next routine release; until then they keep working because `JoinMessages` and the existing sanitize call still produce correct (if non-multi-line) output.

## Out of scope

- Metadata/fields support for `Multiline` (documented v1 gap).
- A pretty-mode-aware multi-line YAML scalar render (separate redesign).
- Special color treatment of multi-line bodies. Each line tints with the level color via the existing `format` logic; no extra code.
- Auto-detecting `\n` in bare string messages. Rejected: re-introduces the security hole the sanitizer exists to close.

## File-level summary

**New:**
- `multiline.go` (root): the type, constructor, `Lines()`, `String()`.
- `multiline_test.go` (root): unit tests.
- `docs/src/logging-api/multiline.md`: doc page.
- `.changeset/multiline-message.md`: changeset.

**Modified:**
- `transport/helpers.go`: add `AssembleMessage`.
- `transport/helpers_test.go`: extend with `AssembleMessage` cases.
- `transport/transporttest/contract.go`: add Multiline scenario to `RunContract`.
- `transports/cli/cli.go`: swap to `AssembleMessage`; remove `sanitizeMessages`.
- `transports/cli/cli_test.go` (and `example_test.go`): add multi-line + regression-guard tests, plus the cli Example.
- `transports/pretty/pretty.go`: swap to `AssembleMessage`.
- `transports/pretty/pretty_test.go`: add multi-line + regression-guard tests.
- `transports/console/console.go`: swap to `AssembleMessage`.
- `transports/console/console_test.go`: add multi-line + regression-guard tests.
- `integrations/loghttp/`: swap at the message-build site, add a test.
- `example_test.go` (root): add `ExampleMultiline`.
- `docs/src/cheatsheet.md`: add the wrapper.
- `docs/src/whats-new.md`: add today's entry.
- `docs/src/public/llms.txt`, `llms-full.txt`: add the surface.
- `docs/.vitepress/config.ts`: add the new sidebar entry.
