// Package maputil provides value-conversion and deep-clone helpers used
// by LogLayer transports and plugins.
//
// Two primitives:
//
//   - [ToMap] normalizes any value to map[string]any. Structs are walked
//     via reflection (respecting json tags); maps and pointers-to-structs
//     are unwrapped; values implementing json.Marshaler at the top level
//     fall back to a JSON roundtrip. Lossy on type, suitable for
//     transports that want a flat shape.
//   - [Cloner] deep-clones a value with replacement predicates applied at
//     any depth. Preserves the runtime type; suitable for redaction and
//     rewrite-style plugins.
//
// Both are safe to use from any goroutine.
package maputil

import (
	"reflect"
	"strings"

	"github.com/goccy/go-json"
)

var jsonMarshalerType = reflect.TypeOf((*json.Marshaler)(nil)).Elem()

// maxWalkDepth bounds recursion in walkStruct/fieldValue so a self-referencing
// struct (a.next = &a) can't infinite-loop. Real metadata shapes never come
// close.
const maxWalkDepth = 32

// ToMap returns v as a map[string]any. Nil and existing maps pass through;
// pointer values are dereferenced; structs are walked via reflection,
// honoring `json` tags (rename, omitempty, "-"). Top-level values with a
// custom MarshalJSON method, and types that don't reduce to an object
// (slices, primitives), fall back to a JSON roundtrip; that roundtrip
// returns nil if the JSON shape isn't an object.
func ToMap(v any) map[string]any {
	if v == nil {
		return nil
	}
	if m, ok := v.(map[string]any); ok {
		return m
	}
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	switch rv.Kind() {
	case reflect.Struct:
		if hasMarshalJSON(rv) {
			return jsonRoundtripMap(v)
		}
		return walkStruct(rv, 0)
	case reflect.Map:
		if rv.Type().Key().Kind() != reflect.String {
			return nil
		}
		return walkStringMap(rv, 0)
	default:
		return jsonRoundtripMap(v)
	}
}

func hasMarshalJSON(rv reflect.Value) bool {
	if rv.Type().Implements(jsonMarshalerType) {
		return true
	}
	if rv.CanAddr() && rv.Addr().Type().Implements(jsonMarshalerType) {
		return true
	}
	return false
}

func walkStruct(rv reflect.Value, depth int) map[string]any {
	if depth >= maxWalkDepth {
		return nil
	}
	rt := rv.Type()
	m := make(map[string]any, rt.NumField())
	walkStructInto(m, rv, rt, depth)
	return m
}

func walkStructInto(out map[string]any, rv reflect.Value, rt reflect.Type, depth int) {
	n := rt.NumField()
	for i := 0; i < n; i++ {
		sf := rt.Field(i)
		tag := sf.Tag.Get("json")
		// An anonymous (embedded) struct field with no json tag has its
		// exported fields inlined into the parent, matching encoding/json.
		// The embedded type's own name may be unexported (lowercase type
		// name) but its exported fields still surface, so this branch runs
		// before the sf.IsExported gate. When the embedded value is a nil
		// pointer or otherwise can't be inlined, the field is dropped (not
		// emitted under the unexported type name, which would leak Go-private
		// identifiers into log output).
		if sf.Anonymous && tag == "" {
			fv := rv.Field(i)
			for fv.Kind() == reflect.Pointer {
				if fv.IsNil() {
					fv = reflect.Value{}
					break
				}
				fv = fv.Elem()
			}
			if !fv.IsValid() || fv.Kind() != reflect.Struct {
				continue
			}
			if hasMarshalJSON(fv) {
				out[sf.Name] = jsonRoundtripValue(fv.Interface())
				continue
			}
			walkStructInto(out, fv, fv.Type(), depth+1)
			continue
		}
		if !sf.IsExported() && !sf.Anonymous {
			continue
		}
		name, omitempty, skip := parseJSONTag(tag, sf.Name)
		if skip {
			continue
		}
		fv := rv.Field(i)
		if omitempty && isEmptyValue(fv) {
			continue
		}
		out[name] = fieldValue(fv, depth+1)
	}
}

// isEmptyValue mirrors encoding/json's omitempty semantics: empty for zero-len
// containers and strings, zero numerics/booleans, and nil pointers/interfaces;
// never empty for structs (e.g. time.Time{} stays in the output).
func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64,
		reflect.Interface, reflect.Pointer:
		return v.IsZero()
	}
	return false
}

func parseJSONTag(tag, defaultName string) (name string, omitempty, skip bool) {
	if tag == "" {
		return defaultName, false, false
	}
	if tag == "-" {
		return "", false, true
	}
	idx := strings.IndexByte(tag, ',')
	if idx == -1 {
		return tag, false, false
	}
	name = tag[:idx]
	if name == "" {
		name = defaultName
	}
	for _, opt := range strings.Split(tag[idx+1:], ",") {
		if opt == "omitempty" {
			omitempty = true
		}
	}
	return name, omitempty, false
}

func fieldValue(rv reflect.Value, depth int) any {
	if depth >= maxWalkDepth {
		return nil
	}
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	if hasMarshalJSON(rv) {
		return jsonRoundtripValue(rv.Interface())
	}
	switch rv.Kind() {
	case reflect.Struct:
		return walkStruct(rv, depth+1)
	case reflect.Map:
		if rv.Type().Key().Kind() != reflect.String {
			return jsonRoundtripValue(rv.Interface())
		}
		return walkStringMap(rv, depth+1)
	case reflect.Slice, reflect.Array:
		// []byte is base64-encoded by encoding/json; defer to the roundtrip.
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			return jsonRoundtripValue(rv.Interface())
		}
		n := rv.Len()
		s := make([]any, n)
		for i := 0; i < n; i++ {
			s[i] = fieldValue(rv.Index(i), depth+1)
		}
		return s
	case reflect.Interface:
		if rv.IsNil() {
			return nil
		}
		return fieldValue(rv.Elem(), depth+1)
	default:
		return rv.Interface()
	}
}

func walkStringMap(rv reflect.Value, depth int) map[string]any {
	if depth >= maxWalkDepth {
		return nil
	}
	m := make(map[string]any, rv.Len())
	iter := rv.MapRange()
	for iter.Next() {
		m[iter.Key().String()] = fieldValue(iter.Value(), depth+1)
	}
	return m
}

func jsonRoundtripMap(v any) map[string]any {
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

func jsonRoundtripValue(v any) any {
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil
	}
	return out
}
