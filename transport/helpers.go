package transport

import (
	"fmt"
	"io"
	"os"
	"strings"

	"go.loglayer.dev"
	"go.loglayer.dev/utils/maputil"
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

// SanitizeMessage strips ASCII control characters from a string so
// that user-controlled input can't forge log lines (via CR / LF) or
// smuggle ANSI escape sequences (via ESC) into terminal renderers.
// Tabs are preserved since they're commonly used for column alignment.
//
// Used by console and pretty (the terminal renderers) before writing
// to their writer. Structured/JSON renderers don't need this because
// encoding/json escapes control characters automatically.
//
// Third-party text-renderer transports should call this on user message
// strings; structured-output transports don't need to.
func SanitizeMessage(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\t' || (r >= 0x20 && r != 0x7f) {
			return r
		}
		return -1
	}, s)
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

// MetadataAsMap extracts a map[string]any from any metadata value. Maps
// (loglayer.Metadata or raw map[string]any) pass through directly; other
// types are converted via a JSON roundtrip so their exported fields land
// at the root of the log object.
//
// Returns nil when the JSON roundtrip fails (cyclic structs, channels,
// functions, custom MarshalJSON returning a non-object). Callers that
// need to know about the failure should compare against nil; the helper
// itself doesn't surface an error to keep the dispatch path simple.
// Transports that want richer error reporting should call json.Marshal
// themselves and surface failures via OnError.
func MetadataAsMap(v any) map[string]any {
	if m, ok := MetadataAsRootMap(v); ok {
		return m
	}
	return maputil.ToMap(v)
}

// MergeFieldsAndMetadata combines params.Data (the assembled fields + error
// map) with the metadata value into a single map for transports that render
// to a flat structure. Returns nil when both inputs are empty so callers can
// short-circuit.
//
// Metadata policy: map metadata merges at the root; non-map metadata is
// JSON-roundtripped via MetadataAsMap and spread at the root (or dropped if
// the roundtrip fails). For "nest non-map metadata under a `metadata` key"
// semantics use MergeIntoMap instead.
//
// Pretty has its own local variant that uses a richer metadata extractor
// (preserves a _metadata fallback for slices/scalars). Other transports
// should use this one.
func MergeFieldsAndMetadata(p loglayer.TransportParams) map[string]any {
	if len(p.Data) == 0 && p.Metadata == nil {
		return nil
	}
	out := make(map[string]any, FieldEstimate(p))
	for k, v := range p.Data {
		out[k] = v
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

// MergeIntoMap copies data and metadata into dst (the same map; mutated in
// place) and returns dst for chaining convenience. Use it from encoders that
// have already seeded dst with their own protocol fields (level/time/msg,
// ddsource/...) and want to layer user data on top.
//
// Metadata policy: map metadata merges at the root; non-map metadata (struct,
// scalar, slice, ...) lands under the `metadata` key without a JSON roundtrip,
// so encoders downstream get the raw value. For "spread non-map metadata at
// root via JSON roundtrip" semantics use MergeFieldsAndMetadata instead.
//
// Pretty has its own variant with an `_metadata` fallback for non-roundtrippable
// values (see transports/pretty/render.go); it doesn't use this helper.
func MergeIntoMap(dst map[string]any, data map[string]any, metadata any) map[string]any {
	for k, v := range data {
		dst[k] = v
	}
	if m, ok := MetadataAsRootMap(metadata); ok {
		for k, v := range m {
			dst[k] = v
		}
	} else if metadata != nil {
		dst["metadata"] = metadata
	}
	return dst
}

// MetadataAsRootMap returns metadata as a map[string]any if it's a
// "flatten at root" shape (loglayer.Metadata or raw map[string]any),
// false otherwise. Use this in transports that need to decide whether
// metadata flattens to individual attributes/fields (true) or nests
// under MetadataFieldName (false). Metadata and map[string]any share an
// underlying type but are distinct named types, so a single type
// assertion handles only one of them; this helper covers both.
func MetadataAsRootMap(v any) (map[string]any, bool) {
	switch m := v.(type) {
	case loglayer.Metadata:
		return m, true
	case map[string]any:
		return m, true
	}
	return nil, false
}

// FieldEstimate returns the expected number of fields a transport will emit
// for the given params. Use it to size pre-allocated slices/maps in transports
// that benefit from capacity hints (zap, charmlog).
func FieldEstimate(p loglayer.TransportParams) int {
	n := len(p.Data)
	if m, ok := MetadataAsRootMap(p.Metadata); ok {
		n += len(m)
	} else if p.Metadata != nil {
		n++
	}
	return n
}
