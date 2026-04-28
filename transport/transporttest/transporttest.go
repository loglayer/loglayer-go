// Package transporttest provides helpers and a contract test suite for
// LogLayer transport implementations.
//
// Use [RunContract] to exercise the wrapper-transport contract (14 sub-tests
// covering message rendering, levels, metadata placement, fields, error
// rendering, level filtering, MetadataOnly / ErrorOnly / Raw, and WithCtx)
// against any transport that wraps a third-party logger and produces JSON
// output. The contract is what every built-in wrapper transport (zerolog,
// zap, slog, logrus, charmlog, phuslu) verifies, parameterised by per-wrapper
// rendering quirks (message key, level rendering, fatal handling).
//
// [ParseJSONLine] and [MessageContains] are general assertion helpers usable
// from any transport test.
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
