package loglayer

import (
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
