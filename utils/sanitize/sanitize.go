// Package sanitize cleans user-controlled strings before they're written to
// a terminal or log line, so untrusted input can't forge log lines (CR / LF),
// smuggle ANSI escape sequences (ESC), spoof text direction (U+202E "Trojan
// Source" bidi overrides), or hide content (zero-width joiners and other
// Unicode formatting characters).
package sanitize

import (
	"strings"
	"unicode"
)

// Message drops non-printable runes from s. Tabs are preserved since they're
// commonly used for column alignment.
//
// "Non-printable" follows unicode.IsPrint, which excludes the C0/C1 control
// sets (0x00-0x1F, 0x7F-0x9F), the Cf "format" category (bidi controls, ZWJ,
// ZWNJ, BOM, etc.), and the rest of the Cc/Cn/Co/Cs categories. ASCII letters,
// digits, punctuation, symbols, accented characters, CJK, emoji, and the like
// all pass through.
//
// Used by console and pretty (the terminal renderers) and by integrations
// like loghttp before writing user-controlled values. Structured/JSON
// renderers don't need this because encoding/json escapes control characters
// automatically.
func Message(s string) string {
	// Fast path: scan for any rune that would be filtered. If none, return the
	// input unchanged so the common case (no injection attempt) doesn't
	// allocate; strings.Map always allocates, regardless of whether the
	// predicate ever returns -1.
	if !needsSanitization(s) {
		return s
	}
	return strings.Map(func(r rune) rune {
		if r == '\t' || unicode.IsPrint(r) {
			return r
		}
		return -1
	}, s)
}

// needsSanitization reports whether s contains any rune Message would drop.
//
// Two-stage: scan bytes first for the dominant pure-ASCII case (no UTF-8
// decode, no table lookup; ~5x faster than range-over-string). On the first
// non-ASCII byte we fall through to a Unicode-aware scan since IsPrint catches
// bidi controls, ZWJ, etc., that the byte path can't classify.
func needsSanitization(s string) bool {
	for i := 0; i < len(s); i++ {
		b := s[i]
		if b >= 0x80 {
			return needsSanitizationUnicode(s[i:])
		}
		if b != '\t' && (b < 0x20 || b == 0x7f) {
			return true
		}
	}
	return false
}

func needsSanitizationUnicode(s string) bool {
	for _, r := range s {
		if r != '\t' && !unicode.IsPrint(r) {
			return true
		}
	}
	return false
}
