package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/fatih/color"

	clitr "go.loglayer.dev/transports/cli/v2"
	"go.loglayer.dev/v2"
)

// makeLogger constructs a logger backed by a cli.Transport whose
// stdout / stderr are captured into the returned buffers. Color is
// forced off so assertions can match plain text.
func makeLogger(t *testing.T, cfg clitr.Config) (*loglayer.LogLayer, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	cfg.Stdout = &stdout
	cfg.Stderr = &stderr
	cfg.Color = clitr.ColorNever
	log := loglayer.New(loglayer.Config{
		Transport: clitr.New(cfg),
	})
	return log, &stdout, &stderr
}

func TestRoutesInfoAndDebugToStdout(t *testing.T) {
	log, stdout, stderr := makeLogger(t, clitr.Config{})
	log.SetLevel(loglayer.LogLevelDebug)

	log.Info("info line")
	log.Debug("debug line")

	if !strings.Contains(stdout.String(), "info line") {
		t.Errorf("info missing from stdout: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "debug: debug line") {
		t.Errorf("debug missing prefix or wrong stream: %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("nothing should land on stderr: %q", stderr.String())
	}
}

func TestRoutesWarnErrorFatalToStderr(t *testing.T) {
	log, stdout, stderr := makeLogger(t, clitr.Config{})

	log.Warn("a warn")
	log.Error("an error")

	if stdout.Len() != 0 {
		t.Errorf("nothing should land on stdout: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "warning: a warn") {
		t.Errorf("warn missing prefix: %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "error: an error") {
		t.Errorf("error missing prefix: %q", stderr.String())
	}
}

func TestInfoHasNoPrefix(t *testing.T) {
	log, stdout, _ := makeLogger(t, clitr.Config{})
	log.Info("hello world")

	if got := strings.TrimRight(stdout.String(), "\n"); got != "hello world" {
		t.Errorf("info output = %q, want %q", got, "hello world")
	}
}

func TestLevelPrefixOverride(t *testing.T) {
	log, stdout, stderr := makeLogger(t, clitr.Config{
		LevelPrefix: map[loglayer.LogLevel]string{
			loglayer.LogLevelInfo:  "INFO  ",
			loglayer.LogLevelWarn:  "WARN  ",
			loglayer.LogLevelDebug: "", // suppress default "debug: "
		},
	})
	log.SetLevel(loglayer.LogLevelDebug)

	log.Info("ix")
	log.Warn("wx")
	log.Debug("dx")

	if !strings.Contains(stdout.String(), "INFO  ix") {
		t.Errorf("info override missing: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "WARN  wx") {
		t.Errorf("warn override missing: %q", stderr.String())
	}
	// Debug prefix suppressed: just the message.
	if !strings.Contains(stdout.String(), "\ndx\n") && !strings.HasPrefix(stdout.String(), "dx\n") {
		// When info also fires, `dx` may not be at start; check it's
		// present without the default "debug: " prefix.
		if strings.Contains(stdout.String(), "debug: dx") {
			t.Errorf("debug prefix not suppressed: %q", stdout.String())
		}
	}
}

func TestDropsFieldsByDefault(t *testing.T) {
	log, stdout, _ := makeLogger(t, clitr.Config{})
	log.WithFields(loglayer.Fields{"requestID": "abc-123"}).
		WithMetadata(loglayer.Metadata{"user": "alice"}).
		Info("done")

	got := strings.TrimRight(stdout.String(), "\n")
	if got != "done" {
		t.Errorf("output should be just the message; got %q", got)
	}
}

func TestShowFieldsAppendsLogfmt(t *testing.T) {
	log, stdout, _ := makeLogger(t, clitr.Config{ShowFields: true})

	log.WithFields(loglayer.Fields{"requestID": "abc-123"}).
		WithMetadata(loglayer.Metadata{"user": "alice"}).
		Info("done")

	got := strings.TrimRight(stdout.String(), "\n")
	for _, want := range []string{"done", "requestID=abc-123", "user=alice"} {
		if !strings.Contains(got, want) {
			t.Errorf("output %q missing %q", got, want)
		}
	}
}

func TestShowFieldsQuotesValuesWithSpaces(t *testing.T) {
	log, stdout, _ := makeLogger(t, clitr.Config{ShowFields: true})

	log.WithMetadata(loglayer.Metadata{"path": "/var/log/my app"}).Info("ok")

	got := strings.TrimRight(stdout.String(), "\n")
	if !strings.Contains(got, `path="/var/log/my app"`) {
		t.Errorf("quoted value missing: %q", got)
	}
}

func TestColorNeverOmitsAnsiEscapes(t *testing.T) {
	log, _, stderr := makeLogger(t, clitr.Config{})
	log.Error("boom")

	if strings.ContainsRune(stderr.String(), 0x1b) {
		t.Errorf("ColorNever should produce no ANSI escapes: %q", stderr.String())
	}
}

func TestColorAlwaysEmitsAnsiEscapesEvenWhenPiped(t *testing.T) {
	var stdout, stderr bytes.Buffer
	log := loglayer.New(loglayer.Config{
		Transport: clitr.New(clitr.Config{
			Stdout: &stdout,
			Stderr: &stderr,
			Color:  clitr.ColorAlways,
		}),
	})
	log.Error("boom")

	if !strings.ContainsRune(stderr.String(), 0x1b) {
		t.Errorf("ColorAlways should produce ANSI escapes; got %q", stderr.String())
	}
}

func TestColorAutoDisabledWhenStdoutIsBuffer(t *testing.T) {
	// The pipe-to-buffer case is not a TTY, so ColorAuto should
	// resolve to no color.
	log, _, stderr := makeLogger(t, clitr.Config{Color: clitr.ColorAuto})
	log.Error("boom")

	if strings.ContainsRune(stderr.String(), 0x1b) {
		t.Errorf("ColorAuto with non-TTY stdout should not produce ANSI; got %q", stderr.String())
	}
}

func TestSanitizesMessages(t *testing.T) {
	// CRLF and ANSI ESC must be scrubbed so a user-controlled
	// message can't smuggle terminal escapes or forge log lines.
	log, stdout, _ := makeLogger(t, clitr.Config{})
	log.Info("first\nsecond\x1b[31mred\x1b[0m")

	got := strings.TrimRight(stdout.String(), "\n")
	if strings.ContainsRune(got, 0x1b) {
		t.Errorf("ANSI ESC leaked through: %q", got)
	}
	// The sanitizer's specific replacement is implementation-defined;
	// just assert raw line breaks don't make it through.
	if strings.Contains(got, "\nsecond") {
		t.Errorf("raw newline leaked through: %q", got)
	}
}

func TestRespectsLevelGating(t *testing.T) {
	log, stdout, stderr := makeLogger(t, clitr.Config{})
	log.SetLevel(loglayer.LogLevelWarn)

	log.Info("hidden")
	log.Debug("hidden too")
	log.Warn("visible")

	if strings.Contains(stdout.String(), "hidden") {
		t.Errorf("info should be filtered: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "warning: visible") {
		t.Errorf("warn should pass: %q", stderr.String())
	}
}

func TestDisableLevelPrefixSuppressesAllPrefixes(t *testing.T) {
	log, stdout, stderr := makeLogger(t, clitr.Config{
		DisableLevelPrefix: true,
	})
	log.SetLevel(loglayer.LogLevelDebug)

	log.Debug("d")
	log.Info("i")
	log.Warn("w")
	log.Error("e")

	// Stdout should contain just "d\ni\n"; no "debug: " prefix.
	if got := stdout.String(); got != "d\ni\n" {
		t.Errorf("stdout = %q, want %q", got, "d\ni\n")
	}
	// Stderr should contain just "w\ne\n"; no "warning: "/"error: ".
	if got := stderr.String(); got != "w\ne\n" {
		t.Errorf("stderr = %q, want %q", got, "w\ne\n")
	}
}

func TestDisableLevelPrefixOverridesLevelPrefixMap(t *testing.T) {
	// DisableLevelPrefix wins over a populated LevelPrefix map; the
	// master switch takes precedence over per-level overrides.
	log, stdout, _ := makeLogger(t, clitr.Config{
		DisableLevelPrefix: true,
		LevelPrefix: map[loglayer.LogLevel]string{
			loglayer.LogLevelInfo: "INFO  ",
		},
	})
	log.Info("hello")

	if got := strings.TrimRight(stdout.String(), "\n"); got != "hello" {
		t.Errorf("output = %q, want %q (DisableLevelPrefix should win)", got, "hello")
	}
}

func TestTableRenderingForMetadataSlice(t *testing.T) {
	log, stdout, _ := makeLogger(t, clitr.Config{})

	log.WithMetadata([]loglayer.Metadata{
		{"package": "transports/foo", "from": "v1.5.0", "to": "v1.6.0"},
		{"package": "transports/bar", "from": "v0.2.0", "to": "v1.0.0"},
	}).Info("Plan:")

	got := stdout.String()
	for _, want := range []string{
		"Plan:",
		"FROM",
		"PACKAGE",
		"TO",
		"transports/foo",
		"transports/bar",
		"v1.5.0",
		"v1.6.0",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q\nfull:\n%s", want, got)
		}
	}
	// Columns should be aligned (tabwriter padding).
	if !strings.Contains(got, "  ") {
		t.Errorf("expected column padding (2+ spaces); got:\n%s", got)
	}
}

func TestTableRenderingFromBareMapSlice(t *testing.T) {
	// []map[string]any (the underlying type) should also trigger
	// the table renderer, not just []loglayer.Metadata.
	log, stdout, _ := makeLogger(t, clitr.Config{})

	log.WithMetadata([]map[string]any{
		{"id": 1, "name": "alpha"},
		{"id": 2, "name": "beta"},
	}).Info("rows:")

	got := stdout.String()
	if !strings.Contains(got, "rows:") || !strings.Contains(got, "ID") || !strings.Contains(got, "NAME") {
		t.Errorf("table not rendered; got:\n%s", got)
	}
}

func TestTableRenderingFromMixedConcreteTypeAnySlice(t *testing.T) {
	// []any whose elements are a mix of map[string]any and
	// loglayer.Metadata (still uniformly map-shaped) should render
	// as a table. The "heterogeneous" case in the bail-out sense is
	// covered by TestTableRenderingSkippedForNonMapSlice.
	log, stdout, _ := makeLogger(t, clitr.Config{})

	log.WithMetadata([]any{
		map[string]any{"k": "v1"},
		loglayer.Metadata{"k": "v2"},
	}).Info("mixed:")

	got := stdout.String()
	if !strings.Contains(got, "mixed:") || !strings.Contains(got, "v1") || !strings.Contains(got, "v2") {
		t.Errorf("table not rendered; got:\n%s", got)
	}
}

func TestTableRenderingSkippedForNonMapSlice(t *testing.T) {
	// []any containing a non-map element must NOT render as a
	// table; the renderer bails out cleanly.
	log, stdout, _ := makeLogger(t, clitr.Config{})

	log.WithMetadata([]any{
		map[string]any{"k": "v"},
		"a string", // non-map element
	}).Info("skip:")

	got := stdout.String()
	// Just the message; no table.
	if !strings.Contains(got, "skip:") {
		t.Errorf("message missing: %q", got)
	}
	if strings.Contains(got, "K\n") || strings.Contains(got, "K  ") {
		t.Errorf("table was rendered for heterogeneous slice; got:\n%s", got)
	}
}

func TestTableRenderingHandlesMissingKeys(t *testing.T) {
	// Rows with different key sets should produce a table whose
	// columns are the union of all keys; missing values show as
	// empty cells.
	log, stdout, _ := makeLogger(t, clitr.Config{})

	log.WithMetadata([]loglayer.Metadata{
		{"a": 1, "b": 2},
		{"b": 3, "c": 4}, // missing 'a'
	}).Info("uneven:")

	got := stdout.String()
	for _, want := range []string{"A", "B", "C", "1", "2", "3", "4"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q\nfull:\n%s", want, got)
		}
	}
}

func TestTableRenderingDeterministicColumnOrder(t *testing.T) {
	// Run twice with reversed map iteration risk; the rendered
	// output must be identical because columns are sorted.
	cfg := clitr.Config{Color: clitr.ColorNever}
	for range 5 {
		var stdout bytes.Buffer
		c := cfg
		c.Stdout = &stdout
		log := loglayer.New(loglayer.Config{Transport: clitr.New(c)})
		log.WithMetadata([]loglayer.Metadata{
			{"z": 1, "a": 2, "m": 3},
		}).Info("ord:")

		// First column header should always be "A" (alphabetic).
		lines := strings.Split(strings.TrimRight(stdout.String(), "\n"), "\n")
		if len(lines) < 2 {
			t.Fatalf("expected at least 2 lines, got: %q", stdout.String())
		}
		header := lines[1] // line 0 is "ord:"; line 1 is the table header
		if !strings.HasPrefix(header, "A") {
			t.Errorf("header should start with A (sorted); got: %q", header)
		}
	}
}

func TestTableColumnOrderPinsLeadingColumns(t *testing.T) {
	log, stdout, _ := makeLogger(t, clitr.Config{
		TableColumnOrder: []string{"package", "changeset"},
	})

	log.WithMetadata([]loglayer.Metadata{
		{"bump": "minor", "changeset": "abc", "package": "transports/foo", "summary": "fix"},
		{"bump": "patch", "changeset": "def", "package": "transports/bar", "summary": "doc"},
	}).Info("status:")

	lines := strings.Split(strings.TrimRight(stdout.String(), "\n"), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected headline plus header row, got: %q", stdout.String())
	}
	header := lines[1]

	// PACKAGE first, CHANGESET second; BUMP and SUMMARY follow lex-sorted.
	wantOrder := []string{"PACKAGE", "CHANGESET", "BUMP", "SUMMARY"}
	idx := 0
	for _, want := range wantOrder {
		hit := strings.Index(header[idx:], want)
		if hit < 0 {
			t.Fatalf("header missing %q in %q", want, header)
		}
		idx += hit + len(want)
	}
}

func TestTableColumnOrderEmptyKeepsLexicographic(t *testing.T) {
	// Regression: empty / nil TableColumnOrder must produce the same
	// output as before the feature shipped.
	log, stdout, _ := makeLogger(t, clitr.Config{})

	log.WithMetadata([]loglayer.Metadata{
		{"z": 1, "a": 2, "m": 3},
	}).Info("ord:")

	lines := strings.Split(strings.TrimRight(stdout.String(), "\n"), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got: %q", stdout.String())
	}
	header := lines[1]
	wantOrder := []string{"A", "M", "Z"}
	idx := 0
	for _, want := range wantOrder {
		hit := strings.Index(header[idx:], want)
		if hit < 0 {
			t.Fatalf("header missing %q at-or-after position %d in %q", want, idx, header)
		}
		idx += hit + len(want)
	}
}

func TestTableColumnOrderSkipsMissingPinnedKeys(t *testing.T) {
	// A pinned key that doesn't appear in any row is silently skipped.
	log, stdout, _ := makeLogger(t, clitr.Config{
		TableColumnOrder: []string{"package", "doesnotexist", "changeset"},
	})

	log.WithMetadata([]loglayer.Metadata{
		{"package": "transports/foo", "changeset": "abc"},
	}).Info("status:")

	got := stdout.String()
	if strings.Contains(strings.ToUpper(got), "DOESNOTEXIST") {
		t.Errorf("missing pinned key leaked into output: %q", got)
	}
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	header := lines[1]
	pkgAt := strings.Index(header, "PACKAGE")
	csAt := strings.Index(header, "CHANGESET")
	if pkgAt < 0 || csAt < 0 || pkgAt > csAt {
		t.Errorf("PACKAGE should precede CHANGESET; got: %q", header)
	}
}

func TestTableColumnOrderUnpinnedKeysSortLexicographicallyAfter(t *testing.T) {
	// Keys NOT named in TableColumnOrder sort lexicographically after
	// the pinned ones.
	log, stdout, _ := makeLogger(t, clitr.Config{
		TableColumnOrder: []string{"package"},
	})

	log.WithMetadata([]loglayer.Metadata{
		{"package": "transports/foo", "z": 1, "a": 2, "m": 3},
	}).Info("ord:")

	lines := strings.Split(strings.TrimRight(stdout.String(), "\n"), "\n")
	header := lines[1]
	wantOrder := []string{"PACKAGE", "A", "M", "Z"}
	idx := 0
	for _, want := range wantOrder {
		hit := strings.Index(header[idx:], want)
		if hit < 0 {
			t.Fatalf("header missing %q at-or-after position %d in %q", want, idx, header)
		}
		idx += hit + len(want)
	}
}

func TestTableRenderingTakesPrecedenceOverShowFields(t *testing.T) {
	// When ShowFields is true AND metadata is a table-shaped slice,
	// the table renderer wins. Logfmt is dropped for that entry.
	log, stdout, _ := makeLogger(t, clitr.Config{ShowFields: true})

	log.WithMetadata([]loglayer.Metadata{
		{"x": 1},
	}).Info("hi")

	got := stdout.String()
	if !strings.Contains(got, "X\n") && !strings.Contains(got, "X  ") {
		t.Errorf("table not rendered; got:\n%s", got)
	}
	// No logfmt-style key=value tail on the message line.
	if strings.Contains(got, "hi x=") {
		t.Errorf("logfmt leaked through despite table rendering; got:\n%s", got)
	}
}

func TestTableRenderingDoesNotKickInForScalarOrMapMetadata(t *testing.T) {
	// Scalar metadata (a single map, a struct) should NOT trigger
	// table mode; that's the existing logfmt path.
	log, stdout, _ := makeLogger(t, clitr.Config{ShowFields: true})

	log.WithMetadata(loglayer.Metadata{"x": 1}).Info("scalar")

	got := strings.TrimRight(stdout.String(), "\n")
	if !strings.HasPrefix(got, "scalar ") {
		t.Errorf("expected logfmt path for single-map metadata; got: %q", got)
	}
	if strings.Contains(got, "X\n") {
		t.Errorf("table was rendered for non-array metadata; got: %q", got)
	}
}

func TestMetadataOnlyWithTableMetadata(t *testing.T) {
	// log.MetadataOnly([]Metadata{...}) should render the table
	// alone with no leading blank line. Empty message must not
	// produce an extra newline before the table.
	log, stdout, _ := makeLogger(t, clitr.Config{})

	log.MetadataOnly([]loglayer.Metadata{
		{"name": "alpha", "id": 1},
		{"name": "beta", "id": 2},
	})

	got := stdout.String()
	// First non-empty line should be the header (ID  NAME), not blank.
	if strings.HasPrefix(got, "\n") {
		t.Errorf("MetadataOnly produced leading blank line; got %q", got)
	}
	if !strings.Contains(got, "alpha") || !strings.Contains(got, "beta") {
		t.Errorf("table rows missing; got %q", got)
	}
}

func TestShowFieldsSanitizesValues(t *testing.T) {
	// ANSI ESC and CRLF inside a metadata value must not leak
	// through the ShowFields logfmt path. Same threat model as
	// message sanitization.
	log, stdout, _ := makeLogger(t, clitr.Config{ShowFields: true})

	log.WithMetadata(loglayer.Metadata{
		"path": "/var/log\nINJECTED",
		"ansi": "\x1b[31mfake-red\x1b[0m",
	}).Info("ok")

	got := stdout.String()
	if strings.ContainsRune(got, 0x1b) {
		t.Errorf("ANSI ESC leaked through ShowFields: %q", got)
	}
	if strings.Contains(got, "\nINJECTED") {
		t.Errorf("raw newline leaked through ShowFields: %q", got)
	}
}

func TestTableSanitizesCellValues(t *testing.T) {
	// Same as above for the table path.
	log, stdout, _ := makeLogger(t, clitr.Config{})

	log.WithMetadata([]loglayer.Metadata{
		{"name": "\x1b[31mred\x1b[0m"},
	}).Info("rows:")

	got := stdout.String()
	if strings.ContainsRune(got, 0x1b) {
		t.Errorf("ANSI ESC leaked through table cell: %q", got)
	}
}

type tableRow struct {
	Pkg  string `json:"package"`
	From string `json:"from"`
	To   string `json:"to"`
}

func TestTableRenderingFromSliceOfStruct(t *testing.T) {
	log, stdout, _ := makeLogger(t, clitr.Config{})

	log.WithMetadata([]tableRow{
		{Pkg: "transports/foo", From: "v1.5.0", To: "v1.6.0"},
		{Pkg: "transports/bar", From: "v0.2.0", To: "v1.0.0"},
	}).Info("Plan:")

	got := stdout.String()
	for _, want := range []string{
		"Plan:",
		"FROM",
		"PACKAGE", // JSON tag honored
		"TO",
		"transports/foo",
		"transports/bar",
		"v1.5.0",
		"v1.0.0",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q\nfull:\n%s", want, got)
		}
	}
}

func TestTableRenderingFromSliceOfPointerToStruct(t *testing.T) {
	log, stdout, _ := makeLogger(t, clitr.Config{})

	log.WithMetadata([]*tableRow{
		{Pkg: "transports/foo", From: "v1.5.0", To: "v1.6.0"},
	}).Info("ptr:")

	got := stdout.String()
	if !strings.Contains(got, "transports/foo") || !strings.Contains(got, "v1.5.0") {
		t.Errorf("slice-of-pointer-to-struct table not rendered:\n%s", got)
	}
}

func TestLevelColorOverride(t *testing.T) {
	// Custom Warn color (cyan) should win over the default yellow
	// when ANSI is forced on. Verify via the specific ANSI color
	// code in the wire output.
	var stdout, stderr bytes.Buffer
	log := loglayer.New(loglayer.Config{
		Transport: clitr.New(clitr.Config{
			Stdout: &stdout,
			Stderr: &stderr,
			Color:  clitr.ColorAlways,
			LevelColor: map[loglayer.LogLevel]*color.Color{
				loglayer.LogLevelWarn: color.New(color.FgCyan),
			},
		}),
	})

	log.Warn("hi")

	got := stderr.String()
	if !strings.Contains(got, "\x1b[36m") {
		// FgCyan = 36; FgYellow (default) = 33. Differentiates the
		// override from the default.
		t.Errorf("expected cyan ANSI escape \\x1b[36m; got: %q", got)
	}
}

func TestLevelColorNilSuppressesColorForOneLevel(t *testing.T) {
	// Setting an entry to nil renders that level without color
	// while keeping other defaults.
	var stdout, stderr bytes.Buffer
	log := loglayer.New(loglayer.Config{
		Transport: clitr.New(clitr.Config{
			Stdout: &stdout,
			Stderr: &stderr,
			Color:  clitr.ColorAlways,
			LevelColor: map[loglayer.LogLevel]*color.Color{
				loglayer.LogLevelWarn: nil,
			},
		}),
	})

	log.Warn("hi")  // nil color → no ANSI on this line
	log.Error("oh") // default red still applies

	parts := strings.Split(stderr.String(), "\n")
	if len(parts) < 2 {
		t.Fatalf("expected two stderr lines; got %q", stderr.String())
	}
	warnLine := parts[0]
	errorLine := parts[1]
	if strings.ContainsRune(warnLine, 0x1b) {
		t.Errorf("warn should be uncolored after nil override; got %q", warnLine)
	}
	if !strings.ContainsRune(errorLine, 0x1b) {
		t.Errorf("error should remain colored; got %q", errorLine)
	}
}

func TestSharedLevelColorAcrossTransportsIsolated(t *testing.T) {
	// Two transports configured with the SAME *color.Color but
	// different Color modes must each honor their own ANSI
	// decision. The transport is required to shallow-copy each
	// supplied color so EnableColor/DisableColor in New() don't
	// stomp the user's instance or each other.
	shared := color.New(color.FgCyan)
	override := map[loglayer.LogLevel]*color.Color{
		loglayer.LogLevelWarn: shared,
	}

	var aBuf, bBuf bytes.Buffer
	logA := loglayer.New(loglayer.Config{
		Transport: clitr.New(clitr.Config{
			Stdout:     &aBuf,
			Stderr:     &aBuf,
			Color:      clitr.ColorAlways,
			LevelColor: override,
		}),
	})
	// Constructing B AFTER A used to flip A's color off via the
	// shared pointer; verify A still emits ANSI after B exists.
	logB := loglayer.New(loglayer.Config{
		Transport: clitr.New(clitr.Config{
			Stdout:     &bBuf,
			Stderr:     &bBuf,
			Color:      clitr.ColorNever,
			LevelColor: override,
		}),
	})

	logA.Warn("from A")
	logB.Warn("from B")

	if !strings.ContainsRune(aBuf.String(), 0x1b) {
		t.Errorf("A (ColorAlways) should have ANSI; got %q", aBuf.String())
	}
	if strings.ContainsRune(bBuf.String(), 0x1b) {
		t.Errorf("B (ColorNever) should not have ANSI; got %q", bBuf.String())
	}
}

func TestEmptyMetadataOnlySuppressesBlankLine(t *testing.T) {
	// log.MetadataOnly with non-tabular metadata that produces an
	// empty body must not emit a stray newline. Empty slice +
	// table fast-path bail is the relevant edge.
	log, stdout, _ := makeLogger(t, clitr.Config{})

	log.MetadataOnly([]loglayer.Metadata{}) // empty slice; no table

	if got := stdout.String(); got != "" {
		t.Errorf("expected no output for empty MetadataOnly; got %q", got)
	}
}

func TestNilElementBailsTableFastPath(t *testing.T) {
	// A nil entry in the slice must drop the entire table for
	// consistency with the heterogeneous-slice rule. Both
	// []loglayer.Metadata and []map[string]any fast paths must
	// match the reflection path's behavior.
	log, stdout, _ := makeLogger(t, clitr.Config{})

	log.WithMetadata([]loglayer.Metadata{
		nil,
		{"k": "v"},
	}).Info("nilmeta:")

	got := stdout.String()
	if !strings.Contains(got, "nilmeta:") {
		t.Errorf("expected message; got %q", got)
	}
	if strings.Contains(got, "K\n") || strings.Contains(got, "K  ") {
		t.Errorf("table should not render when slice contains nil; got %q", got)
	}
}

func TestWithPrefixRendersInline(t *testing.T) {
	// WithPrefix renders between the level prefix and the message
	// body, plain text when ColorNever.
	log, stdout, stderr := makeLogger(t, clitr.Config{})

	prefixed := log.WithPrefix("[auth]")
	prefixed.Info("starting")
	prefixed.Warn("retrying")
	prefixed.Error("failed")

	if got := strings.TrimRight(stdout.String(), "\n"); got != "[auth] starting" {
		t.Errorf("Info: got %q, want %q", got, "[auth] starting")
	}
	if !strings.Contains(stderr.String(), "warning: [auth] retrying") {
		t.Errorf("Warn missing inline prefix: %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "error: [auth] failed") {
		t.Errorf("Error missing inline prefix: %q", stderr.String())
	}
}

func TestWithPrefixGetsDimGreyAnsiSeparateFromLevel(t *testing.T) {
	// With Color: ColorAlways and a warn-level entry, the level
	// prefix and message body should carry the warn color (yellow,
	// FgYellow = 33), while the user prefix should carry FgHiBlack
	// (90). Both ANSI color codes appear in the line.
	var stdout, stderr bytes.Buffer
	log := loglayer.New(loglayer.Config{
		Transport: clitr.New(clitr.Config{
			Stdout: &stdout,
			Stderr: &stderr,
			Color:  clitr.ColorAlways,
		}),
	})
	log.WithPrefix("[auth]").Warn("retrying")

	out := stderr.String()
	if !strings.Contains(out, "\x1b[33m") {
		t.Errorf("expected yellow (33) ANSI for warn level/body; got %q", out)
	}
	if !strings.Contains(out, "\x1b[90m") {
		t.Errorf("expected FgHiBlack (90) ANSI for user prefix; got %q", out)
	}
}

func TestWithPrefixSanitizesAnsiSmuggling(t *testing.T) {
	// A prefix loaded from env or config that carries ANSI escapes
	// must not smuggle them through cli's smart-rendering path.
	log, stdout, _ := makeLogger(t, clitr.Config{})

	log.WithPrefix("\x1b[31mFAKE\x1b[0m").Info("ok")

	got := stdout.String()
	if strings.ContainsRune(got, 0x1b) {
		t.Errorf("ANSI ESC in WithPrefix value leaked through: %q", got)
	}
}

func TestWithPrefixWithNoLevelColorOnInfo(t *testing.T) {
	// At info level the default level color is nil. With a user
	// prefix, only the user prefix carries ANSI; the message body
	// is unstyled.
	var stdout, stderr bytes.Buffer
	log := loglayer.New(loglayer.Config{
		Transport: clitr.New(clitr.Config{
			Stdout: &stdout,
			Stderr: &stderr,
			Color:  clitr.ColorAlways,
		}),
	})
	log.WithPrefix("[auth]").Info("hi")

	out := stdout.String()
	if !strings.Contains(out, "\x1b[90m[auth] \x1b[0m") {
		t.Errorf("user prefix should be wrapped in FgHiBlack ANSI; got %q", out)
	}
}

func TestColorAlwaysDoesNotTintTableBody(t *testing.T) {
	// When color is forced on AND the level is warn (yellow), the
	// headline should be tinted but the table body should not.
	// Otherwise the data rows look like warnings.
	var stdout, stderr bytes.Buffer
	log := loglayer.New(loglayer.Config{
		Transport: clitr.New(clitr.Config{
			Stdout: &stdout,
			Stderr: &stderr,
			Color:  clitr.ColorAlways,
		}),
	})

	log.WithMetadata([]loglayer.Metadata{
		{"k": "v"},
	}).Warn("rows:")

	out := stderr.String()
	head, body, ok := strings.Cut(out, "\n")
	if !ok {
		t.Fatalf("expected multi-line output; got %q", out)
	}
	if !strings.ContainsRune(head, 0x1b) {
		t.Errorf("expected ANSI on the headline; got %q", head)
	}
	if strings.ContainsRune(body, 0x1b) {
		t.Errorf("table body should not be tinted; got %q", body)
	}
}

func TestDisableFatalExitOptOut(t *testing.T) {
	// Sanity check: with Config.DisableFatalExit, a Fatal call
	// should write the message and return rather than os.Exit.
	var stdout, stderr bytes.Buffer
	log := loglayer.New(loglayer.Config{
		Transport: clitr.New(clitr.Config{
			Stdout: &stdout,
			Stderr: &stderr,
			Color:  clitr.ColorNever,
		}),
		DisableFatalExit: true,
	})

	log.Fatal("the end")

	if !strings.Contains(stderr.String(), "fatal: the end") {
		t.Errorf("fatal output missing: %q", stderr.String())
	}
}
