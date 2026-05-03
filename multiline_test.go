package loglayer_test

import (
	"reflect"
	"testing"

	"go.loglayer.dev/v2"
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
