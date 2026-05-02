package maputil_test

import (
	"reflect"
	"strings"
	"testing"

	"go.loglayer.dev/v2/utils/maputil"
)

func keyIn(set ...string) func(string) bool {
	m := make(map[string]struct{}, len(set))
	for _, k := range set {
		m[k] = struct{}{}
	}
	return func(k string) bool {
		_, ok := m[k]
		return ok
	}
}

func TestCloner_StructPreservesType(t *testing.T) {
	type user struct {
		Name     string `json:"name"`
		Password string `json:"password"`
	}
	c := &maputil.Cloner{
		MatchKey: keyIn("password"),
		Censor:   "[REDACTED]",
	}
	in := user{Name: "alice", Password: "hunter2"}
	out := c.Clone(in)
	got, ok := out.(user)
	if !ok {
		t.Fatalf("clone should preserve struct type, got %T", out)
	}
	if got.Password != "[REDACTED]" {
		t.Errorf("Password not redacted: %q", got.Password)
	}
	if got.Name != "alice" {
		t.Errorf("Name should pass through: %q", got.Name)
	}
}

func TestCloner_StructHonorsJSONTag(t *testing.T) {
	type evt struct {
		APIKey string `json:"apiKey"`
	}
	c := &maputil.Cloner{MatchKey: keyIn("apiKey"), Censor: "X"}
	out := c.Clone(evt{APIKey: "sk_live"}).(evt)
	if out.APIKey != "X" {
		t.Errorf("json tag match should redact: %q", out.APIKey)
	}
}

func TestCloner_StructFallbackToFieldName(t *testing.T) {
	type evt struct {
		Token string
	}
	c := &maputil.Cloner{MatchKey: keyIn("Token"), Censor: "X"}
	out := c.Clone(evt{Token: "t"}).(evt)
	if out.Token != "X" {
		t.Errorf("field name match should redact: %q", out.Token)
	}
}

func TestCloner_NestedStruct(t *testing.T) {
	type creds struct {
		Password string `json:"password"`
	}
	type req struct {
		User  string `json:"user"`
		Creds creds  `json:"creds"`
	}
	c := &maputil.Cloner{MatchKey: keyIn("password"), Censor: "X"}
	out := c.Clone(req{User: "alice", Creds: creds{Password: "p"}}).(req)
	if out.Creds.Password != "X" {
		t.Errorf("nested redaction failed: %q", out.Creds.Password)
	}
}

func TestCloner_PointerToStruct(t *testing.T) {
	type user struct {
		Password string `json:"password"`
	}
	c := &maputil.Cloner{MatchKey: keyIn("password"), Censor: "X"}
	in := &user{Password: "p"}
	out := c.Clone(in).(*user)
	if out.Password != "X" {
		t.Errorf("pointer-deref redact failed: %q", out.Password)
	}
	if in.Password != "p" {
		t.Errorf("caller pointer mutated: %q", in.Password)
	}
}

func TestCloner_MapStringAny(t *testing.T) {
	c := &maputil.Cloner{MatchKey: keyIn("password"), Censor: "X"}
	in := map[string]any{
		"user":     "alice",
		"password": "p",
	}
	out := c.Clone(in).(map[string]any)
	if out["password"] != "X" {
		t.Errorf("map redact failed: %v", out["password"])
	}
	if in["password"] != "p" {
		t.Errorf("input map mutated: %v", in["password"])
	}
}

func TestCloner_NestedMap(t *testing.T) {
	c := &maputil.Cloner{MatchKey: keyIn("secret"), Censor: "X"}
	in := map[string]any{
		"user": map[string]any{"secret": "s", "name": "alice"},
	}
	out := c.Clone(in).(map[string]any)
	user := out["user"].(map[string]any)
	if user["secret"] != "X" {
		t.Errorf("nested map redact failed: %v", user["secret"])
	}
}

func TestCloner_SliceOfMaps(t *testing.T) {
	c := &maputil.Cloner{MatchKey: keyIn("password"), Censor: "X"}
	in := map[string]any{
		"users": []any{
			map[string]any{"password": "a"},
			map[string]any{"password": "b"},
		},
	}
	out := c.Clone(in).(map[string]any)
	users := out["users"].([]any)
	for i, u := range users {
		if u.(map[string]any)["password"] != "X" {
			t.Errorf("users[%d] not redacted: %v", i, u)
		}
	}
}

func TestCloner_TypedSliceOfStructs(t *testing.T) {
	type item struct {
		Token string `json:"token"`
	}
	c := &maputil.Cloner{MatchKey: keyIn("token"), Censor: "X"}
	in := []item{{Token: "a"}, {Token: "b"}}
	out := c.Clone(in).([]item)
	for i, it := range out {
		if it.Token != "X" {
			t.Errorf("[%d] not redacted: %q", i, it.Token)
		}
	}
}

func TestCloner_MapWithNonStringKeys(t *testing.T) {
	c := &maputil.Cloner{MatchKey: keyIn("password"), Censor: "X"}
	in := map[int]string{1: "password", 2: "other"}
	out := c.Clone(in).(map[int]string)
	if !reflect.DeepEqual(out, in) {
		t.Errorf("non-string-keyed map should pass through unchanged: %v", out)
	}
}

func TestCloner_ValuePatternString(t *testing.T) {
	c := &maputil.Cloner{
		MatchValue: func(s string) bool { return strings.HasPrefix(s, "sk_") },
		Censor:     "X",
	}
	in := map[string]any{"api": "sk_live", "user": "alice"}
	out := c.Clone(in).(map[string]any)
	if out["api"] != "X" {
		t.Errorf("value pattern not applied: %v", out["api"])
	}
	if out["user"] != "alice" {
		t.Errorf("non-matching pattern leaked: %v", out["user"])
	}
}

func TestCloner_ValuePatternInsideStruct(t *testing.T) {
	type item struct {
		Note string `json:"note"`
	}
	c := &maputil.Cloner{
		MatchValue: func(s string) bool { return s == "secret" },
		Censor:     "X",
	}
	out := c.Clone(item{Note: "secret"}).(item)
	if out.Note != "X" {
		t.Errorf("value pattern not applied to struct field: %q", out.Note)
	}
}

func TestCloner_NonStringFieldZeroedOnMatch(t *testing.T) {
	type evt struct {
		Count int `json:"count"`
	}
	c := &maputil.Cloner{MatchKey: keyIn("count"), Censor: "X"}
	out := c.Clone(evt{Count: 42}).(evt)
	if out.Count != 0 {
		t.Errorf("non-string field should be zeroed on match: %v", out.Count)
	}
}

func TestCloner_UnexportedFieldsSkipped(t *testing.T) {
	type evt struct {
		Public  string `json:"public"`
		private string //nolint:unused
	}
	c := &maputil.Cloner{MatchKey: keyIn("private"), Censor: "X"}
	out := c.Clone(evt{Public: "p", private: "x"}).(evt)
	if out.Public != "p" {
		t.Errorf("public field touched: %q", out.Public)
	}
}

func TestCloner_NilValuesPassThrough(t *testing.T) {
	c := &maputil.Cloner{MatchKey: keyIn("k"), Censor: "X"}
	if got := c.Clone(nil); got != nil {
		t.Errorf("nil should pass through: %v", got)
	}
	var nilMap map[string]any
	if got := c.Clone(nilMap); got != nil {
		// nil map's zero is nil; passes back as nil interface
		if m, ok := got.(map[string]any); !ok || m != nil {
			t.Errorf("nil map should pass through: %T %v", got, got)
		}
	}
}

func TestCloner_NoPredicatesIsDeepClone(t *testing.T) {
	c := &maputil.Cloner{}
	in := map[string]any{"a": map[string]any{"b": 1}}
	out := c.Clone(in).(map[string]any)
	out["a"].(map[string]any)["b"] = 99
	if in["a"].(map[string]any)["b"] != 1 {
		t.Errorf("clone should be independent of input: %v", in)
	}
}

func TestCloner_StructWithNestedMapIsIndependent(t *testing.T) {
	type evt struct {
		Tags map[string]string `json:"tags"`
	}
	c := &maputil.Cloner{}
	in := evt{Tags: map[string]string{"env": "prod"}}
	out := c.Clone(in).(evt)
	out.Tags["env"] = "mutated"
	if in.Tags["env"] != "prod" {
		t.Errorf("inner map of cloned struct should be independent: in.Tags=%v", in.Tags)
	}
}

func TestCloner_StructWithSliceIsIndependent(t *testing.T) {
	type evt struct {
		Items []string `json:"items"`
	}
	c := &maputil.Cloner{}
	in := evt{Items: []string{"a", "b"}}
	out := c.Clone(in).(evt)
	out.Items[0] = "X"
	if in.Items[0] != "a" {
		t.Errorf("inner slice of cloned struct should be independent: in.Items=%v", in.Items)
	}
}

// A self-referencing pointer must not stack-overflow Clone. The cycle
// breaks at the second visit and the caller gets a finite output.
func TestCloner_CyclicPointer_DoesNotStackOverflow(t *testing.T) {
	type node struct {
		Name string `json:"name"`
		Next *node  `json:"next,omitempty"`
	}
	a := &node{Name: "a"}
	a.Next = a // self-loop

	c := &maputil.Cloner{}
	out := c.Clone(a)
	if out == nil {
		t.Fatal("cyclic input should not produce nil output")
	}
	cloned, ok := out.(*node)
	if !ok || cloned == nil {
		t.Fatalf("expected *node, got %T", out)
	}
	if cloned.Name != "a" {
		t.Errorf("first level Name: got %q", cloned.Name)
	}
	// The cycle must terminate: walking Next should hit nil within a
	// bounded number of hops, not loop forever.
	cur := cloned
	for i := 0; i < 10 && cur != nil; i++ {
		cur = cur.Next
	}
	if cur != nil {
		t.Error("cyclic chain should terminate within a bounded depth, but Next is still non-nil after 10 hops")
	}
}

// Cycle via a map value. Same idea but exercises the cloneMap path.
func TestCloner_CyclicViaMapValue_DoesNotStackOverflow(t *testing.T) {
	type box struct {
		Name string `json:"name"`
		M    map[string]*box
	}
	root := &box{Name: "root", M: map[string]*box{}}
	root.M["self"] = root

	c := &maputil.Cloner{}
	out := c.Clone(root)
	if out == nil {
		t.Fatal("cyclic map should not produce nil output")
	}
	cloned, ok := out.(*box)
	if !ok || cloned == nil {
		t.Fatalf("expected *box, got %T", out)
	}
	if cloned.Name != "root" {
		t.Errorf("Name: got %q", cloned.Name)
	}
}

// Depth cap protects against pathologically deep but acyclic input.
// Clone must terminate without panicking on a 1000-level chain.
func TestCloner_DeeplyNested_HitsDepthCap(t *testing.T) {
	type chain struct {
		Inner *chain
	}
	root := &chain{}
	cur := root
	for i := 0; i < 1000; i++ {
		cur.Inner = &chain{}
		cur = cur.Inner
	}

	c := &maputil.Cloner{}
	// Must not panic; the depth cap kicks in.
	_ = c.Clone(root)
}

// Measures the steady-state per-Clone cost. Each Clone allocates a
// fresh visited map and walks the input. Compare against the
// pre-cycle-protection baseline if you want the delta from the security
// hardening.
func BenchmarkClone_TypicalStruct(b *testing.B) {
	type req struct {
		Method   string `json:"method"`
		Path     string `json:"path"`
		User     string `json:"user"`
		Password string `json:"password"`
		Headers  map[string]string
	}
	in := req{
		Method:   "POST",
		Path:     "/api/v1/users",
		User:     "alice",
		Password: "shouldnotappear",
		Headers:  map[string]string{"Content-Type": "application/json", "Accept": "*/*"},
	}
	c := &maputil.Cloner{
		MatchKey: func(k string) bool { return k == "password" },
		Censor:   "[REDACTED]",
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.Clone(in)
	}
}

// Maps are the dominant input shape for redact in practice (most users
// pass loglayer.Metadata{...}). Measures the map-walk path specifically.
func BenchmarkClone_TypicalMap(b *testing.B) {
	in := map[string]any{
		"requestId": "abc-123",
		"userId":    42,
		"path":      "/api/v1/users",
		"password":  "shouldnotappear",
	}
	c := &maputil.Cloner{
		MatchKey: func(k string) bool { return k == "password" },
		Censor:   "[REDACTED]",
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.Clone(in)
	}
}
