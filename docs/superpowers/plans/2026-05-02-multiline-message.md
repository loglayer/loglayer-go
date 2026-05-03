# `loglayer.Multiline` Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `loglayer.Multiline(lines ...any)` constructor that lets developers author multi-line message content that survives the terminal-renderer sanitization in `cli`, `pretty`, and `console` while preserving the per-line ANSI/CR/bidi/ZWJ stripping that defeats log-forging and escape-smuggling.

**Architecture:** A new `*MultilineMessage` value type rides through `params.Messages` unchanged. JSON sinks and wrapper transports flatten via Stringer (`String()` joins with `\n`); zero code change. Sanitizing terminal transports adopt a new `transport.AssembleMessage(messages, sanitize)` helper that sanitizes per-line and joins with `\n`. The `transport.JoinPrefixAndMessages` helper is extended to handle `*MultilineMessage` (prefix lands on the first line) and to fix a pre-existing bug where the prefix was silently dropped for any non-string first message.

**Tech Stack:** Go 1.25+ (main module's existing floor). Standard library only for the type itself; existing `utils/sanitize` for per-line cleaning. Tests use `go test`. Per-module tests run via `bash scripts/foreach-module.sh test`. Spec at `docs/superpowers/specs/2026-05-02-multiline-message-design.md`.

## File structure

**New files:**
- `multiline.go` (root): the `*MultilineMessage` type, the `Multiline` constructor, `Lines()`, `String()`, `MarshalJSON()`.
- `multiline_test.go` (root): unit tests for the type.
- `docs/src/logging-api/multiline.md`: doc page.
- `.changeset/multiline-message.md`: changeset file.

**Modified files:**
- `transport/helpers.go`: add `AssembleMessage`; extend `JoinPrefixAndMessages` for `*MultilineMessage` and the general non-string case.
- `transport/helpers_test.go`: extend with `AssembleMessage` cases and updated `JoinPrefixAndMessages` cases.
- `transport/transporttest/contract.go`: add `Multiline` and `WithPrefix + Multiline` scenarios to `RunContract`.
- `transports/cli/cli.go`: swap to `AssembleMessage`; remove `sanitizeMessages` helper.
- `transports/cli/cli_test.go`: add multi-line, regression-guard, mixed-args, prefix, and empty tests.
- `transports/cli/example_test.go`: add a multi-line Example.
- `transports/pretty/pretty.go`: swap to `AssembleMessage`.
- `transports/pretty/pretty_test.go`: add the same suite of tests.
- `transports/console/console.go`: swap to `AssembleMessage` per the "Console swap" decision.
- `transports/console/console_test.go`: add tests covering both `MessageField` and default modes.
- `example_test.go` (root): add `ExampleMultiline`.
- `docs/src/cheatsheet.md`: add the wrapper.
- `docs/src/whats-new.md`: add today's entry.
- `docs/src/public/llms.txt` and `llms-full.txt`: add the surface.
- `docs/.vitepress/config.ts`: add the new sidebar entry.
- `docs/src/plugins/creating-plugins.md`: add the Multiline preservation note.

---

## Working agreements

- **Per-task verify command:** `go build ./... && go test ./...` from the repo root, unless the task explicitly says otherwise (sub-modules: `bash scripts/foreach-module.sh test`).
- **Commit format:** Conventional Commits, scoped where relevant (`feat:`, `feat(transports/cli):`, `docs:`, `test:`). Body wraps at 72 cols. Trailers include `Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>` per the project's repo convention.
- **No `--no-verify`:** lefthook hooks must pass. If staticcheck flags something, fix the underlying issue.
- **One task = one PR-worthy unit:** each task ends in a green commit. Don't roll multiple unrelated changes into one task.

---

## Task 1: `MultilineMessage` skeleton: type, constructor, `Lines()`

**Files:**
- Create: `multiline.go`
- Create: `multiline_test.go`

- [ ] **Step 1: Write the failing test for the simplest constructor + Lines case**

Create `multiline_test.go`:

```go
package loglayer_test

import (
	"reflect"
	"testing"

	"go.loglayer.dev/v2"
)

func TestMultiline_LinesReturnsAuthoredSlice(t *testing.T) {
	m := loglayer.Multiline("a", "b", "c")
	got := m.Lines()
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Lines() = %#v, want %#v", got, want)
	}
}
```

- [ ] **Step 2: Run the test to confirm it fails**

Run: `go test ./... -run TestMultiline_LinesReturnsAuthoredSlice`
Expected: build failure (`undefined: loglayer.Multiline`).

- [ ] **Step 3: Implement the minimal constructor + Lines accessor**

Create `multiline.go`:

```go
package loglayer

// MultilineMessage wraps a sequence of authored lines so terminal
// transports render them on separate rows. Construct with [Multiline].
//
// Token of trust: the wrapper signals that the developer authored the
// line boundaries, so terminal renderers permit \n between elements
// while still sanitizing ANSI / control bytes within each line.
type MultilineMessage struct {
	lines []string
}

// Multiline wraps the supplied arguments as separate authored lines.
//
// This minimal form treats every argument as already-string-shaped.
// Later steps in this plan extend the constructor with non-string %v
// formatting, nested-wrapper flattening, and per-arg "\n" splitting.
func Multiline(lines ...any) *MultilineMessage {
	out := make([]string, len(lines))
	for i, l := range lines {
		s, _ := l.(string)
		out[i] = s
	}
	return &MultilineMessage{lines: out}
}

// Lines returns the authored line list. Transport authors call this
// when rendering each line independently.
func (m *MultilineMessage) Lines() []string {
	return m.lines
}
```

- [ ] **Step 4: Run the test to confirm it passes**

Run: `go test ./... -run TestMultiline_LinesReturnsAuthoredSlice`
Expected: PASS.

- [ ] **Step 5: Run the full test suite to confirm no regressions**

Run: `go build ./... && go test ./...`
Expected: PASS (no new failures).

- [ ] **Step 6: Commit**

```bash
git add multiline.go multiline_test.go
git commit -m "$(cat <<'EOF'
feat: add MultilineMessage skeleton (constructor + Lines)

Initial shape of the *MultilineMessage type with a string-only
constructor. Subsequent commits extend the constructor with non-string
%v formatting, nested-wrapper flattening, "\n" splitting, String(),
and MarshalJSON.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: `String()` Stringer implementation

**Files:**
- Modify: `multiline.go`
- Modify: `multiline_test.go`

- [ ] **Step 1: Write the failing test**

Append to `multiline_test.go`:

```go
func TestMultiline_StringJoinsWithNewline(t *testing.T) {
	m := loglayer.Multiline("a", "b", "c")
	if got := m.String(); got != "a\nb\nc" {
		t.Errorf("String() = %q, want %q", got, "a\nb\nc")
	}
}

func TestMultiline_StringEmpty(t *testing.T) {
	m := loglayer.Multiline()
	if got := m.String(); got != "" {
		t.Errorf("String() on zero-arg = %q, want empty", got)
	}
}

func TestMultiline_StringSingle(t *testing.T) {
	m := loglayer.Multiline("only")
	if got := m.String(); got != "only" {
		t.Errorf("String() on single-arg = %q, want %q", got, "only")
	}
}
```

- [ ] **Step 2: Run to confirm it fails**

Run: `go test ./... -run "TestMultiline_String"`
Expected: build failure (`*MultilineMessage has no field or method String`).

- [ ] **Step 3: Implement `String()`**

Append to `multiline.go`:

```go
import "strings"

// String joins the lines with "\n". Used by the fmt.Stringer fallback
// path in transports that don't special-case the type (JSON sinks and
// every wrapper transport).
func (m *MultilineMessage) String() string {
	return strings.Join(m.lines, "\n")
}
```

- [ ] **Step 4: Run to confirm tests pass**

Run: `go test ./... -run "TestMultiline_String"`
Expected: PASS.

- [ ] **Step 5: Run the full test suite**

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add multiline.go multiline_test.go
git commit -m "$(cat <<'EOF'
feat: implement MultilineMessage.String() (Stringer)

JSON sinks and wrapper transports rely on the Stringer fallback to
flatten *MultilineMessage to "\n"-joined text via fmt.Sprintf("%v", v).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Constructor: non-string args via `fmt.Sprintf("%v", v)`

**Files:**
- Modify: `multiline.go`
- Modify: `multiline_test.go`

- [ ] **Step 1: Write the failing test**

Append to `multiline_test.go`:

```go
func TestMultiline_NonStringArgsFormatViaPercentV(t *testing.T) {
	m := loglayer.Multiline(42, true, nil)
	got := m.Lines()
	want := []string{"42", "true", "<nil>"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Lines() = %#v, want %#v", got, want)
	}
}

type stringerOnly struct{ v string }

func (s stringerOnly) String() string { return "stringer:" + s.v }

func TestMultiline_StringerArgsCallStringMethod(t *testing.T) {
	m := loglayer.Multiline(stringerOnly{v: "x"})
	if got := m.Lines(); !reflect.DeepEqual(got, []string{"stringer:x"}) {
		t.Errorf("Lines() = %#v, want [stringer:x]", got)
	}
}
```

- [ ] **Step 2: Run to confirm it fails**

Run: `go test ./... -run "TestMultiline_NonStringArgs|TestMultiline_StringerArgs"`
Expected: FAIL: current constructor stores `""` for non-string args.

- [ ] **Step 3: Extend the constructor**

Replace the constructor body in `multiline.go`:

```go
import "fmt"  // add to existing imports

func Multiline(lines ...any) *MultilineMessage {
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		if s, ok := l.(string); ok {
			out = append(out, s)
			continue
		}
		out = append(out, fmt.Sprintf("%v", l))
	}
	return &MultilineMessage{lines: out}
}
```

- [ ] **Step 4: Run to confirm tests pass**

Run: `go test ./... -run "TestMultiline_"`
Expected: PASS.

- [ ] **Step 5: Run the full suite**

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add multiline.go multiline_test.go
git commit -m "$(cat <<'EOF'
feat: format non-string Multiline args via fmt.Sprintf("%v", v)

Resolves the Stringer interface and matches existing JoinMessages
semantics for non-string elements.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Constructor: flatten nested `*MultilineMessage`

**Files:**
- Modify: `multiline.go`
- Modify: `multiline_test.go`

- [ ] **Step 1: Write the failing test**

Append to `multiline_test.go`:

```go
func TestMultiline_NestedFlattens(t *testing.T) {
	inner := loglayer.Multiline("a", "b")
	outer := loglayer.Multiline(inner, "c")
	got := outer.Lines()
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("nested Lines() = %#v, want %#v", got, want)
	}
	if s := outer.String(); s != "a\nb\nc" {
		t.Errorf("nested String() = %q, want %q", s, "a\nb\nc")
	}
}

func TestMultiline_NestedDeep(t *testing.T) {
	inner := loglayer.Multiline("x")
	mid := loglayer.Multiline(inner, "y")
	outer := loglayer.Multiline(mid, "z")
	got := outer.Lines()
	want := []string{"x", "y", "z"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("deep-nested Lines() = %#v, want %#v", got, want)
	}
}
```

- [ ] **Step 2: Run to confirm it fails**

Run: `go test ./... -run "TestMultiline_Nested"`
Expected: FAIL: current constructor calls `%v` on `*MultilineMessage`, producing `Lines() == ["a\nb", "c"]` (one line with embedded `\n`).

- [ ] **Step 3: Extend the constructor with the flattening branch**

Replace the constructor body in `multiline.go`:

```go
func Multiline(lines ...any) *MultilineMessage {
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		switch v := l.(type) {
		case *MultilineMessage:
			out = append(out, v.lines...)
		case string:
			out = append(out, v)
		default:
			out = append(out, fmt.Sprintf("%v", v))
		}
	}
	return &MultilineMessage{lines: out}
}
```

- [ ] **Step 4: Run to confirm tests pass**

Run: `go test ./... -run "TestMultiline_"`
Expected: PASS.

- [ ] **Step 5: Run the full suite**

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add multiline.go multiline_test.go
git commit -m "$(cat <<'EOF'
feat: flatten nested *MultilineMessage at construction

Multiline(Multiline("a","b"), "c").Lines() now returns ["a","b","c"]
rather than ["a\nb","c"]. Without flattening, terminal sinks (per-line
sanitize) and JSON sinks (Stringer) would render the same value
differently. The inner "\n" would survive the JSON path but vanish
from the terminal path.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Constructor: split each string on `\n`

**Files:**
- Modify: `multiline.go`
- Modify: `multiline_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `multiline_test.go`:

```go
func TestMultiline_SplitsEmbeddedNewline(t *testing.T) {
	m := loglayer.Multiline("a\nb")
	got := m.Lines()
	want := []string{"a", "b"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Lines() = %#v, want %#v", got, want)
	}
	if s := m.String(); s != "a\nb" {
		t.Errorf("String() = %q, want %q", s, "a\nb")
	}
}

func TestMultiline_SplitMixedWithLiteralArgs(t *testing.T) {
	m := loglayer.Multiline("a\nb", "c")
	got := m.Lines()
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Lines() = %#v, want %#v", got, want)
	}
}

func TestMultiline_SplitCRLFKeepsCRForSanitizerToStrip(t *testing.T) {
	// CRLF input: split on "\n" produces ["a\r", "b"]. Per-line
	// sanitize at the terminal-transport boundary will strip the "\r";
	// JSON sinks see "a\r\nb" via Stringer (encodings escape both bytes).
	m := loglayer.Multiline("a\r\nb")
	got := m.Lines()
	want := []string{"a\r", "b"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Lines() = %#v, want %#v", got, want)
	}
}

func TestMultiline_SplitAppliesAfterStringerFormatting(t *testing.T) {
	// A %v-formatted argument that yields a string with "\n" should
	// split too, so pre- and post-stringification arguments behave
	// the same way.
	type multilineStringer struct{}
	// We can't define this helper inside the test in a way that
	// reflect.DeepEqual works against; the test relies on the value
	// produced by fmt.Sprintf("%v", multilineStringer{}) being literal
	// "{multi\nline}" via the default Go format for a struct. Use a
	// dedicated stringerOnly to control output.
	m := loglayer.Multiline(stringerOnly{v: "x\ny"})
	got := m.Lines()
	want := []string{"stringer:x", "y"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Lines() = %#v, want %#v", got, want)
	}
}
```

- [ ] **Step 2: Run to confirm they fail**

Run: `go test ./... -run "TestMultiline_Split"`
Expected: FAIL: constructor doesn't split.

- [ ] **Step 3: Extend the constructor**

Replace the constructor body in `multiline.go`:

```go
func Multiline(lines ...any) *MultilineMessage {
	out := make([]string, 0, len(lines))
	appendSplit := func(s string) {
		if s == "" {
			out = append(out, "")
			return
		}
		out = append(out, strings.Split(s, "\n")...)
	}
	for _, l := range lines {
		switch v := l.(type) {
		case *MultilineMessage:
			out = append(out, v.lines...)
		case string:
			appendSplit(v)
		default:
			appendSplit(fmt.Sprintf("%v", v))
		}
	}
	return &MultilineMessage{lines: out}
}
```

- [ ] **Step 4: Run to confirm tests pass**

Run: `go test ./... -run "TestMultiline_"`
Expected: PASS.

- [ ] **Step 5: Run the full suite**

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add multiline.go multiline_test.go
git commit -m "$(cat <<'EOF'
feat: split each Multiline arg on "\n" at construction

Multiline("a\nb") and Multiline("a","b") are now interchangeable; their
Lines() shape and rendered output match across every transport. CRLF
input splits to [..."\r", ...] so the per-line sanitize at terminal
transports strips the "\r" and yields the same display as plain LF.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: `MarshalJSON`

**Files:**
- Modify: `multiline.go`
- Modify: `multiline_test.go`

- [ ] **Step 1: Write the failing test**

Append to `multiline_test.go`:

```go
import (
	"encoding/json"  // add to existing imports
)

func TestMultiline_MarshalJSON(t *testing.T) {
	m := loglayer.Multiline("a", "b", "c")
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal err: %v", err)
	}
	want := `"a\nb\nc"`
	if string(b) != want {
		t.Errorf("Marshal = %s, want %s", b, want)
	}
}

func TestMultiline_MarshalJSON_EmptyLines(t *testing.T) {
	m := loglayer.Multiline()
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal err: %v", err)
	}
	if string(b) != `""` {
		t.Errorf("Marshal = %s, want \"\"", b)
	}
}

func TestMultiline_MarshalJSON_RoundtripFromMetadata(t *testing.T) {
	// A user accidentally putting Multiline inside Metadata should
	// serialize as a JSON string, not the empty {} that an unexported-
	// field struct would produce.
	wrapped := map[string]any{"detail": loglayer.Multiline("x", "y")}
	b, err := json.Marshal(wrapped)
	if err != nil {
		t.Fatalf("Marshal err: %v", err)
	}
	want := `{"detail":"x\ny"}`
	if string(b) != want {
		t.Errorf("Marshal = %s, want %s", b, want)
	}
}
```

- [ ] **Step 2: Run to confirm they fail**

Run: `go test ./... -run "TestMultiline_MarshalJSON"`
Expected: FAIL: default Go marshaling produces `{}` for a struct with no exported fields.

- [ ] **Step 3: Implement `MarshalJSON`**

Append to `multiline.go`:

```go
import (
	"encoding/json"  // add to existing imports
)

// MarshalJSON returns the "\n"-joined string as a JSON string. Provided
// so a wrapper that accidentally lands inside Fields or Metadata
// serializes as a string rather than {} (no exported fields). Terminal
// transports still sanitize metadata values to a single line in v1;
// this just prevents silent data loss in JSON sinks.
func (m *MultilineMessage) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.String())
}
```

- [ ] **Step 4: Run to confirm tests pass**

Run: `go test ./... -run "TestMultiline_MarshalJSON"`
Expected: PASS.

- [ ] **Step 5: Run the full suite**

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add multiline.go multiline_test.go
git commit -m "$(cat <<'EOF'
feat: MultilineMessage.MarshalJSON serializes joined string

Prevents silent data loss when a *MultilineMessage value is placed in
Fields or Metadata and reaches a JSON sink. With no exported fields,
default Go marshaling would produce {}. The wrapper now produces the
"\n"-joined string instead, consistent with String().

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: Negative-assertion test: wrapper does not implement `error`

**Files:**
- Modify: `multiline_test.go`

- [ ] **Step 1: Write the test**

Append to `multiline_test.go`:

```go
func TestMultiline_DoesNotImplementError(t *testing.T) {
	// The wrapper is a message-content sentinel. Implementing error
	// would force a rendering policy that fits neither role. This
	// test pins that decision so a future "convenience" PR doesn't
	// accidentally add it.
	var v any = loglayer.Multiline("a", "b")
	if _, ok := v.(error); ok {
		t.Fatal("*MultilineMessage must not implement error")
	}
}
```

- [ ] **Step 2: Run to confirm it passes (current state already meets the requirement)**

Run: `go test ./... -run "TestMultiline_DoesNotImplementError"`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add multiline_test.go
git commit -m "$(cat <<'EOF'
test: pin that *MultilineMessage does not implement error

Negative assertion guarding against a future "convenience" addition
that would conflate message-content sentinels with error values.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 8: `transport.AssembleMessage` helper

**Files:**
- Modify: `transport/helpers.go`
- Modify: `transport/helpers_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `transport/helpers_test.go`:

```go
import (
	"go.loglayer.dev/v2/utils/sanitize"  // add to existing imports if absent
)

func TestAssembleMessage_PlainStrings(t *testing.T) {
	got := transport.AssembleMessage([]any{"hello", "world"}, sanitize.Message)
	if got != "hello world" {
		t.Errorf("got %q, want %q", got, "hello world")
	}
}

func TestAssembleMessage_NonStringElement(t *testing.T) {
	got := transport.AssembleMessage([]any{"value:", 42}, sanitize.Message)
	if got != "value: 42" {
		t.Errorf("got %q, want %q", got, "value: 42")
	}
}

func TestAssembleMessage_MultilineAlone(t *testing.T) {
	got := transport.AssembleMessage([]any{loglayer.Multiline("a", "b")}, sanitize.Message)
	if got != "a\nb" {
		t.Errorf("got %q, want %q", got, "a\nb")
	}
}

func TestAssembleMessage_MultilineMixedWithString(t *testing.T) {
	got := transport.AssembleMessage([]any{"Header:", loglayer.Multiline("a", "b")}, sanitize.Message)
	if got != "Header: a\nb" {
		t.Errorf("got %q, want %q", got, "Header: a\nb")
	}
}

func TestAssembleMessage_BareNewlineGetsStripped(t *testing.T) {
	// No wrapper, no trust: a string with embedded "\n" still has it
	// stripped by per-element sanitize.
	got := transport.AssembleMessage([]any{"a\nb"}, sanitize.Message)
	if got != "ab" {
		t.Errorf("got %q, want %q (newline must be stripped from a bare string)", got, "ab")
	}
}

func TestAssembleMessage_PerLineANSIStrippedInsideMultiline(t *testing.T) {
	// ANSI inside a single line still strips. Only the boundaries
	// between authored elements are preserved.
	got := transport.AssembleMessage(
		[]any{loglayer.Multiline("clean", "evil\x1b[31mred")},
		sanitize.Message,
	)
	if got != "clean\nevilred" {
		t.Errorf("got %q, want %q", got, "clean\nevilred")
	}
}

func TestAssembleMessage_EmptyInputProducesEmptyString(t *testing.T) {
	if got := transport.AssembleMessage(nil, sanitize.Message); got != "" {
		t.Errorf("got %q, want empty", got)
	}
	if got := transport.AssembleMessage([]any{}, sanitize.Message); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestAssembleMessage_NoSanitizerIdentity(t *testing.T) {
	// A no-op sanitizer leaves chars in place; useful for transports
	// that want assembly without sanitization (none today, but the
	// helper shouldn't assume sanitize is non-nil).
	identity := func(s string) string { return s }
	got := transport.AssembleMessage([]any{"a\nb"}, identity)
	if got != "a\nb" {
		t.Errorf("got %q, want %q", got, "a\nb")
	}
}
```

Add the import for `loglayer` at the top if it isn't already imported in this test file:

```go
import (
	"go.loglayer.dev/v2"  // add if not already imported
)
```

- [ ] **Step 2: Run to confirm they fail**

Run: `go test ./transport -run "TestAssembleMessage"`
Expected: build failure (`undefined: transport.AssembleMessage`).

- [ ] **Step 3: Implement `AssembleMessage`**

Append to `transport/helpers.go`:

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
// Adjacent elements are joined with " ". Empty messages produce "".
//
// Used by terminal-style transports (cli, pretty, console). Wrapper
// transports and JSON sinks call JoinMessages instead; the
// *MultilineMessage.String method handles flattening transparently
// for them.
func AssembleMessage(messages []any, sanitize func(string) string) string {
	if len(messages) == 0 {
		return ""
	}
	parts := make([]string, len(messages))
	for i, m := range messages {
		parts[i] = assembleElement(m, sanitize)
	}
	return strings.Join(parts, " ")
}

func assembleElement(v any, sanitize func(string) string) string {
	switch x := v.(type) {
	case *loglayer.MultilineMessage:
		lines := x.Lines()
		out := make([]string, len(lines))
		for i, l := range lines {
			out[i] = sanitize(l)
		}
		return strings.Join(out, "\n")
	case string:
		return sanitize(x)
	default:
		return sanitize(fmt.Sprintf("%v", v))
	}
}
```

- [ ] **Step 4: Run to confirm tests pass**

Run: `go test ./transport -run "TestAssembleMessage"`
Expected: PASS.

- [ ] **Step 5: Run the full suite**

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add transport/helpers.go transport/helpers_test.go
git commit -m "$(cat <<'EOF'
feat(transport): add AssembleMessage helper

Per-line sanitize-aware message assembler for terminal-style
transports. *MultilineMessage values render with authored "\n"
boundaries preserved; bare strings with embedded "\n" still strip.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 9: `AssembleMessage` adversarial smuggling tests

**Files:**
- Modify: `transport/helpers_test.go`

- [ ] **Step 1: Write the tests**

Append to `transport/helpers_test.go`:

```go
func TestAssembleMessage_CRStrippedFromLine(t *testing.T) {
	got := transport.AssembleMessage(
		[]any{loglayer.Multiline("a\r", "b")},
		sanitize.Message,
	)
	if got != "a\nb" {
		t.Errorf("got %q, want %q (CR must strip; LF boundary survives)", got, "a\nb")
	}
}

func TestAssembleMessage_AnsiSplitAcrossLinesCannotReconstruct(t *testing.T) {
	// Bare ESC byte alone in line 0; bracket sequence in line 1.
	// After per-line sanitize: line 0 -> "" (ESC strips), line 1 ->
	// "[31mred" (printable bracket text, no ESC anywhere). Joined
	// output has no actual ANSI escape. The smuggling fails.
	got := transport.AssembleMessage(
		[]any{loglayer.Multiline("\x1b", "[31mred")},
		sanitize.Message,
	)
	if got != "\n[31mred" {
		t.Errorf("got %q, want %q (ANSI must NOT reconstruct across lines)", got, "\n[31mred")
	}
}

func TestAssembleMessage_BidiOverrideStripped(t *testing.T) {
	// U+202E "right-to-left override"; Trojan Source attack.
	got := transport.AssembleMessage(
		[]any{loglayer.Multiline("‮", "evil")},
		sanitize.Message,
	)
	if got != "\nevil" {
		t.Errorf("got %q, want %q (bidi override must strip)", got, "\nevil")
	}
}

func TestAssembleMessage_ZeroWidthSpaceStripped(t *testing.T) {
	// U+200B inside a line.
	got := transport.AssembleMessage(
		[]any{loglayer.Multiline("zero​width", "y")},
		sanitize.Message,
	)
	if got != "zerowidth\ny" {
		t.Errorf("got %q, want %q (ZWSP must strip)", got, "zerowidth\ny")
	}
}
```

- [ ] **Step 2: Run to confirm they pass**

Run: `go test ./transport -run "TestAssembleMessage_"`
Expected: PASS: the previous step's implementation already satisfies these because per-line sanitize is intrinsic to the helper's design.

- [ ] **Step 3: Commit**

```bash
git add transport/helpers_test.go
git commit -m "$(cat <<'EOF'
test(transport): adversarial smuggling cases for AssembleMessage

Pin the per-line sanitize semantics: ANSI escape split across two
authored lines cannot reconstruct; CR strips inside a line; bidi
override and ZWSP strip; only the LF boundary between authored
elements survives.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 10: `JoinPrefixAndMessages`: `*MultilineMessage` branch

**Files:**
- Modify: `transport/helpers.go`
- Modify: `transport/helpers_test.go`

- [ ] **Step 1: Write the failing test**

Append to `transport/helpers_test.go`:

```go
func TestJoinPrefixAndMessages_MultilinePrependsToFirstLine(t *testing.T) {
	in := []any{loglayer.Multiline("a", "b", "c")}
	got := transport.JoinPrefixAndMessages("[p]", in)
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	m, ok := got[0].(*loglayer.MultilineMessage)
	if !ok {
		t.Fatalf("got[0] = %T, want *MultilineMessage", got[0])
	}
	want := []string{"[p] a", "b", "c"}
	if gotLines := m.Lines(); !reflect.DeepEqual(gotLines, want) {
		t.Errorf("lines = %#v, want %#v", gotLines, want)
	}
}

func TestJoinPrefixAndMessages_MultilineDoesNotMutateInput(t *testing.T) {
	original := loglayer.Multiline("a", "b")
	in := []any{original}
	_ = transport.JoinPrefixAndMessages("[p]", in)
	if got := original.Lines(); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Errorf("original mutated: %#v", got)
	}
}
```

Add the `reflect` import if absent.

- [ ] **Step 2: Run to confirm it fails**

Run: `go test ./transport -run "TestJoinPrefixAndMessages_Multiline"`
Expected: FAIL: current helper hits the `not a string` branch and returns `messages` unchanged, so the prefix never lands.

- [ ] **Step 3: Extend the helper**

Find `JoinPrefixAndMessages` in `transport/helpers.go` and replace its body:

```go
func JoinPrefixAndMessages(prefix string, messages []any) []any {
	if prefix == "" || len(messages) == 0 {
		return messages
	}
	out := make([]any, len(messages))
	copy(out, messages)
	switch v := messages[0].(type) {
	case *loglayer.MultilineMessage:
		lines := v.Lines()
		head := prefix + " " + lines[0]
		rebuilt := make([]string, len(lines))
		rebuilt[0] = head
		copy(rebuilt[1:], lines[1:])
		// Use the unexported field via the package-internal route in
		// the loglayer package; transport can't reach it directly. So
		// build a new wrapper through a helper added to loglayer.
		out[0] = loglayer.NewMultilineMessage(rebuilt)
	case string:
		out[0] = prefix + " " + v
	default:
		out[0] = prefix + " " + fmt.Sprintf("%v", v)
	}
	return out
}
```

This requires a small addition to `multiline.go` so `transport/` can build a wrapper from a slice without re-running construction-time normalization (we don't want to re-split or re-flatten; `lines` is already in canonical shape):

Append to `multiline.go`:

```go
// NewMultilineMessage wraps an already-canonicalized slice of lines.
// Use this only when you know the input has been processed by a prior
// Multiline call. The slice is taken without any normalization
// (no "\n" splitting, no nested flattening). The `transport` package
// uses this when folding a prefix into the first line of an existing
// wrapper without rebuilding the whole tree.
//
// Most callers should use [Multiline] instead.
func NewMultilineMessage(lines []string) *MultilineMessage {
	return &MultilineMessage{lines: lines}
}
```

- [ ] **Step 4: Run to confirm the new tests pass**

Run: `go test ./transport -run "TestJoinPrefixAndMessages_Multiline"`
Expected: PASS.

- [ ] **Step 5: Run the existing helper tests to confirm string + empty cases still pass**

Run: `go test ./transport -run "TestJoinPrefixAndMessages"`
Expected: PASS for the empty-prefix, nil-messages, empty-messages, and string cases. The pre-existing "non-string messages[0] returns input slice unchanged" subtest will still pass for now because we haven't generalized that branch; the next task does.

- [ ] **Step 6: Commit**

```bash
git add transport/helpers.go transport/helpers_test.go multiline.go
git commit -m "$(cat <<'EOF'
feat(transport): JoinPrefixAndMessages handles *MultilineMessage

When Messages[0] is a *MultilineMessage, the prefix folds into the
first authored line; subsequent lines are unchanged. Without this,
log.WithPrefix("X").Info(loglayer.Multiline(...)) would silently drop
the prefix because the helper's existing fallback returns Messages
unchanged for any non-string first arg.

Adds an internal NewMultilineMessage(lines []string) constructor for
reusing an already-canonicalized slice without re-running the
"\n"-split + flatten normalization.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 11: `JoinPrefixAndMessages`: generalize the non-string fallback

**Files:**
- Modify: `transport/helpers_test.go`

- [ ] **Step 1: Update the existing pinning test and add new ones**

Find this test in `transport/helpers_test.go`:

```go
t.Run("non-string messages[0] returns input slice unchanged", func(t *testing.T) {
	in := []any{42, "rest"}
	got := transport.JoinPrefixAndMessages("[p]", in)
	if len(got) != len(in) || got[0] != 42 || got[1] != "rest" {
		t.Errorf("non-string first arg should pass through: %v", got)
	}
})
```

Replace the block with:

```go
t.Run("non-string messages[0] folds prefix via %v formatting", func(t *testing.T) {
	in := []any{42, "rest"}
	got := transport.JoinPrefixAndMessages("[p]", in)
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
	if got[0] != "[p] 42" {
		t.Errorf("got[0] = %v, want %q", got[0], "[p] 42")
	}
	if got[1] != "rest" {
		t.Errorf("got[1] = %v, want %q", got[1], "rest")
	}
})
t.Run("Stringer messages[0] folds prefix via String()", func(t *testing.T) {
	type stringerVal struct{ v string }
	// Define inline as a method-bearing local type.
	got := transport.JoinPrefixAndMessages("[p]", []any{stringerImpl{v: "x"}})
	if got[0] != "[p] s:x" {
		t.Errorf("got[0] = %v, want %q", got[0], "[p] s:x")
	}
})
```

Then add the helper type at file scope (place near the top of `helpers_test.go`):

```go
type stringerImpl struct{ v string }

func (s stringerImpl) String() string { return "s:" + s.v }
```

- [ ] **Step 2: Run to confirm the tests fail (they pin behavior the current code doesn't yet have)**

Run: `go test ./transport -run "TestJoinPrefixAndMessages"`
Expected: FAIL: the existing helper's `string` type assertion still hits the early-return-unchanged path for `42` and `stringerImpl`.

- [ ] **Step 3: Generalize the helper's fallback**

The current helper (after Task 10) already has the `default:` branch that calls `fmt.Sprintf("%v", v)`. Verify it; no further code change is required if Task 10's edit was applied as written. If for any reason the `default:` branch is missing, restore it:

```go
default:
	out[0] = prefix + " " + fmt.Sprintf("%v", v)
```

- [ ] **Step 4: Run to confirm tests pass**

Run: `go test ./transport -run "TestJoinPrefixAndMessages"`
Expected: PASS for every subtest.

- [ ] **Step 5: Run the full repo test suite**

Run: `go test ./...`
Expected: PASS. If a wrapper transport's existing test was relying on the silent-drop behavior, it will surface here; investigate before silencing.

- [ ] **Step 6: Commit**

```bash
git add transport/helpers_test.go
git commit -m "$(cat <<'EOF'
fix(transport): JoinPrefixAndMessages folds prefix for non-string args

Pre-existing bug: when Messages[0] was not a string, the helper
returned Messages unchanged so the WithPrefix value was silently
dropped. Every caller flattens via %v downstream anyway, so the
prefix had no semantic reason to be omitted. The default branch
added in the previous commit folds the prefix in front of
fmt.Sprintf("%v", v); this commit updates the existing test that
pinned the broken behavior and adds a Stringer case.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 12: `cli` transport: `AssembleMessage` swap

**Files:**
- Modify: `transports/cli/cli.go`

- [ ] **Step 1: Write the driving test (it will fail)**

Append to `transports/cli/cli_test.go`:

```go
func TestCLI_MultilineRendersAcrossLines(t *testing.T) {
	var buf bytes.Buffer
	log := loglayer.New(loglayer.Config{
		Transport: cli.New(cli.Config{
			Stdout: &buf,
			Color:  cli.ColorNever,
		}),
	})
	log.Info(loglayer.Multiline("Header:", "  port: 8080", "  host: ::1"))
	got := buf.String()
	want := "Header:\n  port: 8080\n  host: ::1\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
```

Confirm the test file already imports `bytes`, `testing`, `go.loglayer.dev/v2`, and `go.loglayer.dev/v2/transports/cli`. Add any missing import.

- [ ] **Step 2: Run to confirm it fails**

Run: `go test ./transports/cli -run TestCLI_MultilineRendersAcrossLines`
Expected: FAIL: the existing CLI transport's `sanitizeMessages` strips `\n` from a Stringer-flattened `*MultilineMessage`, collapsing the output.

- [ ] **Step 3: Swap the call site**

In `transports/cli/cli.go`, find this line in `format`:

```go
msg := transport.JoinMessages(sanitizeMessages(params.Messages))
```

Replace with:

```go
msg := transport.AssembleMessage(params.Messages, sanitize.Message)
```

Then remove the now-unused `sanitizeMessages` helper (currently around line 382 onward) entirely:

```go
// sanitizeMessages scrubs CRLF and ANSI ESC from each string-shaped
// message so a user-controlled value can't smuggle terminal escapes
// or forge log lines.
func sanitizeMessages(in []any) []any {
	out := make([]any, len(in))
	for i, m := range in {
		if s, ok := m.(string); ok {
			out[i] = sanitize.Message(s)
			continue
		}
		out[i] = m
	}
	return out
}
```

- [ ] **Step 4: Run the cli sub-module's tests**

Run: `go test ./transports/cli/...`
Expected: PASS for `TestCLI_MultilineRendersAcrossLines` and every existing test (single-line rendering must still work).

- [ ] **Step 5: Run the full repo build**

Run: `bash scripts/foreach-module.sh test`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add transports/cli/cli.go transports/cli/cli_test.go
git commit -m "$(cat <<'EOF'
feat(transports/cli): adopt AssembleMessage for multi-line support

log.Info(loglayer.Multiline(...)) now renders authored "\n" boundaries
on the cli transport. Single-line messages are unchanged. The private
sanitizeMessages helper goes away since AssembleMessage covers its job
directly.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 13: `cli` transport: regression-guard, mixed, prefix, empty tests

**Files:**
- Modify: `transports/cli/cli_test.go`

- [ ] **Step 1: Write the tests**

Append to `transports/cli/cli_test.go`:

```go
func TestCLI_BareNewlineStringStillStripped(t *testing.T) {
	// No wrapper, no trust: a plain string with embedded "\n" still
	// renders on a single line.
	var buf bytes.Buffer
	log := loglayer.New(loglayer.Config{
		Transport: cli.New(cli.Config{Stdout: &buf, Color: cli.ColorNever}),
	})
	log.Info("a\nb")
	got := buf.String()
	want := "ab\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCLI_MixedStringAndMultiline(t *testing.T) {
	var buf bytes.Buffer
	log := loglayer.New(loglayer.Config{
		Transport: cli.New(cli.Config{Stdout: &buf, Color: cli.ColorNever}),
	})
	log.Info("Header:", loglayer.Multiline("a", "b"))
	got := buf.String()
	want := "Header: a\nb\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCLI_PrefixFoldsIntoFirstAuthoredLine(t *testing.T) {
	var buf bytes.Buffer
	log := loglayer.New(loglayer.Config{
		Transport: cli.New(cli.Config{Stdout: &buf, Color: cli.ColorNever}),
	}).WithPrefix("[svc]")
	log.Info(loglayer.Multiline("a", "b", "c"))
	got := buf.String()
	want := "[svc] a\nb\nc\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCLI_EmptyMultilineProducesNoOutput(t *testing.T) {
	// Matches the behavior of log.Info("") on cli (the empty-body
	// skip in SendToLogger short-circuits the Fprintln).
	var buf bytes.Buffer
	log := loglayer.New(loglayer.Config{
		Transport: cli.New(cli.Config{Stdout: &buf, Color: cli.ColorNever}),
	})
	log.Info(loglayer.Multiline())
	if got := buf.String(); got != "" {
		t.Errorf("expected no output, got %q", got)
	}
}
```

- [ ] **Step 2: Run to confirm they pass**

Run: `go test ./transports/cli -run "TestCLI_(BareNewline|MixedStringAndMultiline|PrefixFolds|EmptyMultiline)"`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add transports/cli/cli_test.go
git commit -m "$(cat <<'EOF'
test(transports/cli): regression-guard, mixed, prefix, empty cases

Pin the cli transport's contract around Multiline:
- bare "\n" string still strips (security regression-guard)
- mixed "Header:" + Multiline(...) produces "Header: line1\nline2"
- WithPrefix folds into the first authored line; later lines unchanged
- Multiline() with zero args emits no output (matches log.Info(""))

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 14: `pretty` transport: `AssembleMessage` swap + tests

**Files:**
- Modify: `transports/pretty/pretty.go`
- Modify: `transports/pretty/pretty_test.go`

- [ ] **Step 1: Write the driving test**

Append to `transports/pretty/pretty_test.go`:

```go
func TestPretty_MultilineRendersAcrossLines(t *testing.T) {
	var buf bytes.Buffer
	log := loglayer.New(loglayer.Config{
		Transport: pretty.New(pretty.Config{
			Writer: &buf,
			DateFn: func() string { return "TIME" },
			Color:  pretty.ColorNever,
		}),
	})
	log.Info(loglayer.Multiline("Header:", "  port: 8080"))
	got := buf.String()
	if !strings.Contains(got, "Header:\n  port: 8080") {
		t.Errorf("expected multi-line headline; got %q", got)
	}
}
```

Confirm the test file imports `bytes`, `strings`, `testing`, `go.loglayer.dev/v2`, and `go.loglayer.dev/v2/transports/pretty`. The exact `Config` field names and color knob may differ slightly from `cli`'s; check `transports/pretty/pretty.go` for the canonical fields and adjust the test accordingly.

- [ ] **Step 2: Run to confirm it fails**

Run: `go test ./transports/pretty -run TestPretty_MultilineRendersAcrossLines`
Expected: FAIL: `\n` stripped by current `sanitize.Message(transport.JoinMessages(...))` path.

- [ ] **Step 3: Swap the call site**

In `transports/pretty/pretty.go`, find:

```go
message := sanitize.Message(transport.JoinMessages(params.Messages))
```

Replace with:

```go
message := transport.AssembleMessage(params.Messages, sanitize.Message)
```

- [ ] **Step 4: Run the pretty sub-module's tests**

Run: `go test ./transports/pretty/...`
Expected: PASS for the new test and every existing one.

- [ ] **Step 5: Add the regression-guard, mixed, prefix, empty tests**

Append to `transports/pretty/pretty_test.go` (using the existing test file's helpers/factory pattern; the bodies below are templates; adapt to the existing `pretty.Config` field names):

```go
func TestPretty_BareNewlineStringStillStripped(t *testing.T) {
	var buf bytes.Buffer
	log := loglayer.New(loglayer.Config{
		Transport: pretty.New(pretty.Config{
			Writer: &buf,
			DateFn: func() string { return "TIME" },
			Color:  pretty.ColorNever,
		}),
	})
	log.Info("a\nb")
	got := buf.String()
	if strings.Contains(got, "\n") && strings.Count(got, "\n") > 1 {
		t.Errorf("bare \\n must strip; rendered: %q", got)
	}
}

func TestPretty_MixedStringAndMultiline(t *testing.T) {
	var buf bytes.Buffer
	log := loglayer.New(loglayer.Config{
		Transport: pretty.New(pretty.Config{
			Writer: &buf,
			DateFn: func() string { return "TIME" },
			Color:  pretty.ColorNever,
		}),
	})
	log.Info("Header:", loglayer.Multiline("a", "b"))
	got := buf.String()
	if !strings.Contains(got, "Header: a\nb") {
		t.Errorf("expected mixed multi-line; got %q", got)
	}
}

func TestPretty_PrefixFoldsIntoFirstAuthoredLine(t *testing.T) {
	var buf bytes.Buffer
	log := loglayer.New(loglayer.Config{
		Transport: pretty.New(pretty.Config{
			Writer: &buf,
			DateFn: func() string { return "TIME" },
			Color:  pretty.ColorNever,
		}),
	}).WithPrefix("[svc]")
	log.Info(loglayer.Multiline("a", "b"))
	got := buf.String()
	if !strings.Contains(got, "[svc] a\nb") {
		t.Errorf("expected prefix on first line; got %q", got)
	}
}
```

- [ ] **Step 6: Run to confirm they pass**

Run: `go test ./transports/pretty/...`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add transports/pretty/pretty.go transports/pretty/pretty_test.go
git commit -m "$(cat <<'EOF'
feat(transports/pretty): adopt AssembleMessage for multi-line support

Same pattern as transports/cli: AssembleMessage replaces the
sanitize.Message(JoinMessages(...)) call so *MultilineMessage values
render with authored line boundaries while bare strings still strip.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 15: `console` transport: `MessageField` mode swap

**Files:**
- Modify: `transports/console/console.go`
- Modify: `transports/console/console_test.go`

- [ ] **Step 1: Write the driving test for MessageField mode**

Append to `transports/console/console_test.go`:

```go
func TestConsole_MultilineInMessageFieldMode(t *testing.T) {
	var buf bytes.Buffer
	log := loglayer.New(loglayer.Config{
		Transport: console.New(console.Config{
			Writer:       &buf,
			MessageField: "msg",
			DateFn:       func() string { return "TIME" },
		}),
	})
	log.Info(loglayer.Multiline("a", "b"))
	var obj map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &obj); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if obj["msg"] != "a\nb" {
		t.Errorf("msg = %v, want %q", obj["msg"], "a\nb")
	}
}
```

Add `encoding/json` and any missing imports.

- [ ] **Step 2: Run to confirm it fails**

Run: `go test ./transports/console -run TestConsole_MultilineInMessageFieldMode`
Expected: FAIL: current code calls `JoinMessages` after a per-element string-only sanitize; `*MultilineMessage` falls through to `%v` (Stringer), and the per-element sanitize doesn't touch it. After `JoinMessages`, the `\n` survives, but the existing per-element sanitize loop above does NOT touch the `*MultilineMessage` value (only strings), so the bug is that adversarial strings inside Multiline lines are not sanitized. Confirm the failure mode: it may be that the test passes by accident on this specific input but the per-line sanitize is missing. Read the failure carefully.

If the assertion passes on the simple input, the test is right but doesn't exercise the security gap. Strengthen the test by appending an adversarial case:

```go
func TestConsole_MultilineLinesAreSanitized(t *testing.T) {
	var buf bytes.Buffer
	log := loglayer.New(loglayer.Config{
		Transport: console.New(console.Config{
			Writer:       &buf,
			MessageField: "msg",
			DateFn:       func() string { return "TIME" },
		}),
	})
	log.Info(loglayer.Multiline("clean", "evil\x1b[31mred"))
	var obj map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &obj); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if obj["msg"] != "clean\nevilred" {
		t.Errorf("msg = %v, want %q (per-line sanitize must strip ESC)", obj["msg"], "clean\nevilred")
	}
}
```

- [ ] **Step 3: Swap the call site for MessageField mode**

In `transports/console/console.go`, find the `MessageField` branch in `buildMessages` (around line 130):

```go
obj[cfg.MessageField] = transport.JoinMessages(messages)
```

Replace with:

```go
obj[cfg.MessageField] = transport.AssembleMessage(params.Messages, sanitize.Message)
```

(Use `params.Messages` (the original slice) rather than the post-loop `messages`, since `AssembleMessage` does its own per-element sanitization.)

- [ ] **Step 4: Run to confirm the new test passes**

Run: `go test ./transports/console -run "TestConsole_Multiline"`
Expected: PASS.

- [ ] **Step 5: Run all existing console tests**

Run: `go test ./transports/console/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add transports/console/console.go transports/console/console_test.go
git commit -m "$(cat <<'EOF'
feat(transports/console): MessageField mode honors Multiline

Direct AssembleMessage swap: the MessageField branch of buildMessages
now produces a "\n"-joined message with per-line sanitization. JSON
encoders escape the literal "\n" to "\\n" in the wire output.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 16: `console` transport: default (logfmt) mode swap

**Files:**
- Modify: `transports/console/console.go`
- Modify: `transports/console/console_test.go`

- [ ] **Step 1: Write the driving tests**

Append to `transports/console/console_test.go`:

```go
func TestConsole_MultilineDefaultMode(t *testing.T) {
	// Default mode (no MessageField): the headline + optional logfmt
	// is space-joined on one Fprintln. After the swap, the assembled
	// headline contains a literal "\n" between authored lines, then a
	// single trailing "\n" from Fprintln.
	var buf bytes.Buffer
	log := loglayer.New(loglayer.Config{
		Transport: console.New(console.Config{
			Writer: &buf,
			DateFn: func() string { return "TIME" },
		}),
	})
	log.Info(loglayer.Multiline("a", "b"))
	got := buf.String()
	want := "a\nb\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestConsole_DefaultModeMultilineWithLogfmtFields(t *testing.T) {
	var buf bytes.Buffer
	log := loglayer.New(loglayer.Config{
		Transport: console.New(console.Config{
			Writer: &buf,
			DateFn: func() string { return "TIME" },
		}),
	})
	log.WithFields(loglayer.Fields{"k": "v"}).Info(loglayer.Multiline("a", "b"))
	got := buf.String()
	// Body assembles as "a\nb"; logfmt suffix attaches with a single
	// space separator. Fprintln adds the trailing newline.
	want := "a\nb k=v\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestConsole_DefaultModeBareNewlineStillStripped(t *testing.T) {
	var buf bytes.Buffer
	log := loglayer.New(loglayer.Config{
		Transport: console.New(console.Config{
			Writer: &buf,
			DateFn: func() string { return "TIME" },
		}),
	})
	log.Info("a\nb")
	got := buf.String()
	want := "ab\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
```

- [ ] **Step 2: Run to confirm they fail**

Run: `go test ./transports/console -run "TestConsole_(MultilineDefaultMode|DefaultModeMultilineWithLogfmt|DefaultModeBareNewline)"`
Expected: FAIL: the default mode still uses the per-element sanitize loop and `Fprintln(messages...)` which doesn't apply the per-line sanitize that `AssembleMessage` provides for `*MultilineMessage` cells.

- [ ] **Step 3: Refactor `buildMessages` for the default mode**

In `transports/console/console.go`, replace the entire `buildMessages` function with the following shape. The key structural change is: assemble the message portion to a single string up-front, then attach logfmt / Stringify suffix to that single string, returning `[]any{single}` so the call site's `Fprintln(c.writer(level), messages...)` produces byte-equivalent output.

```go
// buildMessages assembles the argument list to pass to Fprintln. In the
// default mode, fields and metadata render as logfmt (key=value, key=value)
// after the message. MessageField and Stringify switch to single-object
// output for callers that want a structured-but-non-pipeline shape.
func buildMessages(params loglayer.TransportParams, cfg Config) []any {
	// Apply MessageFn override before Multiline-aware assembly so the
	// override produces the canonical input (a single string).
	rawMessages := params.Messages
	if cfg.MessageFn != nil {
		rawMessages = []any{cfg.MessageFn(params)}
	}

	// Per-line, Multiline-aware sanitize-and-join for the headline.
	headline := transport.AssembleMessage(rawMessages, sanitize.Message)

	combined := transport.MergeFieldsAndMetadata(params)

	// MessageField: single structured object as the sole arg.
	if cfg.MessageField != "" {
		obj := make(map[string]any, len(combined)+3)
		for k, v := range combined {
			obj[k] = v
		}
		obj[cfg.MessageField] = headline
		if cfg.DateField != "" {
			obj[cfg.DateField] = dateValue(cfg)
		}
		if cfg.LevelField != "" {
			obj[cfg.LevelField] = levelValue(cfg, params.LogLevel)
		}
		return []any{maybeStringify(obj, cfg.Stringify)}
	}

	// Bake in date/level as additional logfmt keys when configured.
	if cfg.DateField != "" || cfg.LevelField != "" {
		if combined == nil {
			combined = make(map[string]any, 2)
		}
		if cfg.DateField != "" {
			combined[cfg.DateField] = dateValue(cfg)
		}
		if cfg.LevelField != "" {
			combined[cfg.LevelField] = levelValue(cfg, params.LogLevel)
		}
		// Stringify: emit a JSON object after the message instead of logfmt.
		if cfg.Stringify {
			if headline == "" {
				return []any{maybeStringify(combined, true)}
			}
			return []any{headline, maybeStringify(combined, true)}
		}
	}

	if len(combined) > 0 {
		suffix := renderLogfmt(combined)
		if headline == "" {
			return []any{suffix}
		}
		return []any{headline, suffix}
	}
	if headline == "" {
		return nil
	}
	return []any{headline}
}
```

A few considerations to verify before committing:

- **Empty headline case:** `Fprintln` on `[]any{}` would write only `"\n"`. The previous code would print only the logfmt or stringify suffix (no leading blank). Returning `nil` from `buildMessages` produces `Fprintln()` with no args → just `"\n"`, which is a regression for `log.MetadataOnly(...)` calls. Audit existing console tests for `MetadataOnly` and `ErrorOnly` semantics; the `[]any{suffix}` and `nil` branches above need to match the previous behavior. If the previous `buildMessages` returned `messages` (which could be empty if the only thing was a logfmt suffix), align with that exact shape. **Adjust the function above as needed** by tracing each path against the existing tests.

- **MessageFn override semantics:** the previous code applied `MessageFn` then ran the per-element sanitize loop, which sanitized the MessageFn output. The version above passes the override result through `AssembleMessage`, which applies `sanitize.Message` per element. Behavior is byte-equivalent for the single-string output of MessageFn.

- [ ] **Step 4: Run the new tests + every existing console test**

Run: `go test ./transports/console/...`
Expected: PASS for new and existing tests. If any existing test fails, trace the previous code shape and adjust.

- [ ] **Step 5: Commit**

```bash
git add transports/console/console.go transports/console/console_test.go
git commit -m "$(cat <<'EOF'
feat(transports/console): default (logfmt) mode honors Multiline

Replaces the per-element sanitize loop with AssembleMessage so the
headline is assembled with per-line sanitization and authored "\n"
boundaries preserved. Logfmt and Stringify suffixes attach to the
assembled headline as a second Fprintln arg, byte-equivalent to the
previous output for single-line messages.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 17: `transporttest` contract: `Multiline` scenario

**Files:**
- Modify: `transport/transporttest/contract.go`

- [ ] **Step 1: Write the contract scenario as a new test function**

Append to `transport/transporttest/contract.go`:

```go
func testMultilineRendersAcrossLines(t *testing.T, c ContractCase) {
	log, buf := c.Factory(FactoryOpts{})
	log.Info(loglayer.Multiline("line1", "line2", "line3"))
	obj := ParseJSONLine(t, buf)
	got, ok := obj[c.Expect.MessageKey].(string)
	if !ok {
		t.Fatalf("%s: missing or non-string; got %T = %v", c.Expect.MessageKey, obj[c.Expect.MessageKey], obj[c.Expect.MessageKey])
	}
	want := "line1\nline2\nline3"
	if got != want {
		t.Errorf("%s: got %q, want %q", c.Expect.MessageKey, got, want)
	}
}

func testMultilineWithPrefixFoldsToFirstLine(t *testing.T, c ContractCase) {
	log, buf := c.Factory(FactoryOpts{})
	prefixed := log.WithPrefix("[svc]")
	prefixed.Info(loglayer.Multiline("a", "b", "c"))
	obj := ParseJSONLine(t, buf)
	got, ok := obj[c.Expect.MessageKey].(string)
	if !ok {
		t.Fatalf("%s: missing or non-string; got %T = %v", c.Expect.MessageKey, obj[c.Expect.MessageKey], obj[c.Expect.MessageKey])
	}
	want := "[svc] a\nb\nc"
	if got != want {
		t.Errorf("%s: got %q, want %q", c.Expect.MessageKey, got, want)
	}
}
```

- [ ] **Step 2: Wire the new scenarios into `RunContract`**

Find the `RunContract` function (lines 54-73) and add two new `t.Run` entries near the other message-shape tests:

```go
t.Run(c.Name+"/Multiline", func(t *testing.T) { t.Parallel(); testMultilineRendersAcrossLines(t, c) })
t.Run(c.Name+"/MultilineWithPrefix", func(t *testing.T) { t.Parallel(); testMultilineWithPrefixFoldsToFirstLine(t, c) })
```

- [ ] **Step 3: Run every consumer of `RunContract`**

Run: `bash scripts/foreach-module.sh test`
Expected: PASS for every wrapper transport that calls `RunContract`. The wrappers (zerolog/zap/slog/logrus/charmlog/phuslu/sentry/otellog/gcplogging/http/datadog/structured/lumberjack/testing) should produce `"line1\nline2\nline3"` because `JoinMessages` calls `fmt.Sprintf("%v", *MultilineMessage)` which calls `String()`, and the wrappers don't sanitize.

If any wrapper fails: investigate per-transport. Likely causes: a wrapper that joins messages by something other than `JoinMessages`, or a transport that strips control characters in its own pipeline. The fix is per-transport; do not silently exclude failing wrappers from the contract.

- [ ] **Step 4: Commit**

```bash
git add transport/transporttest/contract.go
git commit -m "$(cat <<'EOF'
test(transport/transporttest): Multiline + prefix contract scenarios

Every wrapper transport that calls RunContract picks up two new cases:
- log.Info(loglayer.Multiline("line1","line2","line3")) emits a
  "line1\nline2\nline3" message via the Stringer fallback.
- log.WithPrefix("[svc]").Info(Multiline("a","b","c")) folds the
  prefix into the first authored line; later lines unchanged.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 18: Main module `ExampleMultiline`

**Files:**
- Modify: `example_test.go`

- [ ] **Step 1: Write the example**

Append to `example_test.go`:

```go
// ExampleMultiline shows how to author a multi-line message that
// survives terminal-renderer sanitization. The Multiline wrapper is a
// developer-issued token of trust: terminal transports (cli, pretty,
// console) preserve "\n" boundaries between authored elements while
// still sanitizing ANSI / control bytes inside each line.
func ExampleMultiline() {
	log := loglayer.New(loglayer.Config{
		Transport: exampleTransport{id: "ex"},
	})
	log.Info(loglayer.Multiline("Header:", "  port: 8080", "  host: ::1"))
	// Output:
	// {"level":"info","time":"2026-04-26T12:00:00Z","msg":"Header:\n  port: 8080\n  host: ::1"}
}
```

- [ ] **Step 2: Run the example to confirm the `// Output:` line matches**

Run: `go test -run ExampleMultiline`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add example_test.go
git commit -m "$(cat <<'EOF'
docs: add ExampleMultiline to main module example_test.go

Renders against the existing exampleTransport (JSON-shaped, fixed
time field) so // Output is deterministic. Mirrors the existing
examples in the file.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 19: `cli` sub-module Example

**Files:**
- Modify: `transports/cli/example_test.go`

- [ ] **Step 1: Add the example**

Append to `transports/cli/example_test.go`:

```go
// ExampleNew_multiline shows the cli transport rendering authored
// multi-line content. cli.Color is set to ColorNever so the output
// is deterministic across TTY / non-TTY runs.
func ExampleNew_multiline() {
	log := loglayer.New(loglayer.Config{
		Transport: cli.New(cli.Config{
			Stdout: os.Stdout,
			Color:  cli.ColorNever,
		}),
	})
	log.Info(loglayer.Multiline("Configuration:", "  port: 8080", "  host: ::1"))
	// Output:
	// Configuration:
	//   port: 8080
	//   host: ::1
}
```

Confirm `os` is imported. Add it if not.

- [ ] **Step 2: Run to confirm the Output matches**

Run: `go test ./transports/cli -run ExampleNew_multiline`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add transports/cli/example_test.go
git commit -m "$(cat <<'EOF'
docs(transports/cli): add Multiline Example for godoc

Demonstrates the cli transport's authored-newline rendering. Uses
ColorNever so output is deterministic.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 20: New doc page `docs/src/logging-api/multiline.md`

**Files:**
- Create: `docs/src/logging-api/multiline.md`

- [ ] **Step 1: Write the page**

Create `docs/src/logging-api/multiline.md`:

```markdown
---
title: Multi-line messages with loglayer.Multiline
description: "Author multi-line message content that survives the cli/pretty/console sanitizer"
---

# Multi-line messages with `loglayer.Multiline`

`loglayer.Multiline(lines ...any)` lets you author a message that renders on multiple lines through `cli`, `pretty`, and `console`. It's a developer-issued token of trust: the wrapper signals that the line boundaries between elements were authored by you, so the sanitizer in those transports preserves the `\n` between them while still stripping ANSI / control bytes inside each line.

## Quickstart

```go
import "go.loglayer.dev/v2"

log.Info(loglayer.Multiline(
    "Configuration:",
    "  port: 8080",
    "  host: ::1",
))
// Configuration:
//   port: 8080
//   host: ::1
```

`Multiline` accepts any number of arguments and treats each one as a separate authored line. Non-string arguments are formatted with `fmt.Sprintf("%v", v)` (Stringer is honored). Strings containing embedded `\n` are split at construction, so `Multiline("a\nb")` and `Multiline("a", "b")` are interchangeable.

## Why bare `\n` doesn't work

If you write `log.Info("Header:\n  port: 8080")` without the wrapper, the cli, pretty, and console transports collapse it to one line:

```
Header:  port: 8080
```

The sanitizer at those rendering boundaries strips `\n` from message strings to defeat two attacks:

1. **Log forging:** untrusted input containing `\n` could write fake follow-up log lines that look like they came from your app.
2. **Terminal escape smuggling:** untrusted input containing ANSI ESC, bidi overrides (Trojan Source), or zero-width joiners could inject color codes, hide content, or exploit terminal vulns.

`Multiline` opts you out of the line-collapsing rule for this *one* call, while keeping every other defense intact. Each authored line is still individually sanitized.

## What's preserved, what's stripped

| Inside one authored line | Across authored lines |
|---|---|
| ANSI ESC: stripped | `\n` boundary: preserved |
| CR: stripped | |
| Bidi overrides (U+202E etc.): stripped | |
| Zero-width joiners / spaces: stripped | |

A bare string with `\n` (no wrapper, no trust) still has the `\n` stripped. `Multiline("\x1b", "[31mred")` cannot reconstruct an ANSI escape across the boundary because each line is sanitized in isolation before joining.

## Per-transport behavior

| Transport | `Multiline("a","b")` | `"Header:", Multiline("a","b")` |
|---|---|---|
| **cli** | `a\nb` (level-colored, on the level's writer) | `Header: a\nb` |
| **pretty** | `[ts] [INFO] a\nb` | `[ts] [INFO] Header: a\nb` |
| **console** | `{"msg":"a\nb",...}` (MessageField mode) or `a\nb [k=v ...]` (default) | analogous |
| **structured** | `{"msg":"a\nb",...}` | `{"msg":"Header: a\nb",...}` |
| **zerolog / zap / slog / logrus / charmlog / phuslu** | underlying logger writes `"a\nb"` | underlying logger writes `"Header: a\nb"` |
| **sentry / otellog / gcplogging / http / datadog / testing** | same: `Stringer` fallback joins with `"\n"` | same |

## With a prefix

`WithPrefix` folds the prefix into the first authored line; subsequent lines are unchanged.

```go
log.WithPrefix("[svc]").Info(loglayer.Multiline("a", "b"))
// [svc] a
// b
```

## Inside fields or metadata

::: warning Messages-only in v1
`Multiline` only applies inside the message slice (`log.Info(...)`, `log.Error(...)`, etc.). Inside `Fields` or `Metadata`, terminal transports still sanitize each value to a single line, so a `Multiline` value placed in metadata gets rendered as a single line on cli/pretty/console.

JSON sinks (structured + every wrapper transport) serialize `Multiline` values via `MarshalJSON` to the `\n`-joined string, so no data is silently lost there.

If you need multi-line value rendering for fields specifically, file an issue describing the use case; the right shape is a separate design (probably routing through pretty's expanded-YAML mode).
:::

## Plugin interactions

Plugin hooks that walk `params.Messages` see the typed value. Most hooks don't need to care: calling `transport.JoinMessages(params.Messages)` flattens correctly via `Stringer`.

The one footgun: if you combine `Multiline` with the `fmtlog` plugin's format-string mode (`log.Info("data: %v", loglayer.Multiline(...))`), `fmt.Sprintf` resolves the wrapper to its `String()` value before the message reaches the transport. The trust signal is lost, and downstream sanitize strips the inner `\n`. To preserve the multi-line shape, construct the wrapper with the formatted lines yourself instead of letting `fmtlog` do it.
```

- [ ] **Step 2: Build the docs site to verify clean**

Run: `cd docs && bun run docs:build`
Expected: PASS (clean build).

- [ ] **Step 3: Commit**

```bash
git add docs/src/logging-api/multiline.md
git commit -m "$(cat <<'EOF'
docs: add logging-api/multiline.md page

Walks through the security rationale, the per-transport behavior, the
WithPrefix interaction, the v1 messages-only scope, and the plugin
interaction note (fmtlog format-string collapse).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 21: Cheatsheet, sidebar, llms.txt updates

**Files:**
- Modify: `docs/src/cheatsheet.md`
- Modify: `docs/.vitepress/config.ts`
- Modify: `docs/src/public/llms.txt`
- Modify: `docs/src/public/llms-full.txt`

- [ ] **Step 1: Add `Multiline` to the cheatsheet**

In `docs/src/cheatsheet.md`, find the section that lists `Lazy` (search for `loglayer.Lazy`). Add a parallel row:

```markdown
| `loglayer.Multiline(lines ...any)` | Wrap multiple lines so terminal transports render them on separate rows. See [Multi-line messages](/logging-api/multiline). |
```

(Adjust to match the surrounding cheatsheet's exact column shape.)

- [ ] **Step 2: Add the sidebar entry**

In `docs/.vitepress/config.ts`, find the `Logging API` sidebar section. Add an entry pointing at the new page:

```typescript
{ text: 'Multi-line messages', link: '/logging-api/multiline' },
```

Place it alphabetically or grouped with similar concept pages, matching the existing convention in that file.

- [ ] **Step 3: Add the surface to `llms.txt` and `llms-full.txt`**

In `docs/src/public/llms.txt`, find the Logging API section and add:

```
- loglayer.Multiline(lines ...any): wrap multi-line message content that survives terminal sanitize. https://go.loglayer.dev/logging-api/multiline
```

In `docs/src/public/llms-full.txt`, find the parallel section and add a slightly more detailed bullet:

```
- loglayer.Multiline(lines ...any) returns a *MultilineMessage that
  terminal-renderer transports (cli, pretty, console) interpret as
  authored "\n" boundaries. JSON sinks and wrapper transports flatten
  via Stringer/MarshalJSON. Each authored line is still sanitized
  individually. v1 is messages-only; metadata/fields values still
  collapse to one line in terminal renderers.
  https://go.loglayer.dev/logging-api/multiline
```

- [ ] **Step 4: Build the docs site**

Run: `cd docs && bun run docs:build`
Expected: PASS, with the new sidebar entry visible.

- [ ] **Step 5: Commit**

```bash
git add docs/src/cheatsheet.md docs/.vitepress/config.ts docs/src/public/llms.txt docs/src/public/llms-full.txt
git commit -m "$(cat <<'EOF'
docs: surface Multiline in cheatsheet, sidebar, llms.txt, llms-full.txt

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 22: `whats-new.md` entry

**Files:**
- Modify: `docs/src/whats-new.md`

- [ ] **Step 1: Add today's entry**

In `docs/src/whats-new.md`, add a new section at the top (under the intro bullet, above the most recent existing date heading) following the format in `.claude/rules/documentation.md`:

```markdown
## May 2, 2026

`loglayer`:

**`loglayer.Multiline(lines ...any)`** is a new value-wrapper that lets terminal transports preserve authored "\n" boundaries. The cli, pretty, and console transports collapse bare-string newlines to one line for security (log-forging, terminal-escape smuggling); the wrapper is a per-call developer-issued opt-in to that defense. Each authored line is still individually sanitized; only the boundaries between them are honored. JSON sinks and wrapper transports flatten via `Stringer` / `MarshalJSON` with no code change. See [Multi-line messages](/logging-api/multiline).

The change also fixes a pre-existing bug in `transport.JoinPrefixAndMessages` where `WithPrefix` was silently dropped when `Messages[0]` was not a string. The prefix now folds in front of the `%v`-formatted first message.
```

- [ ] **Step 2: Build the docs site**

Run: `cd docs && bun run docs:build`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add docs/src/whats-new.md
git commit -m "$(cat <<'EOF'
docs(whats-new): add May 2 2026 entry for Multiline and prefix fix

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 23: `creating-plugins.md` Multiline preservation note

**Files:**
- Modify: `docs/src/plugins/creating-plugins.md`

- [ ] **Step 1: Add the section**

In `docs/src/plugins/creating-plugins.md`, find the section that discusses message-mutating hooks (search for `MessageHook` or "Messages"). Append the following paragraph in an appropriate spot:

```markdown
### Preserving `*MultilineMessage` values

If your plugin walks `params.Messages` and replaces elements, be careful with `*loglayer.MultilineMessage` values. The wrapper is a developer-issued token of trust that lets terminal transports preserve authored "\n" boundaries. If your hook flattens it to a `string` (e.g., via `fmt.Sprintf` or `transport.JoinMessages`), the trust signal is lost and downstream terminal sanitize strips the inner "\n".

To preserve the multi-line shape, pass `*MultilineMessage` values through unchanged, or rebuild a new wrapper at the end of your transformation:

```go
out := make([]any, 0, len(p.Messages))
for _, m := range p.Messages {
    if ml, ok := m.(*loglayer.MultilineMessage); ok {
        out = append(out, ml) // pass through
        continue
    }
    out = append(out, transformString(m))
}
```

The built-in `fmtlog` plugin's format-string mode is one example where the wrapper is intentionally collapsed: `log.Info("data: %v", loglayer.Multiline(...))` runs `fmt.Sprintf` on the wrapper, which calls `String()` and yields a flat string. Document this trade-off in your plugin's GoDoc if it applies.
```

- [ ] **Step 2: Build the docs site**

Run: `cd docs && bun run docs:build`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add docs/src/plugins/creating-plugins.md
git commit -m "$(cat <<'EOF'
docs(plugins): note for plugin authors on preserving Multiline

Hooks that mutate params.Messages should pass *MultilineMessage values
through unchanged, or rebuild a new wrapper. Flattening to a string
loses the trust signal and the inner "\n" gets stripped downstream.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 24: Changeset

**Files:**
- Create: `.changeset/multiline-message.md`

- [ ] **Step 1: Write the changeset**

Create `.changeset/multiline-message.md` with the exact content below (per the spec's Changeset section). The frontmatter names the root and the three sub-modules whose code changed; the body documents the prefix-handling behavior change.

```markdown
---
"go.loglayer.dev": minor
"transports/cli": minor
"transports/pretty": minor
"transports/console": minor
---

Add `loglayer.Multiline(lines ...any)` for authoring multi-line message
content that survives terminal-renderer sanitization. The wrapper is
messages-only in v1; field/metadata values are still sanitized to a
single line in terminal transports (JSON sinks serialize via
`MarshalJSON` to the joined string).

Also fixes a pre-existing bug in `transport.JoinPrefixAndMessages`
where a `WithPrefix` value was silently dropped when `Messages[0]`
was not a string (e.g. `log.WithPrefix("X").Info(42)` lost the
prefix). The prefix now folds in front of the `%v`-formatted first
message.

See https://go.loglayer.dev/logging-api/multiline.
```

- [ ] **Step 2: Validate the file with monorel**

Run: `monorel preview --check`
Expected: shows the planned per-package version bumps without error. (If `monorel` isn't on the local PATH, skip this step; CI's `release-pr.yml` will catch any malformed changeset.)

- [ ] **Step 3: Commit**

```bash
git add .changeset/multiline-message.md
git commit -m "$(cat <<'EOF'
chore: add changeset for loglayer.Multiline

Names go.loglayer.dev plus the three sub-modules with call-site swaps
(cli, pretty, console) so the feature ships atomically.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 25: Final verification

**Files:** none (verification only)

- [ ] **Step 1: Run the full per-module test suite**

Run: `bash scripts/foreach-module.sh test`
Expected: PASS for every module (root + every transport sub-module + every plugin sub-module + every integration sub-module).

- [ ] **Step 2: Run gofmt across the touched files**

Run: `gofmt -l multiline.go transport/helpers.go transports/cli/cli.go transports/pretty/pretty.go transports/console/console.go`
Expected: no output (no files need formatting).

If any file needs formatting, run `gofmt -w <file>` and create a follow-up commit.

- [ ] **Step 3: Run `go vet` and `staticcheck` per-module**

Run: `bash scripts/foreach-module.sh vet` and `bash scripts/foreach-module.sh staticcheck`
Expected: PASS.

- [ ] **Step 4: Build the docs site**

Run: `cd docs && bun run docs:build`
Expected: PASS.

- [ ] **Step 5: Run `go test -bench=. -benchmem -run=^$ -benchtime=1x .` to confirm no obvious benchmark regressions**

Run: `go test -bench=. -benchmem -run=^$ -benchtime=1x .`
Expected: benchmarks run cleanly. Compare top-line numbers (`SimpleMessage`, `WithFields`, `WithError`) against the previous numbers in `docs/src/benchmarks.md`. Significant regression (>5%) requires investigation; the new core code should not affect the SimpleMessage path because Multiline only triggers when the wrapper is constructed.

- [ ] **Step 6: Final dispatch to a code-reviewer subagent for the whole branch**

Per `.claude/rules/git-workflow.md`, before opening the PR run a code review on the full branch range. Dispatch via the `superpowers:code-reviewer` agent with a self-contained brief covering: the design spec, the implementation plan, and the SHA range from the first task's commit through `HEAD`.

Apply any Critical / Important findings before pushing.

- [ ] **Step 7: Open the PR**

Per `.claude/rules/git-workflow.md`:

```bash
git fetch origin main
git rebase origin/main  # or --onto for stacked branches
git push -u origin <branch>
gh pr create --title "feat: loglayer.Multiline for multi-line messages" --body "$(cat <<'EOF'
## Summary

- New `loglayer.Multiline(lines ...any)` constructor and `*MultilineMessage` type. Developer-issued token of trust that lets terminal transports (cli, pretty, console) preserve authored `\n` boundaries while keeping per-line ANSI / CR / bidi / ZWJ sanitization intact.
- New `transport.AssembleMessage(messages, sanitize)` helper.
- Pre-existing `transport.JoinPrefixAndMessages` bug fixed: `WithPrefix` no longer silently drops when `Messages[0]` is non-string.
- Doc page: https://go.loglayer.dev/logging-api/multiline

## Test plan

- [ ] `bash scripts/foreach-module.sh test` passes
- [ ] `cd docs && bun run docs:build` passes
- [ ] Manual: `log.Info(loglayer.Multiline("a","b"))` renders multi-line on cli, pretty, console
- [ ] Manual: bare `"a\nb"` still strips on cli, pretty, console (regression-guard)
- [ ] Manual: `WithPrefix(...) + Multiline(...)` folds prefix into first line

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

---

## Self-review checklist (run before submitting the PR)

After completing every task, run through this list to make sure nothing was missed:

- [ ] **Spec coverage:** every section of `docs/superpowers/specs/2026-05-02-multiline-message-design.md` maps to at least one task.
- [ ] **Public API:** `loglayer.Multiline`, `*MultilineMessage`, `Lines()`, `String()`, `MarshalJSON()` all exist and are tested.
- [ ] **Helper API:** `transport.AssembleMessage`, `transport.JoinPrefixAndMessages` (extended), and the internal `loglayer.NewMultilineMessage` all exist and are tested.
- [ ] **Call-site swaps:** cli, pretty, console all use `AssembleMessage` (or the appropriate equivalent for console's two modes).
- [ ] **Contract test:** `transport/transporttest/contract.go` includes `Multiline` and `MultilineWithPrefix` scenarios; every wrapper that calls `RunContract` picks them up.
- [ ] **Examples:** `ExampleMultiline` in main module, `ExampleNew_multiline` (or analogous) in `transports/cli`.
- [ ] **Docs:** `docs/src/logging-api/multiline.md`, cheatsheet entry, sidebar entry, `whats-new.md` entry, `llms.txt` + `llms-full.txt`, `creating-plugins.md` note.
- [ ] **Changeset:** `.changeset/multiline-message.md` names root + cli + pretty + console with `:minor` bumps and documents the prefix-handling behavior change.
- [ ] **No em dashes** anywhere in the new docs (per `.claude/rules/documentation.md`).
- [ ] **Conventional Commits** formatting on every commit message.
