package sanitize_test

import (
	"testing"

	"go.loglayer.dev/utils/sanitize"
)

// Message on a typical clean log string. The fast-path should short-circuit
// and return the input without allocating; this measures the per-rune scan
// cost for the dominant "no injection attempt" case.
func BenchmarkMessage_Clean(b *testing.B) {
	s := "user logged in successfully from 192.168.1.42"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sanitize.Message(s)
	}
}

// Message on a string with control chars. Forces strings.Map to allocate
// and rewrite the string. Measures the worst case.
func BenchmarkMessage_Dirty(b *testing.B) {
	s := "line1\r\nline2\x1b[31mred\x1b[0m end"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sanitize.Message(s)
	}
}

// Message on Unicode content (multi-byte runes, no controls). Verifies the
// IsPrint scan handles non-ASCII without unexpected cost.
func BenchmarkMessage_Unicode(b *testing.B) {
	s := "user café 🚀 中文 logged in"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sanitize.Message(s)
	}
}

func TestMessage(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain ascii", "hello world", "hello world"},
		{"crlf injection", "line1\r\nline2", "line1line2"},
		{"ansi esc", "red\x1b[31mtext", "red[31mtext"},
		{"keeps tab", "col1\tcol2", "col1\tcol2"},
		// "Trojan Source" CVE-2021-42574: U+202E flips text direction and can
		// make a viewer see code/log lines that don't match the underlying
		// bytes.
		{"trojan source bidi (U+202E)", "user\u202eevil", "userevil"},
		// Zero-width joiner can hide content boundaries (e.g. abc<ZWJ>def
		// renders as 'abcdef' but contains a separator).
		{"zero-width joiner (U+200D)", "abc\u200ddef", "abcdef"},
		// C1 control characters (0x80-0x9F) are non-ASCII control codes some
		// terminals interpret as ANSI commands.
		{"c1 control (U+0085)", "beforeafter", "beforeafter"},
		{"unicode passthrough", "café 🚀 中文", "café 🚀 中文"},
		{"already clean", "no-op for clean strings", "no-op for clean strings"},
		{"empty", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := sanitize.Message(c.in)
			if got != c.want {
				t.Errorf("Message(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}
