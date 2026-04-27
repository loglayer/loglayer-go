package maputil

import (
	"fmt"
	"reflect"
	"strings"
)

// Cloner deep-clones a value, replacing sensitive content with Censor.
// Designed for redaction-style plugins that walk arbitrary user values
// (maps, structs, slices, pointers) without forcing the type to change.
//
// Canonical use case: the plugins/redact plugin. Cloner exists in this
// public package so third-party plugin authors can build their own
// redactors with the same type-preservation semantics. If you only need
// flat-map mutation, prefer ToMap and modify the resulting map directly;
// Cloner's reflection walk is only worth its cost when the caller's
// runtime type matters.
//
// MatchKey is invoked with the JSON name (or struct field name when no
// json tag is present) for struct fields and with the literal key for
// string-keyed maps. MatchValue is invoked with every string value
// encountered, at any depth.
//
// The caller's input is never mutated. Containers (maps, slices, structs,
// pointers) are freshly allocated in the returned value.
type Cloner struct {
	MatchKey   func(string) bool
	MatchValue func(string) bool
	Censor     any
}

// maxCloneDepth caps Clone's recursion. Bounds memory and stack usage
// when a caller passes a cyclic or pathologically deep value. Real
// payloads almost never nest more than a handful of levels; 64 leaves
// generous headroom while still guaranteeing termination.
const maxCloneDepth = 64

// cloneState carries per-call recursion state through cloneValue and
// its helpers. visited is a path-local set of pointer addresses for
// cycle breaking: a pointer that's revisited mid-walk yields the
// element type's zero value instead of recursing further. depth is a
// fallback bound for non-pointer recursion paths (very deep nested
// maps or slices) and as defense-in-depth if visited misses something.
type cloneState struct {
	depth   int
	visited map[uintptr]struct{}
}

// Clone returns a deep clone of v with sensitive content replaced by
// Censor. Returns v unchanged when nothing in v is reachable for matching
// (basic scalars without a value match, nil, channels, functions).
//
// Safe against cyclic inputs: a self-referencing pointer chain breaks
// at the cycle (the second visit returns the element's zero value).
// Pathologically deep non-pointer nesting bottoms out at maxCloneDepth.
func (c *Cloner) Clone(v any) any {
	if v == nil {
		return nil
	}
	st := &cloneState{visited: make(map[uintptr]struct{})}
	out := c.cloneValue(reflect.ValueOf(v), st)
	if !out.IsValid() {
		return nil
	}
	return out.Interface()
}

func (c *Cloner) cloneValue(v reflect.Value, st *cloneState) reflect.Value {
	if !v.IsValid() {
		return v
	}
	if st.depth >= maxCloneDepth {
		return reflect.Zero(v.Type())
	}
	st.depth++
	defer func() { st.depth-- }()
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return v
		}
		// Cycle detection: if we've already entered this pointer
		// during the current walk, treat it as "seen" and stop.
		// Returning a zero pointer breaks the cycle without bloating
		// the output with N copies of the same subgraph.
		addr := v.Pointer()
		if _, seen := st.visited[addr]; seen {
			return reflect.Zero(v.Type())
		}
		st.visited[addr] = struct{}{}
		defer delete(st.visited, addr)
		cloned := c.cloneValue(v.Elem(), st)
		ptr := reflect.New(v.Type().Elem())
		if cloned.IsValid() {
			ptr.Elem().Set(cloned)
		}
		return ptr
	case reflect.Interface:
		if v.IsNil() {
			return v
		}
		return c.cloneValue(v.Elem(), st)
	case reflect.Struct:
		return c.cloneStruct(v, st)
	case reflect.Map:
		return c.cloneMap(v, st)
	case reflect.Slice:
		if v.IsNil() {
			return v
		}
		return c.cloneSlice(v, st)
	case reflect.Array:
		return c.cloneArray(v, st)
	case reflect.String:
		if c.MatchValue != nil && c.MatchValue(v.String()) {
			return c.censorAs(v.Type())
		}
		return v
	default:
		return v
	}
}

func (c *Cloner) cloneStruct(v reflect.Value, st *cloneState) reflect.Value {
	t := v.Type()
	out := reflect.New(t).Elem()
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if !sf.IsExported() {
			continue
		}
		field := v.Field(i)
		if c.MatchKey != nil && c.MatchKey(jsonName(sf)) {
			out.Field(i).Set(c.censorAs(sf.Type))
			continue
		}
		out.Field(i).Set(c.cloneValue(field, st))
	}
	return out
}

func (c *Cloner) cloneMap(v reflect.Value, st *cloneState) reflect.Value {
	if v.IsNil() {
		return v
	}
	out := reflect.MakeMapWithSize(v.Type(), v.Len())
	stringKeys := v.Type().Key().Kind() == reflect.String
	valueType := v.Type().Elem()
	iter := v.MapRange()
	for iter.Next() {
		k := iter.Key()
		val := iter.Value()
		if stringKeys && c.MatchKey != nil && c.MatchKey(k.String()) {
			out.SetMapIndex(k, c.censorAs(valueType))
			continue
		}
		cloned := c.cloneValue(val, st)
		if !cloned.IsValid() {
			out.SetMapIndex(k, reflect.Zero(valueType))
			continue
		}
		// SetMapIndex requires the value to match the map's value type.
		// When the map's value type is interface and the cloned value is
		// concrete, wrap it explicitly so reflection assigns it.
		if valueType.Kind() == reflect.Interface && cloned.Kind() != reflect.Interface {
			wrapped := reflect.New(valueType).Elem()
			wrapped.Set(cloned)
			cloned = wrapped
		}
		out.SetMapIndex(k, cloned)
	}
	return out
}

func (c *Cloner) cloneSlice(v reflect.Value, st *cloneState) reflect.Value {
	t := v.Type()
	out := reflect.MakeSlice(t, v.Len(), v.Len())
	elemType := t.Elem()
	for i := 0; i < v.Len(); i++ {
		cloned := c.cloneValue(v.Index(i), st)
		if elemType.Kind() == reflect.Interface && cloned.IsValid() && cloned.Kind() != reflect.Interface {
			wrapped := reflect.New(elemType).Elem()
			wrapped.Set(cloned)
			cloned = wrapped
		}
		if cloned.IsValid() {
			out.Index(i).Set(cloned)
		}
	}
	return out
}

func (c *Cloner) cloneArray(v reflect.Value, st *cloneState) reflect.Value {
	out := reflect.New(v.Type()).Elem()
	for i := 0; i < v.Len(); i++ {
		cloned := c.cloneValue(v.Index(i), st)
		if cloned.IsValid() {
			out.Index(i).Set(cloned)
		}
	}
	return out
}

// censorAs returns a reflect.Value of type t holding the censor. String
// targets get the censor stringified; interface targets get the censor as
// passed; other types fall back to the type's zero value (since we can't
// safely substitute a foreign-typed censor into, say, an int field).
func (c *Cloner) censorAs(t reflect.Type) reflect.Value {
	switch t.Kind() {
	case reflect.String:
		s := stringify(c.Censor)
		v := reflect.New(t).Elem()
		v.SetString(s)
		return v
	case reflect.Interface:
		if c.Censor == nil {
			return reflect.Zero(t)
		}
		cv := reflect.ValueOf(c.Censor)
		if cv.Type().AssignableTo(t) {
			wrapped := reflect.New(t).Elem()
			wrapped.Set(cv)
			return wrapped
		}
		return reflect.Zero(t)
	default:
		return reflect.Zero(t)
	}
}

func stringify(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

// jsonName returns the field's json tag name, falling back to the field
// name. Tags of "-" or empty fall back to the field name.
func jsonName(f reflect.StructField) string {
	tag, ok := f.Tag.Lookup("json")
	if !ok {
		return f.Name
	}
	if i := strings.IndexByte(tag, ','); i >= 0 {
		tag = tag[:i]
	}
	if tag == "" || tag == "-" {
		return f.Name
	}
	return tag
}
