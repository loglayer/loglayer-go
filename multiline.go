package loglayer

import (
	"encoding/json"
	"fmt"
	"strings"
)

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
//  1. Non-string args convert via fmt.Sprintf("%v", v).
//  2. *MultilineMessage args flatten: their Lines() append into the
//     outer's slice.
//  3. Every resulting string is split on "\n", and each piece becomes
//     one entry of Lines(). After this step, no Lines() entry contains
//     an embedded "\n".
//
// The split rule means Multiline("a\nb") and Multiline("a","b") are
// interchangeable. CRLF input (e.g. "a\r\nb") splits to ["a\r","b"]
// and the trailing "\r" is stripped by per-line sanitization in
// terminal transports, yielding the same rendered output as
// Multiline("a","b").
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

// NewMultilineMessage wraps an already-canonicalized slice of lines.
// Use this only when you know the input has been processed by a prior
// Multiline call. The slice is taken without any normalization
// (no "\n" splitting, no nested flattening). The transport package
// uses this when folding a prefix into the first line of an existing
// wrapper without rebuilding the whole tree.
//
// Most callers should use [Multiline] instead.
func NewMultilineMessage(lines []string) *MultilineMessage {
	return &MultilineMessage{lines: lines}
}

// Lines returns the authored line list. Transport authors call this
// when rendering each line independently.
func (m *MultilineMessage) Lines() []string {
	return m.lines
}

// String joins the lines with "\n". Used by the fmt.Stringer fallback
// path in transports that don't special-case the type (JSON sinks and
// every wrapper transport).
func (m *MultilineMessage) String() string {
	return strings.Join(m.lines, "\n")
}

// MarshalJSON returns the "\n"-joined string as a JSON string. Provided
// so a wrapper that accidentally lands inside Fields or Metadata
// serializes as a string rather than {} (no exported fields). Terminal
// transports still sanitize metadata values to a single line in v1;
// this just prevents silent data loss in JSON sinks.
func (m *MultilineMessage) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.String())
}
