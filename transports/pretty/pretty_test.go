package pretty_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/fatih/color"

	"go.loglayer.dev/transports/pretty/v2"
	"go.loglayer.dev/v2"
	"go.loglayer.dev/v2/transport"
)

const fixedTime = "12:34:56.789"

func fixedTimestamp(time.Time) string { return fixedTime }

func newLogger(cfg pretty.Config) (*loglayer.LogLayer, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	cfg.Writer = buf
	cfg.NoColor = true
	if cfg.TimestampFn == nil {
		cfg.TimestampFn = fixedTimestamp
	}
	if cfg.BaseConfig.ID == "" {
		cfg.BaseConfig.ID = "pretty"
	}
	t := pretty.New(cfg)
	return loglayer.New(loglayer.Config{DisableFatalExit: true, Transport: t}), buf
}

func TestInlineSimpleMessage(t *testing.T) {
	log, buf := newLogger(pretty.Config{})
	log.Info("hello")
	got := buf.String()
	want := "12:34:56.789 ▶ INFO hello\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestInlineWithMetadataMap(t *testing.T) {
	log, buf := newLogger(pretty.Config{})
	log.WithMetadata(map[string]any{"user": "alice", "n": 42}).Info("served")
	got := buf.String()
	// keys are sorted
	want := "12:34:56.789 ▶ INFO served n=42 user=alice\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestInlineWithMetadataStruct(t *testing.T) {
	type user struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	log, buf := newLogger(pretty.Config{})
	log.WithMetadata(user{ID: 7, Name: "Alice"}).Info("hi")
	got := buf.String()
	want := "12:34:56.789 ▶ INFO hi id=7 name=Alice\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestInlineQuotesAmbiguousValues(t *testing.T) {
	log, buf := newLogger(pretty.Config{})
	log.WithMetadata(map[string]any{"text": "with spaces"}).Info("msg")
	got := buf.String()
	if !strings.Contains(got, `text="with spaces"`) {
		t.Errorf("expected quoted value, got: %q", got)
	}
}

func TestInlineNoData(t *testing.T) {
	log, buf := newLogger(pretty.Config{})
	log.Info("plain")
	got := buf.String()
	if strings.Contains(got, "  ") {
		t.Errorf("no double-space when there's no data, got: %q", got)
	}
}

func TestMessageOnlyMode(t *testing.T) {
	log, buf := newLogger(pretty.Config{ViewMode: pretty.ViewModeMessageOnly})
	log.WithMetadata(map[string]any{"k": "v"}).Info("msg")
	got := buf.String()
	want := "12:34:56.789 ▶ INFO msg\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if strings.Contains(got, "k=v") {
		t.Error("message-only mode should drop metadata")
	}
}

func TestExpandedMode(t *testing.T) {
	log, buf := newLogger(pretty.Config{ViewMode: pretty.ViewModeExpanded})
	log.WithMetadata(map[string]any{
		"user":   "alice",
		"nested": map[string]any{"a": 1, "b": 2},
	}).Info("served")
	got := buf.String()
	want := "12:34:56.789 ▶ INFO served\n  nested:\n    a: 1\n    b: 2\n  user: alice\n"
	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestExpandedModeArrays(t *testing.T) {
	log, buf := newLogger(pretty.Config{ViewMode: pretty.ViewModeExpanded})
	log.WithMetadata(map[string]any{
		"items": []any{"a", "b", "c"},
	}).Info("list")
	got := buf.String()
	want := "12:34:56.789 ▶ INFO list\n  items:\n    - a\n    - b\n    - c\n"
	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestExpandedModeNoData(t *testing.T) {
	log, buf := newLogger(pretty.Config{ViewMode: pretty.ViewModeExpanded})
	log.Info("plain")
	got := buf.String()
	want := "12:34:56.789 ▶ INFO plain\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// Three-level nesting renders as YAML at every depth, with column
// alignment applied per level (each level pads to its own longest
// same-line key).
func TestExpandedModeDeeplyNested(t *testing.T) {
	log, buf := newLogger(pretty.Config{ViewMode: pretty.ViewModeExpanded})
	log.WithMetadata(map[string]any{
		"user": map[string]any{
			"id": 42,
			"address": map[string]any{
				"city": "Brooklyn",
				"zip":  "11201",
			},
		},
	}).Info("served")
	got := buf.String()
	// At each level, sibling keys with same-line values pad to the longest:
	//  level 1: only "user" (multi-line, no padding needed)
	//  level 2: "address" (multi-line) + "id" (same-line; alone, no pad)
	//  level 3: "city" (len 4) and "zip" (len 3); maxKey = 4, "zip" pads.
	want := "" +
		"12:34:56.789 ▶ INFO served\n" +
		"  user:\n" +
		"    address:\n" +
		"      city: Brooklyn\n" +
		"      zip:  11201\n" +
		"    id: 42\n"
	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

// Column alignment: same-level scalar keys pad to the longest sibling
// so their values line up. Multi-line keys (whose values render on
// their own lines) don't participate in the alignment width.
func TestExpandedModeAlignedColumns(t *testing.T) {
	log, buf := newLogger(pretty.Config{ViewMode: pretty.ViewModeExpanded})
	log.WithMetadata(map[string]any{
		"a":           1,
		"longerKey":   "v",
		"middle":      true,
		"nested":      map[string]any{"x": 1}, // multi-line, doesn't count
		"alsoIgnored": []any{"a", "b"},        // multi-line, doesn't count
	}).Info("aligned")
	got := buf.String()
	// Sorted keys: a, alsoIgnored (multi-line), longerKey, middle, nested (multi-line).
	// Among same-line keys: a (1), longerKey (9), middle (6). maxKey = 9.
	want := "" +
		"12:34:56.789 ▶ INFO aligned\n" +
		"  a:         1\n" +
		"  alsoIgnored:\n" +
		"    - a\n" +
		"    - b\n" +
		"  longerKey: v\n" +
		"  middle:    true\n" +
		"  nested:\n" +
		"    x: 1\n"
	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestLevels(t *testing.T) {
	cases := []struct {
		fn    func(*loglayer.LogLayer)
		level string
	}{
		{func(l *loglayer.LogLayer) { l.Debug("x") }, "DEBUG"},
		{func(l *loglayer.LogLayer) { l.Info("x") }, "INFO"},
		{func(l *loglayer.LogLayer) { l.Warn("x") }, "WARN"},
		{func(l *loglayer.LogLayer) { l.Error("x") }, "ERROR"},
		{func(l *loglayer.LogLayer) { l.Fatal("x") }, "FATAL"},
	}
	for _, c := range cases {
		log, buf := newLogger(pretty.Config{})
		c.fn(log)
		if !strings.Contains(buf.String(), "▶ "+c.level+" ") {
			t.Errorf("level %s not in output: %q", c.level, buf.String())
		}
	}
}

func TestFatalDoesNotExit(t *testing.T) {
	log, buf := newLogger(pretty.Config{})
	log.Fatal("survives")
	if !strings.Contains(buf.String(), "FATAL") {
		t.Errorf("expected fatal label, got %q", buf.String())
	}
}

func TestContextRendersAsData(t *testing.T) {
	log, buf := newLogger(pretty.Config{})
	log = log.WithFields(loglayer.Fields{"requestId": "abc"})
	log.Info("ctx test")
	got := buf.String()
	if !strings.Contains(got, "requestId=abc") {
		t.Errorf("expected requestId in output, got %q", got)
	}
}

func TestErrorRendersAsData(t *testing.T) {
	log, buf := newLogger(pretty.Config{})
	log.WithError(errors.New("boom")).Error("failed")
	got := buf.String()
	if !strings.Contains(got, "err=") {
		t.Errorf("expected err= in output, got %q", got)
	}
	if !strings.Contains(got, "boom") {
		t.Errorf("expected error message in output, got %q", got)
	}
}

func TestShowLogID(t *testing.T) {
	log, buf := newLogger(pretty.Config{ShowLogID: true})
	log.Info("first")
	log.Info("second")
	got := buf.String()
	// Two lines, each containing a [hex-id] token
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %q", len(lines), got)
	}
	for i, line := range lines {
		if !strings.Contains(line, "[") || !strings.Contains(line, "]") {
			t.Errorf("line %d missing log id: %q", i, line)
		}
	}
}

func TestLevelFiltering(t *testing.T) {
	buf := &bytes.Buffer{}
	tr := pretty.New(pretty.Config{
		BaseConfig:  transport.BaseConfig{ID: "pretty", Level: loglayer.LogLevelError},
		Writer:      buf,
		NoColor:     true,
		TimestampFn: fixedTimestamp,
	})
	log := loglayer.New(loglayer.Config{DisableFatalExit: true, Transport: tr})
	log.Warn("dropped")
	if buf.Len() != 0 {
		t.Errorf("warn should be filtered, got: %q", buf.String())
	}
	log.Error("passes")
	if !strings.Contains(buf.String(), "ERROR") {
		t.Errorf("error should pass, got: %q", buf.String())
	}
}

func TestCustomTimestampFormat(t *testing.T) {
	// Build directly without the newLogger helper so we don't get the default
	// TimestampFn injected.
	buf := &bytes.Buffer{}
	tr := pretty.New(pretty.Config{
		BaseConfig:      transport.BaseConfig{ID: "pretty"},
		Writer:          buf,
		NoColor:         true,
		TimestampFormat: "2006-01-02",
	})
	log := loglayer.New(loglayer.Config{DisableFatalExit: true, Transport: tr})
	log.Info("date")
	got := buf.String()
	// Sanity-check the date pattern (year-month-day) shows up.
	if !strings.Contains(got, time.Now().Format("2006")) {
		t.Errorf("expected date-formatted timestamp, got: %q", got)
	}
}

func TestThemeApplied(t *testing.T) {
	// fatih/color auto-disables when stdout is not a TTY (the case in tests),
	// so force-enable for this assertion and restore afterwards.
	prev := color.NoColor
	color.NoColor = false
	defer func() { color.NoColor = prev }()

	buf := &bytes.Buffer{}
	tr := pretty.New(pretty.Config{
		BaseConfig:  transport.BaseConfig{ID: "pretty"},
		Writer:      buf,
		Theme:       pretty.Neon(),
		TimestampFn: fixedTimestamp,
	})
	log := loglayer.New(loglayer.Config{DisableFatalExit: true, Transport: tr})
	log.Info("colored")
	got := buf.String()
	if !strings.Contains(got, "\x1b[") {
		t.Errorf("expected ANSI escape codes, got: %q", got)
	}
}

func TestInlineDepthLimit(t *testing.T) {
	log, buf := newLogger(pretty.Config{MaxInlineDepth: 1})
	log.WithMetadata(map[string]any{
		"shallow": "ok",
		"deep":    map[string]any{"nested": "x"},
	}).Info("depth")
	got := buf.String()
	// At depth 1, the nested map should collapse to {...}
	if !strings.Contains(got, "deep={...}") {
		t.Errorf("expected deep={...} truncation, got: %q", got)
	}
	if !strings.Contains(got, "shallow=ok") {
		t.Errorf("expected shallow=ok, got: %q", got)
	}
}

func TestMultipleMessages(t *testing.T) {
	log, buf := newLogger(pretty.Config{})
	log.Info("part1", "part2")
	got := buf.String()
	if !strings.Contains(got, "part1 part2") {
		t.Errorf("expected joined message, got %q", got)
	}
}

func TestPretty_MultilineRendersAcrossLines(t *testing.T) {
	log, buf := newLogger(pretty.Config{})
	log.Info(loglayer.Multiline("Header:", "  port: 8080"))
	got := buf.String()
	if !strings.Contains(got, "Header:\n  port: 8080") {
		t.Errorf("expected multi-line headline; got %q", got)
	}
}

func TestPretty_BareNewlineStringStillStripped(t *testing.T) {
	// No wrapper, no trust: a bare "\n" must be stripped.
	log, buf := newLogger(pretty.Config{})
	log.Info("a\nb")
	got := buf.String()
	// The pretty header is one line; the body should be "ab" with no
	// newline inside it. Total newlines in the output should be 1
	// (the trailing one from Fprintln).
	if strings.Count(got, "\n") != 1 {
		t.Errorf("bare \\n must strip; rendered: %q", got)
	}
}

func TestPretty_MixedStringAndMultiline(t *testing.T) {
	log, buf := newLogger(pretty.Config{})
	log.Info("Header:", loglayer.Multiline("a", "b"))
	got := buf.String()
	if !strings.Contains(got, "Header: a\nb") {
		t.Errorf("expected mixed multi-line; got %q", got)
	}
}

func TestPretty_PrefixFoldsIntoFirstAuthoredLine(t *testing.T) {
	log, buf := newLogger(pretty.Config{})
	prefixed := log.WithPrefix("[svc]")
	prefixed.Info(loglayer.Multiline("a", "b"))
	got := buf.String()
	if !strings.Contains(got, "[svc] a\nb") {
		t.Errorf("expected prefix on first line; got %q", got)
	}
}
