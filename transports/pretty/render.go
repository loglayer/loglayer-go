package pretty

import (
	"fmt"
	"io"
	"maps"
	"sort"
	"strings"

	"github.com/goccy/go-json"

	"go.loglayer.dev"
)

// combineData merges Data (fields + error) and Metadata into a single map for
// rendering. Behavior depends on params.Schema.MetadataFieldName:
//   - empty (default): map metadata merges at root; struct metadata is
//     JSON-roundtripped into root fields; non-roundtrippable values land
//     under "_metadata".
//   - set: the entire metadata value nests under the key. Maps and
//     structs render as nested YAML; scalars render inline.
func combineData(params loglayer.TransportParams) map[string]any {
	if len(params.Data) == 0 && params.Metadata == nil {
		return nil
	}
	out := make(map[string]any)
	maps.Copy(out, params.Data)
	if params.Metadata != nil {
		if key := params.Schema.MetadataFieldName; key != "" {
			out[key] = metadataValueForKey(params.Metadata)
		} else {
			maps.Copy(out, metadataAsRootFields(params.Metadata))
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// metadataValueForKey returns a value suitable for nesting under a single
// key in the YAML output. Maps and structs JSON-roundtrip into a generic
// map[string]any (so writeMap renders nested keys and slices land as []any
// for YAML list rendering); scalars and non-roundtrippable values pass
// through so writeMap's scalar path renders them inline.
func metadataValueForKey(v any) any {
	if v == nil {
		return v
	}
	if b, err := json.Marshal(v); err == nil {
		var m map[string]any
		if json.Unmarshal(b, &m) == nil {
			return m
		}
	}
	return v
}

// metadataAsRootFields is a pretty-local variant of transport.MetadataAsMap
// that preserves an `_metadata` fallback when JSON roundtripping fails, so
// unknown values still surface in pretty output instead of being dropped.
func metadataAsRootFields(v any) map[string]any {
	if v == nil {
		return nil
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
	keys := sortedKeys(m)
	// Pad keys at this level to the longest key's width, but only count
	// keys whose values render on the same line (scalars and empty
	// containers). Map/slice values that recurse onto new lines don't
	// participate in alignment.
	maxKey := 0
	for _, k := range keys {
		if !sameLineValue(m[k]) {
			continue
		}
		if len(k) > maxKey {
			maxKey = len(k)
		}
	}
	for _, k := range keys {
		v := m[k]
		switch val := v.(type) {
		case map[string]any:
			if len(val) == 0 {
				t.writeAlignedScalar(w, prefix, k, maxKey, "{}")
				continue
			}
			fmt.Fprintf(w, "%s%s:\n", prefix, t.theme.DataKey(k))
			t.writeMap(w, val, indent+1)
		case []any:
			if len(val) == 0 {
				t.writeAlignedScalar(w, prefix, k, maxKey, "[]")
				continue
			}
			fmt.Fprintf(w, "%s%s:\n", prefix, t.theme.DataKey(k))
			t.writeSlice(w, val, indent+1)
		default:
			t.writeAlignedScalar(w, prefix, k, maxKey, t.formatScalarExpanded(v))
		}
	}
}

// writeAlignedScalar emits "<prefix><key>:<padding> <value>\n" so values
// line up with the longest sibling key at the same level. The padding
// goes between the colon and the value (after the theme styling has
// wrapped the key) so coloring stays localized to the key text.
func (t *Transport) writeAlignedScalar(w io.Writer, prefix, key string, maxKey int, value string) {
	pad := strings.Repeat(" ", maxKey-len(key))
	fmt.Fprintf(w, "%s%s:%s %s\n", prefix, t.theme.DataKey(key), pad, t.theme.DataValue(value))
}

// sameLineValue reports whether v renders on the same line as its key
// (scalars and empty containers do; non-empty maps and slices recurse).
func sameLineValue(v any) bool {
	switch val := v.(type) {
	case map[string]any:
		return len(val) == 0
	case []any:
		return len(val) == 0
	default:
		return true
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
