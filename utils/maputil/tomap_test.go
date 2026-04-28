package maputil

import (
	"reflect"
	"testing"
	"time"
)

type tagged struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type withSkip struct {
	Keep   string `json:"keep"`
	Hidden string `json:"-"`
}

type withOmit struct {
	Always string `json:"always"`
	Maybe  string `json:"maybe,omitempty"`
}

type untagged struct {
	A int
	B string
}

type nested struct {
	Outer string `json:"outer"`
	Inner tagged `json:"inner"`
}

type withTime struct {
	When time.Time `json:"when"`
}

type withSlice struct {
	Items []int `json:"items"`
}

type withStringMap struct {
	Tags map[string]string `json:"tags"`
}

type withPointer struct {
	Ptr *tagged `json:"ptr,omitempty"`
}

type embeddedBase struct {
	BaseField string `json:"base_field"`
}

type embedderInline struct {
	embeddedBase
	Own string `json:"own"`
}

type embedderTagged struct {
	embeddedBase `json:"base"`
	Own          string `json:"own"`
}

type unexportedField struct {
	Public  string `json:"public"`
	private string //nolint:unused
}

func TestToMap_NilAndMaps(t *testing.T) {
	if got := ToMap(nil); got != nil {
		t.Errorf("nil: got %v", got)
	}
	m := map[string]any{"a": 1}
	if got := ToMap(m); !reflect.DeepEqual(got, m) {
		t.Errorf("map passthrough: got %v", got)
	}
}

func TestToMap_TaggedStruct(t *testing.T) {
	got := ToMap(tagged{ID: 42, Name: "Alice", Email: "a@b"})
	want := map[string]any{"id": 42, "name": "Alice", "email": "a@b"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestToMap_PointerToStruct(t *testing.T) {
	v := &tagged{ID: 1, Name: "x", Email: "y"}
	got := ToMap(v)
	want := map[string]any{"id": 1, "name": "x", "email": "y"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestToMap_NilPointer(t *testing.T) {
	var v *tagged
	if got := ToMap(v); got != nil {
		t.Errorf("nil pointer: got %v", got)
	}
}

func TestToMap_SkipDash(t *testing.T) {
	got := ToMap(withSkip{Keep: "k", Hidden: "h"})
	want := map[string]any{"keep": "k"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestToMap_OmitEmpty(t *testing.T) {
	got := ToMap(withOmit{Always: "a"})
	want := map[string]any{"always": "a"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("zero-value omitted: got %v, want %v", got, want)
	}
	got = ToMap(withOmit{Always: "a", Maybe: "b"})
	want = map[string]any{"always": "a", "maybe": "b"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("populated: got %v, want %v", got, want)
	}
}

func TestToMap_UntaggedDefaults(t *testing.T) {
	got := ToMap(untagged{A: 1, B: "x"})
	want := map[string]any{"A": 1, "B": "x"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("untagged keeps Go field names: got %v, want %v", got, want)
	}
}

func TestToMap_NestedStruct(t *testing.T) {
	got := ToMap(nested{Outer: "o", Inner: tagged{ID: 7, Name: "n", Email: "e"}})
	want := map[string]any{
		"outer": "o",
		"inner": map[string]any{"id": 7, "name": "n", "email": "e"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestToMap_TimeMarshalerFallback(t *testing.T) {
	now := time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC)
	got := ToMap(withTime{When: now})
	if s, ok := got["when"].(string); !ok || s != "2026-04-27T12:00:00Z" {
		t.Errorf("time field should serialize via MarshalJSON, got %#v", got["when"])
	}
}

func TestToMap_TopLevelMarshaler(t *testing.T) {
	now := time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC)
	if got := ToMap(now); got != nil {
		t.Errorf("top-level Marshaler that doesn't yield an object should return nil, got %v", got)
	}
}

func TestToMap_Slice(t *testing.T) {
	got := ToMap(withSlice{Items: []int{1, 2, 3}})
	items, ok := got["items"].([]any)
	if !ok {
		t.Fatalf("items: got %#v", got["items"])
	}
	want := []any{1, 2, 3}
	if !reflect.DeepEqual(items, want) {
		t.Errorf("got %v, want %v", items, want)
	}
}

func TestToMap_StringKeyedMap(t *testing.T) {
	got := ToMap(withStringMap{Tags: map[string]string{"env": "prod"}})
	tags, ok := got["tags"].(map[string]any)
	if !ok {
		t.Fatalf("tags: got %#v", got["tags"])
	}
	if tags["env"] != "prod" {
		t.Errorf("got %v", tags)
	}
}

func TestToMap_NilFieldPointer(t *testing.T) {
	got := ToMap(withPointer{})
	want := map[string]any{"ptr": nil}
	if got["ptr"] != nil {
		t.Errorf("nil pointer omitempty: got %v, want %v", got, want)
	}
}

func TestToMap_EmbeddedStructInlines(t *testing.T) {
	got := ToMap(embedderInline{embeddedBase: embeddedBase{BaseField: "b"}, Own: "o"})
	want := map[string]any{"base_field": "b", "own": "o"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestToMap_EmbeddedStructWithTagNotInlined(t *testing.T) {
	got := ToMap(embedderTagged{embeddedBase: embeddedBase{BaseField: "b"}, Own: "o"})
	if _, ok := got["base"]; !ok {
		t.Errorf("embedded with json tag should not be inlined, got %v", got)
	}
}

func TestToMap_UnexportedSkipped(t *testing.T) {
	got := ToMap(unexportedField{Public: "p", private: "x"})
	want := map[string]any{"public": "p"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("unexported should be skipped: got %v", got)
	}
}

type withZeroTime struct {
	When time.Time `json:"when,omitempty"`
}

type withEmptyMap struct {
	Tags map[string]string `json:"tags,omitempty"`
}

type withEmptySlice struct {
	Items []int `json:"items,omitempty"`
}

// TestToMap_OmitEmptyMatchesEncodingJSON locks in the encoding/json contract
// for omitempty: zero-value structs (e.g. time.Time{}) are NOT considered
// empty (encoding/json keeps them); empty non-nil maps and slices ARE empty
// (encoding/json drops them).
func TestToMap_OmitEmptyMatchesEncodingJSON(t *testing.T) {
	got := ToMap(withZeroTime{})
	if _, ok := got["when"]; !ok {
		t.Errorf("zero time.Time with omitempty should be kept (encoding/json semantics): got %v", got)
	}
	got = ToMap(withEmptyMap{Tags: map[string]string{}})
	if _, ok := got["tags"]; ok {
		t.Errorf("empty non-nil map with omitempty should be dropped: got %v", got)
	}
	got = ToMap(withEmptySlice{Items: []int{}})
	if _, ok := got["items"]; ok {
		t.Errorf("empty non-nil slice with omitempty should be dropped: got %v", got)
	}
}

type embPtr struct {
	X int `json:"x"`
}

type wrapWithEmbPtr struct {
	*embPtr
	Other string `json:"other"`
}

// TestToMap_EmbeddedNilPointerOmitted ensures we don't leak the unexported
// anonymous-field type name as a key when the embedded pointer is nil.
func TestToMap_EmbeddedNilPointerOmitted(t *testing.T) {
	got := ToMap(wrapWithEmbPtr{Other: "o"})
	want := map[string]any{"other": "o"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("nil embedded pointer should be omitted (no Go-private type name leak): got %v, want %v", got, want)
	}
}

type cyclic struct {
	Name string   `json:"name"`
	Self *cyclic  `json:"self,omitempty"`
	Kids []cyclic `json:"kids,omitempty"`
}

// TestToMap_CycleTerminates guarantees a self-referencing struct doesn't
// infinite-loop the walker; the depth limit kicks in.
func TestToMap_CycleTerminates(t *testing.T) {
	a := &cyclic{Name: "a"}
	a.Self = a
	got := ToMap(a)
	if got["name"] != "a" {
		t.Errorf("cyclic struct should terminate with at least one level: got %v", got)
	}
}
