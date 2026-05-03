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

// Multiline wraps the supplied arguments as separate authored lines.
//
// This minimal form treats every argument as already-string-shaped.
// Later steps in this plan extend the constructor with non-string %v
// formatting, nested-wrapper flattening, and per-arg "\n" splitting.
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
