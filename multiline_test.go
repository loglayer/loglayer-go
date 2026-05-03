package loglayer_test

import (
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
