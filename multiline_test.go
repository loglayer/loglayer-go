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
