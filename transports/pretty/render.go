package pretty

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"go.loglayer.dev/loglayer"
)

// combineData merges Data (fields + error) and Metadata into a single map for
// rendering. Map metadata merges at root; struct metadata is JSON-roundtripped
// into root fields.
func combineData(params loglayer.TransportParams) map[string]any {
	if !params.HasData && params.Metadata == nil {
		return nil
	}
	out := make(map[string]any)
	if params.HasData {
		for k, v := range params.Data {
			out[k] = v
		}
	}
	if params.Metadata != nil {
		for k, v := range metadataAsMap(params.Metadata) {
			out[k] = v
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// metadataAsMap is a pretty-local variant of transport.MetadataAsMap that
// preserves an `_metadata` fallback when JSON roundtripping fails, so unknown
// values still surface in pretty output instead of being dropped.
func metadataAsMap(v any) map[string]any {
	if v == nil {
		return nil
	}
	if m, ok := v.(map[string]any); ok {
		return m
	}
	b, err := json.Marshal(v)
	if err != nil {
		return map[string]any{"_metadata": fmt.Sprintf("%v", v)}
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return map[string]any{"_metadata": string(b)}
	}
	return m
}

// renderInline produces a compact "key=value, ..." representation of data,
// truncated past depth.
func (t *Transport) renderInline(data map[string]any, depth int) string {
	if len(data) == 0 {
		return ""
	}
	keys := sortedKeys(data)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s",
			t.theme.DataKey(k),
			t.theme.DataValue(t.formatValueInline(data[k], depth-1))))
	}
	return strings.Join(parts, " ")
}

func (t *Transport) formatValueInline(v any, depth int) string {
	if v == nil {
		return "null"
	}
	switch val := v.(type) {
	case string:
		return strconvQuoteIfNeeded(val)
	case bool, int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64, json.Number:
		return fmt.Sprintf("%v", val)
	case map[string]any:
		if depth <= 0 {
			return "{...}"
		}
		keys := sortedKeys(val)
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			parts = append(parts, fmt.Sprintf("%s=%s",
				k, t.formatValueInline(val[k], depth-1)))
		}
		return "{" + strings.Join(parts, " ") + "}"
	case []any:
		if depth <= 0 {
			return "[...]"
		}
		parts := make([]string, len(val))
		for i, item := range val {
			parts[i] = t.formatValueInline(item, depth-1)
		}
		return "[" + strings.Join(parts, " ") + "]"
	case error:
		return strconvQuoteIfNeeded(val.Error())
	default:
		b, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprintf("%v", val)
		}
		return string(b)
	}
}

// renderExpanded writes data as YAML-like indented lines beneath the header.
func (t *Transport) renderExpanded(w io.Writer, data map[string]any) {
	if len(data) == 0 {
		return
	}
	t.writeMap(w, data, 1)
}

func (t *Transport) writeMap(w io.Writer, m map[string]any, indent int) {
	prefix := strings.Repeat("  ", indent)
	for _, k := range sortedKeys(m) {
		v := m[k]
		switch val := v.(type) {
		case map[string]any:
			if len(val) == 0 {
				fmt.Fprintf(w, "%s%s: %s\n", prefix, t.theme.DataKey(k), t.theme.DataValue("{}"))
				continue
			}
			fmt.Fprintf(w, "%s%s:\n", prefix, t.theme.DataKey(k))
			t.writeMap(w, val, indent+1)
		case []any:
			if len(val) == 0 {
				fmt.Fprintf(w, "%s%s: %s\n", prefix, t.theme.DataKey(k), t.theme.DataValue("[]"))
				continue
			}
			fmt.Fprintf(w, "%s%s:\n", prefix, t.theme.DataKey(k))
			t.writeSlice(w, val, indent+1)
		default:
			fmt.Fprintf(w, "%s%s: %s\n", prefix, t.theme.DataKey(k),
				t.theme.DataValue(t.formatScalarExpanded(v)))
		}
	}
}

func (t *Transport) writeSlice(w io.Writer, s []any, indent int) {
	prefix := strings.Repeat("  ", indent)
	for _, item := range s {
		switch val := item.(type) {
		case map[string]any:
			fmt.Fprintf(w, "%s-\n", prefix)
			t.writeMap(w, val, indent+1)
		case []any:
			fmt.Fprintf(w, "%s-\n", prefix)
			t.writeSlice(w, val, indent+1)
		default:
			fmt.Fprintf(w, "%s- %s\n", prefix, t.theme.DataValue(t.formatScalarExpanded(item)))
		}
	}
}

func (t *Transport) formatScalarExpanded(v any) string {
	if v == nil {
		return "null"
	}
	switch val := v.(type) {
	case string:
		return val
	case bool, int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64, json.Number:
		return fmt.Sprintf("%v", val)
	case error:
		return val.Error()
	default:
		b, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprintf("%v", val)
		}
		return string(b)
	}
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// strconvQuoteIfNeeded wraps a string in quotes when it contains whitespace or
// equals signs that would make inline output ambiguous.
func strconvQuoteIfNeeded(s string) string {
	if s == "" {
		return `""`
	}
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '=' || r == '"' {
			return fmt.Sprintf("%q", s)
		}
	}
	return s
}
