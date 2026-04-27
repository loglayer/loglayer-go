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

// Clone returns a deep clone of v with sensitive content replaced by
// Censor. Returns v unchanged when nothing in v is reachable for matching
// (basic scalars without a value match, nil, channels, functions).
func (c *Cloner) Clone(v any) any {
	if v == nil {
		return nil
	}
	out := c.cloneValue(reflect.ValueOf(v))
	if !out.IsValid() {
		return nil
	}
	return out.Interface()
}

func (c *Cloner) cloneValue(v reflect.Value) reflect.Value {
	if !v.IsValid() {
		return v
	}
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return v
		}
		cloned := c.cloneValue(v.Elem())
		ptr := reflect.New(v.Type().Elem())
		ptr.Elem().Set(cloned)
		return ptr
	case reflect.Interface:
		if v.IsNil() {
			return v
		}
		return c.cloneValue(v.Elem())
	case reflect.Struct:
		return c.cloneStruct(v)
	case reflect.Map:
		return c.cloneMap(v)
	case reflect.Slice:
		if v.IsNil() {
			return v
		}
		return c.cloneSlice(v)
	case reflect.Array:
		return c.cloneArray(v)
	case reflect.String:
		if c.MatchValue != nil && c.MatchValue(v.String()) {
			return c.censorAs(v.Type())
		}
		return v
	default:
		return v
	}
}

func (c *Cloner) cloneStruct(v reflect.Value) reflect.Value {
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
		out.Field(i).Set(c.cloneValue(field))
	}
	return out
}

func (c *Cloner) cloneMap(v reflect.Value) reflect.Value {
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
		cloned := c.cloneValue(val)
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

func (c *Cloner) cloneSlice(v reflect.Value) reflect.Value {
	t := v.Type()
	out := reflect.MakeSlice(t, v.Len(), v.Len())
	elemType := t.Elem()
	for i := 0; i < v.Len(); i++ {
		cloned := c.cloneValue(v.Index(i))
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

func (c *Cloner) cloneArray(v reflect.Value) reflect.Value {
	out := reflect.New(v.Type()).Elem()
	for i := 0; i < v.Len(); i++ {
		cloned := c.cloneValue(v.Index(i))
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
