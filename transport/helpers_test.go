package transport_test

import (
	"bytes"
	"os"
	"reflect"
	"testing"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
)

func TestWriterOrStderr(t *testing.T) {
	if got := transport.WriterOrStderr(nil); got != os.Stderr {
		t.Errorf("nil case: got %v, want os.Stderr", got)
	}
	var buf bytes.Buffer
	if got := transport.WriterOrStderr(&buf); got != &buf {
		t.Errorf("non-nil case: got %v, want &buf", got)
	}
}

func TestWriterOrStdout(t *testing.T) {
	if got := transport.WriterOrStdout(nil); got != os.Stdout {
		t.Errorf("nil case: got %v, want os.Stdout", got)
	}
	var buf bytes.Buffer
	if got := transport.WriterOrStdout(&buf); got != &buf {
		t.Errorf("non-nil case: got %v, want &buf", got)
	}
}

func TestJoinMessages(t *testing.T) {
	cases := []struct {
		name string
		in   []any
		want string
	}{
		{"empty", nil, ""},
		{"single string fast path", []any{"hello"}, "hello"},
		{"single non-string", []any{42}, "42"},
		{"single error", []any{benchErr("boom")}, "boom"},
		{"two strings", []any{"hello", "world"}, "hello world"},
		{"mixed types", []any{"count:", 42, true}, "count: 42 true"},
		{"all non-strings", []any{1, 2, 3}, "1 2 3"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := transport.JoinMessages(c.in); got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

type metaUser struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func TestMetadataAsMap(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		if got := transport.MetadataAsMap(nil); got != nil {
			t.Errorf("nil input: got %v, want nil", got)
		}
	})

	t.Run("map passes through unchanged", func(t *testing.T) {
		in := map[string]any{"k": "v", "n": 42}
		got := transport.MetadataAsMap(in)
		if !reflect.DeepEqual(got, in) {
			t.Errorf("got %v, want %v", got, in)
		}
	})

	t.Run("loglayer.Metadata alias passes through", func(t *testing.T) {
		in := loglayer.Metadata{"k": "v"}
		got := transport.MetadataAsMap(in)
		if got["k"] != "v" {
			t.Errorf("alias not preserved: got %v", got)
		}
	})

	t.Run("struct round-trips via JSON", func(t *testing.T) {
		got := transport.MetadataAsMap(metaUser{ID: 7, Name: "Alice"})
		if got["id"] != float64(7) || got["name"] != "Alice" {
			t.Errorf("struct fields not extracted: got %v", got)
		}
	})

	t.Run("unmarshalable returns nil", func(t *testing.T) {
		// A channel cannot be JSON-marshaled.
		ch := make(chan int)
		if got := transport.MetadataAsMap(ch); got != nil {
			t.Errorf("channel: got %v, want nil", got)
		}
	})

	t.Run("scalar that marshals but doesn't unmarshal to map returns nil", func(t *testing.T) {
		// A string marshals to `"hello"`, which unmarshals to string not map.
		if got := transport.MetadataAsMap("hello"); got != nil {
			t.Errorf("string: got %v, want nil", got)
		}
		if got := transport.MetadataAsMap(42); got != nil {
			t.Errorf("int: got %v, want nil", got)
		}
	})

	t.Run("slice returns nil (not a key/value shape)", func(t *testing.T) {
		if got := transport.MetadataAsMap([]int{1, 2, 3}); got != nil {
			t.Errorf("slice: got %v, want nil", got)
		}
	})
}

func TestFieldEstimate(t *testing.T) {
	cases := []struct {
		name string
		p    loglayer.TransportParams
		want int
	}{
		{
			"empty",
			loglayer.TransportParams{},
			0,
		},
		{
			"data only",
			loglayer.TransportParams{
				Data: loglayer.Data{"a": 1, "b": 2, "c": 3},
			},
			3,
		},
		{
			"map metadata only",
			loglayer.TransportParams{
				Metadata: map[string]any{"k1": "v", "k2": "v"},
			},
			2,
		},
		{
			"loglayer.Metadata alias counts as map",
			loglayer.TransportParams{
				Metadata: loglayer.Metadata{"k1": "v", "k2": "v"},
			},
			2,
		},
		{
			"struct metadata counts as 1",
			loglayer.TransportParams{
				Metadata: metaUser{ID: 7, Name: "Alice"},
			},
			1,
		},
		{
			"scalar metadata counts as 1",
			loglayer.TransportParams{
				Metadata: "hello",
			},
			1,
		},
		{
			"data plus map metadata sums",
			loglayer.TransportParams{
				Data:     loglayer.Data{"a": 1, "b": 2},
				Metadata: map[string]any{"c": 3, "d": 4},
			},
			4,
		},
		{
			"data plus struct metadata sums",
			loglayer.TransportParams{
				Data:     loglayer.Data{"a": 1, "b": 2},
				Metadata: metaUser{ID: 7, Name: "Alice"},
			},
			3,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := transport.FieldEstimate(c.p); got != c.want {
				t.Errorf("got %d, want %d", got, c.want)
			}
		})
	}
}

func TestMergeFieldsAndMetadata(t *testing.T) {
	cases := []struct {
		name string
		p    loglayer.TransportParams
		want map[string]any
	}{
		{
			name: "empty",
			p:    loglayer.TransportParams{},
			want: nil,
		},
		{
			name: "data only",
			p: loglayer.TransportParams{
				Data: loglayer.Data{"a": 1, "b": "two"},
			},
			want: map[string]any{"a": 1, "b": "two"},
		},
		{
			name: "metadata map only",
			p: loglayer.TransportParams{
				Metadata: map[string]any{"k": "v"},
			},
			want: map[string]any{"k": "v"},
		},
		{
			name: "data and metadata merge",
			p: loglayer.TransportParams{
				Data:     loglayer.Data{"a": 1},
				Metadata: map[string]any{"b": 2},
			},
			want: map[string]any{"a": 1, "b": 2},
		},
		{
			name: "metadata overrides data on key conflict",
			p: loglayer.TransportParams{
				Data:     loglayer.Data{"k": "from-data"},
				Metadata: map[string]any{"k": "from-metadata"},
			},
			want: map[string]any{"k": "from-metadata"},
		},
		{
			name: "metadata that fails to roundtrip is dropped",
			p: loglayer.TransportParams{
				Metadata: []int{1, 2, 3}, // not a map[string]any after roundtrip
			},
			want: nil,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := transport.MergeFieldsAndMetadata(c.p)
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

func TestMergeIntoMap(t *testing.T) {
	cases := []struct {
		name     string
		dst      map[string]any
		data     map[string]any
		metadata any
		want     map[string]any
	}{
		{
			name: "all empty preserves dst",
			dst:  map[string]any{"x": 1},
			want: map[string]any{"x": 1},
		},
		{
			name: "data merges into dst",
			dst:  map[string]any{"x": 1},
			data: map[string]any{"y": 2},
			want: map[string]any{"x": 1, "y": 2},
		},
		{
			name:     "map metadata merges at root",
			dst:      map[string]any{"x": 1},
			metadata: map[string]any{"y": 2},
			want:     map[string]any{"x": 1, "y": 2},
		},
		{
			name:     "non-map metadata nests under metadata key",
			dst:      map[string]any{"x": 1},
			metadata: []int{1, 2, 3},
			want:     map[string]any{"x": 1, "metadata": []int{1, 2, 3}},
		},
		{
			name:     "metadata overrides data on key conflict",
			dst:      map[string]any{},
			data:     map[string]any{"k": "from-data"},
			metadata: map[string]any{"k": "from-metadata"},
			want:     map[string]any{"k": "from-metadata"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := transport.MergeIntoMap(c.dst, c.data, c.metadata)
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

type benchErr string

func (e benchErr) Error() string { return string(e) }

// SanitizeMessage on a typical clean log string. The fast-path should
// short-circuit and return the input without allocating; this measures
// the per-rune scan cost for the dominant "no injection attempt" case.
func BenchmarkSanitizeMessage_Clean(b *testing.B) {
	s := "user logged in successfully from 192.168.1.42"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = transport.SanitizeMessage(s)
	}
}

// SanitizeMessage on a string with control chars. Forces strings.Map
// to allocate and rewrite the string. Measures the worst case.
func BenchmarkSanitizeMessage_Dirty(b *testing.B) {
	s := "line1\r\nline2\x1b[31mred\x1b[0m end"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = transport.SanitizeMessage(s)
	}
}

// SanitizeMessage on Unicode content (multi-byte runes, no controls).
// Verifies the IsPrint scan handles non-ASCII without unexpected cost.
func BenchmarkSanitizeMessage_Unicode(b *testing.B) {
	s := "user café 🚀 中文 logged in"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = transport.SanitizeMessage(s)
	}
}

func TestSanitizeMessage(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain ascii", "hello world", "hello world"},
		{"crlf injection", "line1\r\nline2", "line1line2"},
		{"ansi esc", "red\x1b[31mtext", "red[31mtext"},
		{"keeps tab", "col1\tcol2", "col1\tcol2"},
		// "Trojan Source" CVE-2021-42574: U+202E flips text direction
		// and can make a viewer see code/log lines that don't match the
		// underlying bytes. Strip it.
		{"trojan source bidi (U+202E)", "user‮evil", "userevil"},
		// Zero-width joiner can hide content boundaries (e.g. abc<ZWJ>def
		// renders as 'abcdef' but contains a separator).
		{"zero-width joiner (U+200D)", "abc‍def", "abcdef"},
		// C1 control characters (0x80-0x9F) are non-ASCII control codes
		// that some terminals interpret as ANSI commands.
		{"c1 control (U+0085)", "beforeafter", "beforeafter"},
		{"unicode passthrough", "café 🚀 中文", "café 🚀 中文"},
		{"already clean", "no-op for clean strings", "no-op for clean strings"},
		{"empty", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := transport.SanitizeMessage(c.in)
			if got != c.want {
				t.Errorf("SanitizeMessage(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}
