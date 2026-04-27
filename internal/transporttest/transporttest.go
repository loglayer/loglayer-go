// Package transporttest provides helpers for transport tests in this module.
// Internal: not part of the public API. Use only from go.loglayer.dev
// and its sub-packages.
package transporttest

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// ParseJSONLine parses the trimmed contents of buf as a single JSON object.
// Fails the test if the contents are not valid JSON.
func ParseJSONLine(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	line := strings.TrimSpace(buf.String())
	var obj map[string]any
	if err := json.Unmarshal([]byte(line), &obj); err != nil {
		t.Fatalf("output is not valid JSON: %v: got %q", err, line)
	}
	return obj
}

// tryParseJSONLine is the non-fatal counterpart of ParseJSONLine: it returns
// (nil, err) instead of failing the test when the buffer doesn't contain
// strict JSON. Used by contract tests that fall back to substring assertions
// for wrappers (charmlog) whose JSON formatter occasionally renders nested
// values as quoted strings rather than nested objects.
func tryParseJSONLine(buf *bytes.Buffer) (map[string]any, error) {
	line := strings.TrimSpace(buf.String())
	var obj map[string]any
	if err := json.Unmarshal([]byte(line), &obj); err != nil {
		return nil, err
	}
	return obj, nil
}

// MessageContains reports whether messages includes a string equal to want.
// Used by tests that assert on the loglayer.LogLine.Messages slice.
func MessageContains(messages []any, want string) bool {
	for _, m := range messages {
		if s, ok := m.(string); ok && s == want {
			return true
		}
	}
	return false
}
