package transport

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"go.loglayer.dev/loglayer"
)

// WriterOrStderr returns w if non-nil, otherwise os.Stderr. Used by wrapper
// transports to default the construction-time writer.
func WriterOrStderr(w io.Writer) io.Writer {
	if w != nil {
		return w
	}
	return os.Stderr
}

// WriterOrStdout returns w if non-nil, otherwise os.Stdout. Used by renderer
// transports that target stdout by default.
func WriterOrStdout(w io.Writer) io.Writer {
	if w != nil {
		return w
	}
	return os.Stdout
}

// JoinMessages concatenates a slice of values into a single space-separated
// string. Strings are passed through; other types use fmt.Sprintf("%v", ...).
//
// The single-string case is the dominant log shape, so it returns the value
// directly without allocating a slice.
func JoinMessages(messages []any) string {
	switch len(messages) {
	case 0:
		return ""
	case 1:
		if s, ok := messages[0].(string); ok {
			return s
		}
		return fmt.Sprintf("%v", messages[0])
	}
	parts := make([]string, len(messages))
	for i, m := range messages {
		if s, ok := m.(string); ok {
			parts[i] = s
		} else {
			parts[i] = fmt.Sprintf("%v", m)
		}
	}
	return strings.Join(parts, " ")
}

// MetadataAsMap extracts a map[string]any from any metadata value. Maps pass
// through directly; other types are converted via a JSON roundtrip so their
// exported fields land at the root of the log object. Returns nil on failure.
func MetadataAsMap(v any) map[string]any {
	if v == nil {
		return nil
	}
	if m, ok := v.(map[string]any); ok {
		return m
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil
	}
	return m
}

// MergeFieldsAndMetadata combines params.Data (the assembled fields + error
// map) with the metadata value into a single map for transports that render
// to a flat structure. Map metadata merges at the root; struct metadata is
// JSON-roundtripped via MetadataAsMap. Returns nil when both inputs are empty
// so callers can short-circuit.
//
// Pretty has its own local variant that uses a richer metadata extractor
// (preserves a _metadata fallback for slices/scalars). Other transports
// should use this one.
func MergeFieldsAndMetadata(p loglayer.TransportParams) map[string]any {
	if !p.HasData && p.Metadata == nil {
		return nil
	}
	out := make(map[string]any, FieldEstimate(p))
	if p.HasData {
		for k, v := range p.Data {
			out[k] = v
		}
	}
	if p.Metadata != nil {
		for k, v := range MetadataAsMap(p.Metadata) {
			out[k] = v
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// FieldEstimate returns the expected number of fields a transport will emit
// for the given params. Use it to size pre-allocated slices/maps in transports
// that benefit from capacity hints (zap, charmlog).
func FieldEstimate(p loglayer.TransportParams) int {
	n := 0
	if p.HasData {
		n += len(p.Data)
	}
	if m, ok := p.Metadata.(map[string]any); ok {
		n += len(m)
	} else if p.Metadata != nil {
		n++
	}
	return n
}
