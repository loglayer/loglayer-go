// Package maputil provides value-conversion and deep-clone helpers used
// by LogLayer transports and plugins.
//
// Two primitives:
//
//   - [ToMap] normalizes any value to map[string]any via JSON roundtrip.
//     Lossy on type; suitable for transports that want a flat shape.
//   - [Cloner] deep-clones a value with replacement predicates applied at
//     any depth. Preserves the runtime type; suitable for redaction and
//     rewrite-style plugins.
//
// Both are safe to use from any goroutine.
package maputil

import "encoding/json"

// ToMap returns v as a map[string]any. Nil and existing maps pass through;
// other values (structs, pointers, ...) are converted via a JSON roundtrip
// so their exported, json-tagged fields land in the resulting map. Returns
// nil when v is nil or the roundtrip fails (channel, function, unsupported
// types).
func ToMap(v any) map[string]any {
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
