package loglayer_test

import (
	"encoding/json"
	"reflect"
	"testing"

	loglayer "go.loglayer.dev/v2"
)

func TestMultiline_LinesReturnsAuthoredSlice(t *testing.T) {
	m := loglayer.Multiline("a", "b", "c")
	got := m.Lines()
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Lines() = %#v, want %#v", got, want)
	}
}

func TestMultiline_StringJoinsWithNewline(t *testing.T) {
	m := loglayer.Multiline("a", "b", "c")
	if got := m.String(); got != "a\nb\nc" {
		t.Errorf("String() = %q, want %q", got, "a\nb\nc")
	}
}

func TestMultiline_StringEmpty(t *testing.T) {
	m := loglayer.Multiline()
	if got := m.String(); got != "" {
		t.Errorf("String() on zero-arg = %q, want empty", got)
	}
}

func TestMultiline_StringSingle(t *testing.T) {
	m := loglayer.Multiline("only")
	if got := m.String(); got != "only" {
		t.Errorf("String() on single-arg = %q, want %q", got, "only")
	}
}

func TestMultiline_NonStringArgsFormatViaPercentV(t *testing.T) {
	m := loglayer.Multiline(42, true, nil)
	got := m.Lines()
	want := []string{"42", "true", "<nil>"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Lines() = %#v, want %#v", got, want)
	}
}

type stringerOnly struct{ v string }

func (s stringerOnly) String() string { return "stringer:" + s.v }

func TestMultiline_StringerArgsCallStringMethod(t *testing.T) {
	m := loglayer.Multiline(stringerOnly{v: "x"})
	if got := m.Lines(); !reflect.DeepEqual(got, []string{"stringer:x"}) {
		t.Errorf("Lines() = %#v, want [stringer:x]", got)
	}
}

func TestMultiline_NestedFlattens(t *testing.T) {
	inner := loglayer.Multiline("a", "b")
	outer := loglayer.Multiline(inner, "c")
	got := outer.Lines()
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("nested Lines() = %#v, want %#v", got, want)
	}
	if s := outer.String(); s != "a\nb\nc" {
		t.Errorf("nested String() = %q, want %q", s, "a\nb\nc")
	}
}

func TestMultiline_NestedDeep(t *testing.T) {
	inner := loglayer.Multiline("x")
	mid := loglayer.Multiline(inner, "y")
	outer := loglayer.Multiline(mid, "z")
	got := outer.Lines()
	want := []string{"x", "y", "z"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("deep-nested Lines() = %#v, want %#v", got, want)
	}
}

func TestMultiline_SplitsEmbeddedNewline(t *testing.T) {
	m := loglayer.Multiline("a\nb")
	got := m.Lines()
	want := []string{"a", "b"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Lines() = %#v, want %#v", got, want)
	}
	if s := m.String(); s != "a\nb" {
		t.Errorf("String() = %q, want %q", s, "a\nb")
	}
}

func TestMultiline_SplitMixedWithLiteralArgs(t *testing.T) {
	m := loglayer.Multiline("a\nb", "c")
	got := m.Lines()
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Lines() = %#v, want %#v", got, want)
	}
}

func TestMultiline_SplitCRLFKeepsCRForSanitizerToStrip(t *testing.T) {
	m := loglayer.Multiline("a\r\nb")
	got := m.Lines()
	want := []string{"a\r", "b"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Lines() = %#v, want %#v", got, want)
	}
}

func TestMultiline_SplitAppliesAfterStringerFormatting(t *testing.T) {
	m := loglayer.Multiline(stringerOnly{v: "x\ny"})
	got := m.Lines()
	want := []string{"stringer:x", "y"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Lines() = %#v, want %#v", got, want)
	}
}

func TestMultiline_MarshalJSON(t *testing.T) {
	m := loglayer.Multiline("a", "b", "c")
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal err: %v", err)
	}
	want := `"a\nb\nc"`
	if string(b) != want {
		t.Errorf("Marshal = %s, want %s", b, want)
	}
}

func TestMultiline_MarshalJSON_EmptyLines(t *testing.T) {
	m := loglayer.Multiline()
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal err: %v", err)
	}
	if string(b) != `""` {
		t.Errorf("Marshal = %s, want \"\"", b)
	}
}

func TestMultiline_MarshalJSON_RoundtripFromMetadata(t *testing.T) {
	wrapped := map[string]any{"detail": loglayer.Multiline("x", "y")}
	b, err := json.Marshal(wrapped)
	if err != nil {
		t.Fatalf("Marshal err: %v", err)
	}
	want := `{"detail":"x\ny"}`
	if string(b) != want {
		t.Errorf("Marshal = %s, want %s", b, want)
	}
}

func TestMultiline_DoesNotImplementError(t *testing.T) {
	// The wrapper is a message-content sentinel. Implementing error
	// would force a rendering policy that fits neither role. This
	// test pins that decision so a future "convenience" PR doesn't
	// accidentally add it.
	var v any = loglayer.Multiline("a", "b")
	if _, ok := v.(error); ok {
		t.Fatal("*MultilineMessage must not implement error")
	}
}

func TestMultiline_EdgeCases(t *testing.T) {
	cases := []struct {
		name    string
		args    []any
		want    []string
		wantStr string
	}{
		{"single empty arg", []any{""}, []string{""}, ""},
		{"trailing empty", []any{"a", ""}, []string{"a", ""}, "a\n"},
		{"interleaved nil", []any{"a", nil, "b"}, []string{"a", "<nil>", "b"}, "a\n<nil>\nb"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := loglayer.Multiline(tc.args...)
			if got := m.Lines(); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Lines() = %#v, want %#v", got, tc.want)
			}
			if got := m.String(); got != tc.wantStr {
				t.Errorf("String() = %q, want %q", got, tc.wantStr)
			}
		})
	}
}
