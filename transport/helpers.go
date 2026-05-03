package transport

import (
	"fmt"
	"io"
	"maps"
	"os"
	"strings"

	"go.loglayer.dev/v2"
	"go.loglayer.dev/v2/utils/maputil"
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

// JoinPrefixAndMessages folds prefix into the first message so the
// rendered output reads as one blob (`"[prefix] message body"`).
// Returns messages unchanged when prefix is empty or messages[0] is
// not a string; otherwise returns a fresh slice whose first element
// is `prefix + " " + messages[0]`.
//
// Most renderer / wrapper transports want this rendering and call
// the helper at the top of SendToLogger:
//
//	func (t *Transport) SendToLogger(p loglayer.TransportParams) {
//	    if !t.ShouldProcess(p.LogLevel) { return }
//	    p.Messages = transport.JoinPrefixAndMessages(p.Prefix, p.Messages)
//	    // ... existing rendering logic ...
//	}
//
// Transports that want to render the prefix differently (cli's
// dim-color treatment, a structured transport's separate JSON
// field, wrapper transports forwarding to the underlying logger's
// structured-field API) should NOT call this helper; instead
// consume p.Prefix directly and emit messages without the prefix
// folded in.
func JoinPrefixAndMessages(prefix string, messages []any) []any {
	if prefix == "" || len(messages) == 0 {
		return messages
	}
	out := make([]any, len(messages))
	copy(out, messages)
	switch v := messages[0].(type) {
	case *loglayer.MultilineMessage:
		lines := v.Lines()
		if len(lines) == 0 {
			out[0] = loglayer.Multiline(prefix)
			break
		}
		rebuilt := make([]any, len(lines))
		rebuilt[0] = prefix + " " + lines[0]
		for i, l := range lines[1:] {
			rebuilt[i+1] = l
		}
		out[0] = loglayer.Multiline(rebuilt...)
	case string:
		out[0] = prefix + " " + v
	default:
		out[0] = prefix + " " + fmt.Sprintf("%v", v)
	}
	return out
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
// Metadata policy is driven by [loglayer.Schema.MetadataFieldName] on the
// params:
//   - Empty (default): map metadata merges at the root; non-map metadata is
//     JSON-roundtripped via MetadataAsMap and spread at the root (or dropped
//     if the roundtrip fails).
//   - Non-empty: the entire metadata value (map or non-map) nests under the
//     configured key. No roundtrip; encoders downstream get the raw value.
//
// For "nest non-map metadata under a fixed key" semantics in encoders without
// access to TransportParams, use MergeIntoMap with an explicit metadataKey.
//
// Pretty has its own local variant that uses a richer metadata extractor
// (preserves a _metadata fallback for slices/scalars). Other transports
// should use this one.
func MergeFieldsAndMetadata(p loglayer.TransportParams) map[string]any {
	if len(p.Data) == 0 && p.Metadata == nil {
		return nil
	}
	out := make(map[string]any, FieldEstimate(p))
	maps.Copy(out, p.Data)
	if p.Metadata != nil {
		if key := p.Schema.MetadataFieldName; key != "" {
			out[key] = p.Metadata
		} else {
			maps.Copy(out, MetadataAsMap(p.Metadata))
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
// metadataKey controls placement: when non-empty, the entire metadata
// value nests under that key uniformly. When empty, map metadata merges
// at root and non-map nests under the literal "metadata" key without
// roundtrip. Callers with access to [loglayer.TransportParams] should
// pass params.Schema.MetadataFieldName.
//
// Pretty has its own variant with an `_metadata` fallback for non-roundtrippable
// values (see transports/pretty/render.go); it doesn't use this helper.
func MergeIntoMap(dst map[string]any, data map[string]any, metadata any, metadataKey string) map[string]any {
	maps.Copy(dst, data)
	if metadata == nil {
		return dst
	}
	if metadataKey != "" {
		dst[metadataKey] = metadata
		return dst
	}
	if m, ok := MetadataAsRootMap(metadata); ok {
		maps.Copy(dst, m)
		return dst
	}
	dst["metadata"] = metadata
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

// AssembleMessage flattens a message slice into a single string,
// applying sanitize to every authored chunk while preserving line
// boundaries inside *MultilineMessage values.
//
// For each element in messages:
//   - string s              -> sanitize(s)
//   - *MultilineMessage m   -> per-line sanitize, joined with "\n"
//   - any other v           -> sanitize(fmt.Sprintf("%v", v))
//
// Adjacent elements are joined with " ". Empty messages produce "".
// Nil elements format as "<nil>" via the default branch (matching
// JoinMessages's behavior for non-string elements).
//
// Used by terminal-style transports (cli, pretty, console). Wrapper
// transports and JSON sinks call JoinMessages instead; the
// *MultilineMessage.String method handles flattening transparently
// for them.
func AssembleMessage(messages []any, sanitize func(string) string) string {
	if len(messages) == 0 {
		return ""
	}
	parts := make([]string, len(messages))
	for i, m := range messages {
		parts[i] = assembleElement(m, sanitize)
	}
	return strings.Join(parts, " ")
}

func assembleElement(v any, sanitize func(string) string) string {
	switch x := v.(type) {
	case *loglayer.MultilineMessage:
		lines := x.Lines()
		out := make([]string, len(lines))
		for i, l := range lines {
			out[i] = sanitize(l)
		}
		return strings.Join(out, "\n")
	case string:
		return sanitize(x)
	default:
		return sanitize(fmt.Sprintf("%v", v))
	}
}
