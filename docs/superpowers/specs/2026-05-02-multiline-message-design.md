# `loglayer.Multiline`: developer-authored multi-line messages

**Date:** 2026-05-02
**Status:** Approved (ready for implementation plan)
**Scope:** Single change in the root `go.loglayer.dev` module, plus call-site swaps in the in-tree terminal-shaped transports (`cli`, `pretty`, `console`).

## Motivation

The terminal-shaped transports (`cli`, `pretty`, `console`) run every message string through `utils/sanitize.Message`, which drops control bytes including `\n`. The threat model is two-fold: untrusted log input must not be able to (1) forge follow-up log lines via embedded `\n` or (2) smuggle ANSI / bidi / zero-width terminal escapes. JSON-shaped sinks (`structured` plus every wrapper transport) skip this because `encoding/json` already escapes control bytes.

The result today: any genuine multi-line content authored by the developer is collapsed when it reaches a terminal renderer. Indented configuration dumps, multi-line CLI hints, list-shaped status output, and stack-trace-style content all render as a single line.

The fix is a developer-issued *token of trust*: a typed wrapper the developer constructs explicitly, that signals "I authored these line boundaries." The wrapper rides through dispatch unchanged. Terminal transports detect the type and permit `\n` between authored elements while still sanitizing escapes inside each line. Untrusted input can never become multi-line by accident; it has to be hand-packed into the wrapper's constructor.

## Public surface

A new constructor in the root `loglayer` package. The wrapper is a developer-issued sentinel value, similar in spirit to `*LazyValue`, but applied to message content rather than fields. The two live in disjoint code paths and have different lifetimes; `Lazy` is resolved inside `Fields` during dispatch, `Multiline` is interpreted by transports at the rendering boundary.

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
// rows.
//
// Construction-time normalization, applied uniformly so every transport
// sees the same Lines() shape:
//
//   1. Non-string args convert via fmt.Sprintf("%v", v).
//   2. *MultilineMessage args flatten: their Lines() append into the
//      outer's slice.
//   3. Every resulting string is split on "\n", and each piece becomes
//      one entry of Lines(). After this step, no Lines() entry contains
//      an embedded "\n".
//
// The split rule means Multiline("a\nb") and Multiline("a","b") are
// interchangeable. CRLF input (e.g. "a\r\nb") splits to ["a\r","b"] and
// the trailing "\r" is stripped by per-line sanitization in terminal
// transports, yielding the same rendered output as Multiline("a","b").
func Multiline(lines ...any) *MultilineMessage

// Lines returns the authored line list. Transport authors call this
// when rendering each line independently. No entry contains "\n".
func (m *MultilineMessage) Lines() []string

// String joins the lines with "\n". Used by the fmt.Stringer fallback
// path in transports that don't special-case the type (JSON sinks and
// every wrapper transport).
func (m *MultilineMessage) String() string

// MarshalJSON returns the "\n"-joined string as a JSON string. Provided
// so a wrapper that accidentally lands inside Fields or Metadata serializes
// as a string rather than {} (no exported fields). v1 still sanitizes
// metadata values to a single line in terminal transports; this just
// prevents silent data loss in JSON sinks.
func (m *MultilineMessage) MarshalJSON() ([]byte, error)
```

The wrapper is read-only after construction. No locks. The wrapper does not implement the `error` interface, intentionally: it's a message-content sentinel, not an error value, and conflating the two would force a rendering policy that doesn't fit either role.

## Architecture

The wrapper is **not** flattened in core. It rides through `params.Messages` as one element of `[]any`. Each transport interprets it.

This is the load-bearing decision. If we flattened to a `\n`-joined string in core or in `JoinMessages`, the sanitizer in cli/pretty/console would strip the `\n` back out: the transport wouldn't know it came from authored content. Keeping the typed value intact through dispatch is what makes the trust signal survive to the rendering boundary.

### No dispatch changes

Core's `processLog`, the plugin pipeline, level filtering, group routing, and the `*LogLayer` thread-safety contract are all unchanged. `*MultilineMessage` is just another value type that can appear in `Messages`, alongside `string`, `int`, `*LazyValue`, etc.

Plugin hooks with a Messages signature see the wrapper as-is. Hooks that just want a flat string can call `transport.JoinMessages(params.Messages)` and let `Stringer` flatten transparently. Hooks that want to mutate per-line can type-assert to `*MultilineMessage` and call `Lines()`. Most hooks won't need to care.

### Two paths for transport authors

**JSON sinks and wrapper transports** (`structured`, `zerolog`, `zap`, `slog`, `logrus`, `charmlog`, `phuslu`, `sentry`, `otellog`, `gcplogging`, `http`, `datadog`, `testing`): zero code change. They call `transport.JoinMessages(params.Messages)`, which on a non-string element calls `fmt.Sprintf("%v", v)`. Go's `fmt` package detects `Stringer` (any type with a `String() string` method) and calls it; the `*MultilineMessage.String()` implementation joins with `\n`. The underlying logger writes the joined string. JSON encoders escape `\n` to `\\n`; non-JSON wrappers just hand the string to the writer. No threat from `\n` in these sinks because they don't render to a TTY.

**Sanitizing terminal transports** (`cli`, `pretty`, `console`): a new helper, `transport.AssembleMessage`, replaces the existing per-transport sanitize-then-join (or join-then-sanitize) pattern.

**`integrations/loghttp`** does not need a swap. Its only `sanitize.Message` calls (`sanitizeForLog`) apply to *fields* (`RequestID`, `Method`, `Path`, the recovered panic value), not to message content. Its actual log calls use hardcoded literal messages (`"request started"`, `"request completed"`, `"request panicked"`). The integration is downstream of the cli/pretty/console transports; users get the new behavior from those.

### `JoinPrefixAndMessages` Multiline-handling

The existing `transport.JoinPrefixAndMessages` helper in `transport/helpers.go` folds a `WithPrefix` value into `Messages[0]`. Its current shape returns Messages unchanged when `Messages[0]` is not a string, which would silently drop the prefix on `log.WithPrefix("X").Info(loglayer.Multiline(...))`. ~14 transports call this helper.

The fix: extend the helper to handle `*MultilineMessage` explicitly. When `Messages[0]` is a `*MultilineMessage`, return a fresh `[]any` whose first element is a new `*MultilineMessage` with the prefix prepended to the first authored line:

```go
// Pseudocode for the added branch:
if m, ok := messages[0].(*MultilineMessage); ok {
    head := prefix + " " + m.Lines()[0]
    rebuilt := &MultilineMessage{lines: append([]string{head}, m.Lines()[1:]...)}
    out := make([]any, len(messages))
    copy(out, messages)
    out[0] = rebuilt
    return out
}
```

The non-string-non-Multiline fallback stays as-is for now: that's a pre-existing limitation independent of this work. A follow-up could generalize via `fmt.Sprint`, but it's out of scope here.

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
// Used by terminal-style transports (cli, pretty, console). Wrapper
// transports and JSON sinks call JoinMessages instead;
// *MultilineMessage's String method handles flattening transparently
// for them.
func AssembleMessage(messages []any, sanitize func(string) string) string
```

The CLI transport's existing `sanitizeMessages` private helper is removed; `AssembleMessage` covers the same job in fewer lines.

## Per-transport call-site swaps

| Caller | Today | After |
|---|---|---|
| `transports/cli/cli.go:256` | `transport.JoinMessages(sanitizeMessages(params.Messages))` | `transport.AssembleMessage(params.Messages, sanitize.Message)` |
| `transports/pretty/pretty.go:127` | `sanitize.Message(transport.JoinMessages(params.Messages))` | `transport.AssembleMessage(params.Messages, sanitize.Message)` |
| `transports/console/console.go:105-163` | per-element sanitize loop in `buildMessages`; two downstream rendering modes | see "Console swap" below |

No other behavior changes in the call sites: color, level prefixes, table rendering, expanded YAML, etc. all keep working. Each rendered line tints with the level color in cli (same as today's single-line body); pretty and console render the multi-line block as one body where their existing layout already works.

### Console swap

`transports/console/console.go`'s `buildMessages` has two output shapes that need to be reconciled with the new helper:

- **MessageField mode** (line ~130): assembles via `JoinMessages(messages)` into `obj[cfg.MessageField]`. Direct swap to `AssembleMessage(params.Messages, sanitize.Message)`.
- **Default / logfmt mode** (lines ~158-160): returns `messages` (the per-element-sanitized `[]any` slice) plus an optional appended logfmt string, passed to `Fprintln(c.writer(level), messages...)` at the call site (line 85). `Fprintln` space-separates the args.

Decision for the default mode: assemble the message portion to a single string via `AssembleMessage`, then if there's a logfmt suffix, concatenate (`assembled + " " + logfmt`), and pass the single string to `Fprintln`. Output is byte-equivalent today (Fprintln space-joins variadic args, just like the explicit `" "` concatenation). The Multiline lines render with literal `\n` between them in the resulting JSON message field or in the logfmt-mode raw line.

The per-element sanitize loop at lines 116-120 goes away; `AssembleMessage` covers it.

## Rendering by transport class

| Transport | `Multiline("a","b")` alone | Mixed: `"Header:", Multiline("a","b")` |
|---|---|---|
| **cli** | `a\nb` (level-colored, on the level's writer) | `Header: a\nb` |
| **pretty** | `[ts] [INFO] a\nb` | `[ts] [INFO] Header: a\nb` |
| **console** (JSON) | `{"msg":"a\nb",...}` | `{"msg":"Header: a\nb",...}` |
| **structured** | `{"msg":"a\nb",...}` | `{"msg":"Header: a\nb",...}` |
| **zerolog/zap/slog/logrus/charmlog/phuslu** | underlying logger receives `"a\nb"` | underlying logger receives `"Header: a\nb"` |

## Edge cases (explicit decisions)

| Case | Behavior | Rationale |
|---|---|---|
| `Multiline()` (zero args) | Empty string. Transport's existing "empty body skip" logic handles whether to print at all. | Symmetric with `log.Info("")`. |
| `Multiline("")` (one empty arg) | Empty string (one empty line, no `\n`). | A single line, even an empty one, isn't multi-line. |
| `Multiline("a", "")` (trailing empty) | `"a\n"`. Preserves the authored boundary. | The developer asked for it. |
| `Multiline("a\nb")` (string with embedded `\n`) | `Lines()` == `["a","b"]`, rendered as `"a\nb"`. | Constructor splits on `\n`. Symmetric with `Multiline("a","b")`; avoids silent mangling when terminal-side sanitize would otherwise strip the inner `\n`. |
| `Multiline("a\r\nb")` (CRLF) | `Lines()` == `["a\r","b"]` after split; per-line sanitize strips the trailing `\r` in terminal transports; rendered as `"a\nb"`. | Falls out of "split on `\n`, sanitize per line." |
| `Multiline(Multiline("a","b"), "c")` (nested) | `"a\nb\nc"`. The constructor flattens nested wrappers so `Lines()` == `["a","b","c"]`. | Without flattening, JSON sinks would render `"a\nb\nc"` (via `Stringer`) but terminal sinks would render `"ab\nc"` (per-line sanitize would strip the inner `\n`). Flattening at construction makes both paths agree. |
| `nil` element (`Multiline("a", nil, "b")`) | `"a\n<nil>\nb"` via `fmt.Sprintf("%v", nil)`. | Matches today's `JoinMessages` behavior. |
| `*MultilineMessage` inside `Fields` or `Metadata` | Untouched by `AssembleMessage`. CLI/pretty/console still sanitize via `writeValue` / cell-render and strip `\n`. JSON sinks (structured + every wrapper) serialize via `MarshalJSON` to the `\n`-joined string, so no silent data loss. | Documented v1 gap for terminal sinks. |

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
4. **Stack-trace fields already work in JSON sinks.** The gap is only CLI's `ShowFields` mode and pretty's expanded mode: diagnostic shapes, not the user-facing path most callers care about.
5. **Plugin-author surface stays narrow.** Constraining the wrapper to messages means hooks for `Fields`/`Metadata` don't have to know it exists.

The gap is documented as a `::: warning Messages-only in v1` callout on the doc page, so it isn't a silent omission. If the field-value case becomes a felt need, address it then with a clearer design.

## Interaction with plugins

`*MultilineMessage` survives the dispatch path unchanged, so it's visible to plugin hooks that walk `params.Messages`. Two interactions worth flagging:

- **`fmtlog`** (the `fmt.Sprintf`-style format-string plugin) replaces `params.Messages` with `[]any{fmt.Sprintf(format, p.Messages[1:]...)}` when called as `log.Info("request: %v", multilineValue)`. The Sprintf path resolves Stringer, producing `"request: a\nb"` as a flat `string`. Downstream sanitize then strips the inner `\n` and the user gets `"request: ab"`. **The trust signal is silently lost** in this combination, by design: the user opted into a plugin whose contract is "flatten args into a format string." A future fmtlog enhancement could grow a Multiline-aware path (e.g., produce a `*MultilineMessage` when any arg is one); out of scope here.
  - Workaround for callers today: don't combine `fmtlog`'s format-string mode with `Multiline`. Construct the wrapper directly with the formatted lines instead.
- **Generic message-mutating hooks**: a plugin that converts `params.Messages` into something else (a `string`, a different slice shape) loses the wrapper. Plugins that want to preserve it should pass `*MultilineMessage` values through unchanged, or rebuild a new wrapper at the end of their transformation. Document this in `docs/src/plugins/creating-plugins.md` as part of the same change.

## Testing

**Core (`multiline_test.go`):**
- `Multiline("a","b","c").Lines()` returns `["a","b","c"]`.
- `String()` joins with `\n`.
- `MarshalJSON` returns `"a\nb\nc"` JSON-encoded (a JSON string with `\\n` escapes).
- Non-string args (`Multiline(42, true, nil)`) get `%v`-formatted.
- Empty (`Multiline()`) returns empty `Lines()` and `""` from `String()`.
- `Multiline("a\nb")` splits at construction: `Lines() == ["a","b"]`.
- `Multiline("a\r\nb")` splits at construction: `Lines() == ["a\r","b"]`.
- Nested flattening: `Multiline(Multiline("a","b"), "c").Lines()` returns `["a","b","c"]` and `String()` returns `"a\nb\nc"`.
- `*MultilineMessage` does not implement `error`.

**Helper (`transport/helpers_test.go`, extending the existing file):**
- `AssembleMessage` on plain strings matches the existing `JoinMessages` + per-element sanitize behavior.
- `AssembleMessage` on a single `*MultilineMessage` produces per-line-sanitized `\n`-joined output.
- `AssembleMessage` mixed (`"Header:", Multiline("a","b")`) produces `"Header: a\nb"` (sanitized).
- Per-line ANSI sanitization within Multiline: `Multiline("clean", "evil\x1b[31mred")` -> `"clean\nevilred"`.
- Per-line CR sanitization: `Multiline("a\r", "b")` -> `"a\nb"` (the `\r` strips, the `\n` boundary survives).
- Cross-line ANSI smuggling defeated: `Multiline("\x1b", "[31mred")` -> `"\n[31mred"` (the bare `\x1b` ESC strips inside line 0; the bracket sequence in line 1 is not preceded by an ESC byte).
- Bidi / ZWJ stripping: `Multiline("‮", "evil")` -> `"\nevil"`; `Multiline("zero​width", "y")` -> `"zerowidth\ny"`.
- Bare string with `\n` still gets stripped (no wrapper, no trust): `AssembleMessage([]any{"a\nb"}, sanitize.Message)` -> `"ab"`.
- `JoinPrefixAndMessages` with `prefix="X"` and `messages=[Multiline("a","b")]` returns `[Multiline("X a","b")]` (prefix folded into first authored line; subsequent lines unchanged).
- `JoinPrefixAndMessages` with `prefix=""` returns `messages` unchanged (existing fast path).

**Per terminal transport (`cli`, `pretty`, `console`):**
- One golden-output test each for `Multiline` rendering.
- One regression-guard test confirming a bare `\n`-containing string still gets stripped.
- One mixed-args test (`log.Info("Header:", Multiline("a","b"))`).
- One empty-Multiline test (`log.Info(Multiline())` should produce the same output as `log.Info("")` for that transport: no extra blank line, no panic).
- One `WithPrefix + Multiline` test confirming the prefix lands on the first line and not the rest.

**Wrapper transports** (zerolog/zap/slog/logrus/charmlog/phuslu/sentry/otellog/gcplogging/http/datadog/testing):
- A new `Multiline` scenario added to `transport/transporttest/contract.go`'s `RunContract`. Every wrapper that already calls `RunContract` picks it up automatically. The scenario asserts the captured output contains a literal `\n` between the authored lines.
- A `WithPrefix + Multiline` contract case verifying the prefix folds into the first authored line.

## Documentation

Per `.claude/rules/documentation.md`:

1. **`docs/src/cheatsheet.md`**: add `loglayer.Multiline(lines...)` to the quick reference (alongside `Lazy`).
2. **`docs/src/whats-new.md`**: entry under today's date in the `` `loglayer`: `` paragraph: a paragraph naming the new wrapper, the `Stringer` fallback contract, and the messages-only scope. Linked as `[Multiline](/logging-api/multiline)`.
3. **`docs/src/public/llms.txt`** + **`llms-full.txt`**: add the surface.
4. **New page `docs/src/logging-api/multiline.md`**:
   - Lead-with-the-conclusion intro.
   - Quickstart code block.
   - Threat model: why bare `\n` gets stripped, what the wrapper unlocks, what's still sanitized inside each line.
   - `::: warning Messages-only in v1` callout naming the metadata/fields gap.
   - Per-transport behavior table (mirroring the table above).
5. **Sidebar entry** in `docs/.vitepress/config.ts` under the logging-api section.
6. **GoDoc Examples** per `.claude/rules/godoc-examples.md`:
   - One `ExampleMultiline` in main module's `example_test.go` showing a multi-line headline. Uses the existing in-file `exampleTransport` pattern (JSON-shaped, fixed `time` field) so `// Output:` is deterministic. Mirrors the existing examples in that file.
   - One Example in `transports/cli`'s `example_test.go` showing the rendered terminal output. cli's color resolution defaults to off when `Stdout` is not a TTY (which it isn't under `go test`), so deterministic output is feasible.

## Changeset

The PR ships one changeset that names the root and every sub-module whose code changes. Atomic shipping avoids the "feature is real but my pinned `transports/cli@v2.0.x` doesn't have it yet" trap: users see a coherent slice once the release PR merges.

```
.changeset/multiline-message.md
---
"go.loglayer.dev": minor
"transports/cli": minor
"transports/pretty": minor
"transports/console": minor
---

Add loglayer.Multiline(lines...) for authoring multi-line message
content that survives terminal-renderer sanitization. The wrapper
is messages-only in v1; field/metadata values are still sanitized
to a single line in terminal transports (JSON sinks serialize via
MarshalJSON to the joined string). See
https://go.loglayer.dev/logging-api/multiline.
```

The wrapper transports (`zerolog`, `zap`, `slog`, `logrus`, `charmlog`, `phuslu`, `sentry`, `otellog`, `gcplogging`, `http`, `datadog`, `testing`) and `structured` need no code change. `Multiline` flows through their existing `JoinMessages` path via Stringer. They pick up the new behavior automatically when their next routine release happens to bump their `go.loglayer.dev` requirement past v2.1.0. No same-PR changeset entries needed for them.

The `integrations/loghttp` integration also needs no code change (its message strings are hardcoded literals; Multiline is irrelevant there).

## Out of scope

- Metadata/fields support for `Multiline` (documented v1 gap).
- A pretty-mode-aware multi-line YAML scalar render (separate redesign).
- Special color treatment of multi-line bodies. Each line tints with the level color via the existing `format` logic; no extra code.
- Auto-detecting `\n` in bare string messages. Rejected: re-introduces the security hole the sanitizer exists to close.

## File-level summary

**New:**
- `multiline.go` (root): the type, constructor, `Lines()`, `String()`, `MarshalJSON()`.
- `multiline_test.go` (root): unit tests.
- `docs/src/logging-api/multiline.md`: doc page.
- `.changeset/multiline-message.md`: changeset.

**Modified:**
- `transport/helpers.go`: add `AssembleMessage`; extend `JoinPrefixAndMessages` to handle `*MultilineMessage`.
- `transport/helpers_test.go`: extend with `AssembleMessage` cases and the `JoinPrefixAndMessages` Multiline case.
- `transport/transporttest/contract.go`: add Multiline scenario (and `WithPrefix + Multiline` scenario) to `RunContract`.
- `transports/cli/cli.go`: swap to `AssembleMessage`; remove `sanitizeMessages`.
- `transports/cli/cli_test.go` (and `example_test.go`): add multi-line + regression-guard tests, plus the cli Example.
- `transports/pretty/pretty.go`: swap to `AssembleMessage`.
- `transports/pretty/pretty_test.go`: add multi-line + regression-guard tests.
- `transports/console/console.go`: swap to `AssembleMessage` per the "Console swap" decision; remove the per-element sanitize loop.
- `transports/console/console_test.go`: add multi-line + regression-guard tests covering both `MessageField` and default modes.
- `example_test.go` (root): add `ExampleMultiline`.
- `docs/src/cheatsheet.md`: add the wrapper.
- `docs/src/whats-new.md`: add today's entry.
- `docs/src/public/llms.txt`, `llms-full.txt`: add the surface.
- `docs/.vitepress/config.ts`: add the new sidebar entry.
- `docs/src/plugins/creating-plugins.md`: add a one-paragraph note about preserving `*MultilineMessage` values through message-mutating hooks.
