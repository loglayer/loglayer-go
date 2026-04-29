package loglayer_test

import (
	"errors"
	"runtime"
	"strings"
	"testing"

	"go.loglayer.dev"
	"go.loglayer.dev/internal/lltest"
)

func newSourceLogger(t *testing.T, addSource bool) (*loglayer.LogLayer, *lltest.TestLoggingLibrary) {
	t.Helper()
	lib := &lltest.TestLoggingLibrary{}
	log := loglayer.New(loglayer.Config{
		Transport:        lltest.New(lltest.Config{Library: lib}),
		Source:           loglayer.SourceConfig{Enabled: addSource},
		DisableFatalExit: true,
	})
	return log, lib
}

func sourceFromLine(t *testing.T, line *lltest.LogLine) *loglayer.Source {
	t.Helper()
	if line.Data == nil {
		t.Fatalf("expected Data to be populated, got nil")
	}
	v, ok := line.Data["source"]
	if !ok {
		t.Fatalf("missing 'source' key: %v", line.Data)
	}
	s, ok := v.(*loglayer.Source)
	if !ok {
		t.Fatalf("source key is not *loglayer.Source: %T", v)
	}
	return s
}

// AddSource off: no source key in Data.
func TestSource_OffByDefault(t *testing.T) {
	t.Parallel()
	log, lib := newSourceLogger(t, false)
	log.Info("hi")
	line := lib.PopLine()
	if line.Data != nil {
		if _, ok := line.Data["source"]; ok {
			t.Errorf("source should not appear when AddSource is off: %v", line.Data)
		}
	}
}

// AddSource on: source captured at the LogLayer.Info call site.
func TestSource_DirectInfo(t *testing.T) {
	t.Parallel()
	log, lib := newSourceLogger(t, true)
	log.Info("here") // CAPTURE_INFO

	line := lib.PopLine()
	src := sourceFromLine(t, line)
	if !strings.HasSuffix(src.File, "source_test.go") {
		t.Errorf("file should be the test file, got %q", src.File)
	}
	if src.Line == 0 {
		t.Errorf("line should be captured, got 0")
	}
	if !strings.Contains(src.Function, "TestSource_DirectInfo") {
		t.Errorf("function should be the test, got %q", src.Function)
	}
}

// AddSource on: source captured at builder.Info call site (chained from
// WithMetadata). Confirms the builder path threads source identically.
func TestSource_BuilderInfo(t *testing.T) {
	t.Parallel()
	log, lib := newSourceLogger(t, true)
	log.WithMetadata(loglayer.M{"k": "v"}).Info("chained")

	line := lib.PopLine()
	src := sourceFromLine(t, line)
	if !strings.HasSuffix(src.File, "source_test.go") {
		t.Errorf("file should be test file, got %q", src.File)
	}
	if !strings.Contains(src.Function, "TestSource_BuilderInfo") {
		t.Errorf("function: got %q, want containing TestSource_BuilderInfo", src.Function)
	}
}

func TestSource_AllLevelsCaptureFromUserSite(t *testing.T) {
	t.Parallel()
	log, lib := newSourceLogger(t, true)

	log.Debug("d")
	log.Info("i")
	log.Warn("w")
	log.Error("e")

	for _, line := range lib.Lines() {
		src := sourceFromLine(t, &line)
		if !strings.Contains(src.Function, "TestSource_AllLevelsCaptureFromUserSite") {
			t.Errorf("level=%v: function=%q", line.Level, src.Function)
		}
	}
}

func TestSource_BuilderAllLevels(t *testing.T) {
	t.Parallel()
	log, lib := newSourceLogger(t, true)

	log.WithError(errors.New("e")).Debug("d")
	log.WithError(errors.New("e")).Info("i")
	log.WithError(errors.New("e")).Warn("w")
	log.WithError(errors.New("e")).Error("e")

	for _, line := range lib.Lines() {
		src := sourceFromLine(t, &line)
		if !strings.Contains(src.Function, "TestSource_BuilderAllLevels") {
			t.Errorf("level=%v: function=%q", line.Level, src.Function)
		}
	}
}

func TestSource_MetadataOnly(t *testing.T) {
	t.Parallel()
	log, lib := newSourceLogger(t, true)
	log.MetadataOnly(loglayer.M{"k": "v"})
	line := lib.PopLine()
	src := sourceFromLine(t, line)
	if !strings.Contains(src.Function, "TestSource_MetadataOnly") {
		t.Errorf("function: got %q", src.Function)
	}
}

func TestSource_ErrorOnly(t *testing.T) {
	t.Parallel()
	log, lib := newSourceLogger(t, true)
	log.ErrorOnly(errors.New("oops"))
	line := lib.PopLine()
	src := sourceFromLine(t, line)
	if !strings.Contains(src.Function, "TestSource_ErrorOnly") {
		t.Errorf("function: got %q", src.Function)
	}
}

// Raw with explicit Source uses it as-is (no runtime capture).
func TestSource_RawHonorsExplicitSource(t *testing.T) {
	t.Parallel()
	log, lib := newSourceLogger(t, true)

	explicit := &loglayer.Source{
		Function: "explicit/foo.Bar",
		File:     "/tmp/synthetic.go",
		Line:     42,
	}
	log.Raw(loglayer.RawLogEntry{
		LogLevel: loglayer.LogLevelInfo,
		Messages: []any{"raw"},
		Source:   explicit,
	})

	line := lib.PopLine()
	src := sourceFromLine(t, line)
	if src != explicit {
		t.Errorf("Raw should pass through the explicit *Source verbatim; got %+v", src)
	}
}

// Raw without an explicit Source captures at the Raw call site when
// AddSource is true.
func TestSource_RawCapturesWhenAddSourceTrue(t *testing.T) {
	t.Parallel()
	log, lib := newSourceLogger(t, true)
	log.Raw(loglayer.RawLogEntry{
		LogLevel: loglayer.LogLevelInfo,
		Messages: []any{"raw"},
	})
	line := lib.PopLine()
	src := sourceFromLine(t, line)
	if !strings.Contains(src.Function, "TestSource_RawCapturesWhenAddSourceTrue") {
		t.Errorf("function: got %q", src.Function)
	}
}

// Raw with no Source and AddSource false yields no source key.
func TestSource_RawNoCaptureWhenDisabled(t *testing.T) {
	t.Parallel()
	log, lib := newSourceLogger(t, false)
	log.Raw(loglayer.RawLogEntry{
		LogLevel: loglayer.LogLevelInfo,
		Messages: []any{"raw"},
	})
	line := lib.PopLine()
	if line.Data != nil {
		if _, ok := line.Data["source"]; ok {
			t.Errorf("source should not appear: %v", line.Data)
		}
	}
}

func TestSource_CustomFieldName(t *testing.T) {
	t.Parallel()
	lib := &lltest.TestLoggingLibrary{}
	log := loglayer.New(loglayer.Config{
		Transport:        lltest.New(lltest.Config{Library: lib}),
		Source:           loglayer.SourceConfig{Enabled: true, FieldName: "caller"},
		DisableFatalExit: true,
	})
	log.Info("hi")
	line := lib.PopLine()
	if _, ok := line.Data["source"]; ok {
		t.Errorf("default key 'source' should not be used when Source.FieldName is set: %v", line.Data)
	}
	if _, ok := line.Data["caller"]; !ok {
		t.Errorf("expected source under 'caller', got %v", line.Data)
	}
}

func TestSource_SourceFromPC(t *testing.T) {
	t.Parallel()
	pcs := make([]uintptr, 1)
	n := runtime.Callers(1, pcs)
	if n < 1 {
		t.Skip("no PC available from runtime.Callers")
	}
	src := loglayer.SourceFromPC(pcs[0])
	if src == nil {
		t.Fatal("SourceFromPC returned nil for a valid PC")
	}
	if !strings.Contains(src.Function, "TestSource_SourceFromPC") {
		t.Errorf("function: got %q", src.Function)
	}
	if src.Line == 0 {
		t.Errorf("line missing")
	}
}

func TestSource_SourceFromPC_ZeroPC(t *testing.T) {
	t.Parallel()
	if loglayer.SourceFromPC(0) != nil {
		t.Error("SourceFromPC(0) should be nil")
	}
}

// Source flows through OnBeforeDataOut: a plugin can observe and rewrite
// the source key just like any other Data entry. (Not a typical use case,
// but proves Source isn't bypassing the plugin pipeline.)
func TestSource_VisibleToPlugins(t *testing.T) {
	t.Parallel()
	log, lib := newSourceLogger(t, true)
	log.AddPlugin(loglayer.NewDataHook("sanitize-source", func(p loglayer.BeforeDataOutParams) loglayer.Data {
		if s, ok := p.Data["source"].(*loglayer.Source); ok && s != nil {
			// Plugin sanitizes the file path.
			p.Data["source"] = &loglayer.Source{Function: s.Function, File: "REDACTED", Line: s.Line}
		}
		return p.Data
	}))

	log.Info("hi")
	line := lib.PopLine()
	src := sourceFromLine(t, line)
	if src.File != "REDACTED" {
		t.Errorf("plugin should have rewritten file to REDACTED, got %q", src.File)
	}
}
