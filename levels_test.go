package loglayer_test

import (
	"errors"
	"testing"

	"go.loglayer.dev/v2"
)

func TestSetLevel(t *testing.T) {
	log, lib := setup(t)
	log.SetLevel(loglayer.LogLevelWarn)

	log.Info("should be dropped")
	log.Debug("should be dropped")
	if lib.Len() != 0 {
		t.Errorf("expected no lines below warn, got %d", lib.Len())
	}

	log.Warn("should appear")
	log.Error("should appear")
	log.Fatal("should appear")
	if lib.Len() != 3 {
		t.Errorf("expected 3 lines at/above warn, got %d", lib.Len())
	}
}

func TestDisableLogging(t *testing.T) {
	log, lib := setup(t)
	log.DisableLogging()
	log.Info("silenced")
	log.Error("also silenced")
	if lib.Len() != 0 {
		t.Errorf("expected no lines after DisableLogging, got %d", lib.Len())
	}
}

func TestEnableLogging(t *testing.T) {
	log, lib := setup(t)
	log.DisableLogging()
	log.EnableLogging()
	log.Info("back on")
	if lib.Len() != 1 {
		t.Errorf("expected 1 line after re-enable, got %d", lib.Len())
	}
}

func TestDisableIndividualLevel(t *testing.T) {
	log, lib := setup(t)
	log.DisableLevel(loglayer.LogLevelDebug)
	log.Debug("dropped")
	log.Info("kept")
	if lib.Len() != 1 {
		t.Errorf("expected 1 line (info), got %d", lib.Len())
	}
}

func TestIsLevelEnabled(t *testing.T) {
	log, _ := setup(t)
	log.SetLevel(loglayer.LogLevelWarn)
	if log.IsLevelEnabled(loglayer.LogLevelInfo) {
		t.Error("info should be disabled after SetLevel(warn)")
	}
	if !log.IsLevelEnabled(loglayer.LogLevelError) {
		t.Error("error should be enabled after SetLevel(warn)")
	}
}

func TestDisabledConfig(t *testing.T) {
	log, lib := setupWithConfig(t, loglayer.Config{Disabled: true})
	log.Info("should not appear")
	if lib.Len() != 0 {
		t.Errorf("expected no lines when Disabled=true, got %d", lib.Len())
	}
}

// Trace sits below Debug. SetLevel(Debug) suppresses Trace; SetLevel(Trace)
// keeps everything.
func TestTrace_LevelOrdering(t *testing.T) {
	log, lib := setup(t)
	log.SetLevel(loglayer.LogLevelDebug)
	log.Trace("dropped")
	if lib.Len() != 0 {
		t.Errorf("Trace should be suppressed by SetLevel(Debug), got %d lines", lib.Len())
	}

	log.SetLevel(loglayer.LogLevelTrace)
	log.Trace("kept")
	line := lib.PopLine()
	if line == nil || line.Level != loglayer.LogLevelTrace {
		t.Errorf("Trace should emit at LogLevelTrace, got %v", line)
	}
}

// Trace via the builder works the same as the direct method.
func TestTrace_Builder(t *testing.T) {
	log, lib := setup(t)
	log.WithMetadata(loglayer.M{"k": "v"}).Trace("via builder")
	line := lib.PopLine()
	if line == nil || line.Level != loglayer.LogLevelTrace {
		t.Fatalf("expected one Trace line, got %v", line)
	}
	m := line.Metadata.(loglayer.Metadata)
	if m["k"] != "v" {
		t.Errorf("metadata not preserved on Trace builder: %v", m)
	}
}

// Panic emits then panics with the joined message string. Recover so the
// test runner doesn't crash; assert the panic value matches.
func TestPanic_EmitsThenPanics(t *testing.T) {
	log, lib := setup(t)

	var recovered any
	func() {
		defer func() { recovered = recover() }()
		log.Panic("boom")
	}()

	if recovered == nil {
		t.Fatal("Panic should have panicked, recovered nil")
	}
	if got, ok := recovered.(string); !ok || got != "boom" {
		t.Errorf("panic value: got %v (%T), want \"boom\"", recovered, recovered)
	}
	line := lib.PopLine()
	if line == nil || line.Level != loglayer.LogLevelPanic {
		t.Errorf("expected one LogLevelPanic line, got %v", line)
	}
}

// Panic via the builder works the same.
func TestPanic_Builder(t *testing.T) {
	log, lib := setup(t)

	var recovered any
	func() {
		defer func() { recovered = recover() }()
		log.WithError(errors.New("test error")).Panic("crash")
	}()

	if recovered != "crash" {
		t.Errorf("panic value: got %v, want \"crash\"", recovered)
	}
	line := lib.PopLine()
	if line == nil || line.Level != loglayer.LogLevelPanic {
		t.Fatalf("expected one panic-level line, got %v", line)
	}
	if line.Data["err"].(map[string]any)["message"] != "test error" {
		t.Errorf("error not preserved on Panic builder: %v", line.Data)
	}
}

// SetLevel(Panic) suppresses everything below Panic, including Fatal.
func TestPanic_LevelOrdering(t *testing.T) {
	log, lib := setup(t)
	log.SetLevel(loglayer.LogLevelPanic)

	log.Fatal("dropped") // DisableFatalExit is true in setup
	log.Error("dropped")
	if lib.Len() != 0 {
		t.Errorf("expected no lines below Panic, got %d", lib.Len())
	}

	func() {
		defer func() { _ = recover() }()
		log.Panic("kept")
	}()
	line := lib.PopLine()
	if line == nil || line.Level != loglayer.LogLevelPanic {
		t.Errorf("Panic should emit at SetLevel(Panic), got %v", line)
	}
}

// IsLevelEnabled honors the new levels.
func TestTraceAndPanic_IsLevelEnabled(t *testing.T) {
	log, _ := setup(t)
	if !log.IsLevelEnabled(loglayer.LogLevelTrace) {
		t.Error("Trace should be enabled by default")
	}
	if !log.IsLevelEnabled(loglayer.LogLevelPanic) {
		t.Error("Panic should be enabled by default")
	}

	log.DisableLevel(loglayer.LogLevelTrace)
	if log.IsLevelEnabled(loglayer.LogLevelTrace) {
		t.Error("Trace should be disabled after DisableLevel")
	}
}

func TestParseLogLevel_TraceAndPanic(t *testing.T) {
	cases := []struct {
		in     string
		want   loglayer.LogLevel
		wantOK bool
	}{
		{"trace", loglayer.LogLevelTrace, true},
		{"panic", loglayer.LogLevelPanic, true},
		{"info", loglayer.LogLevelInfo, true},
		{"unknown", loglayer.LogLevelInfo, false},
	}
	for _, c := range cases {
		got, ok := loglayer.ParseLogLevel(c.in)
		if got != c.want || ok != c.wantOK {
			t.Errorf("ParseLogLevel(%q) = (%v, %v); want (%v, %v)", c.in, got, ok, c.want, c.wantOK)
		}
	}
}

func TestLogLevelString_TraceAndPanic(t *testing.T) {
	if loglayer.LogLevelTrace.String() != "trace" {
		t.Errorf("Trace.String() = %q, want \"trace\"", loglayer.LogLevelTrace.String())
	}
	if loglayer.LogLevelPanic.String() != "panic" {
		t.Errorf("Panic.String() = %q, want \"panic\"", loglayer.LogLevelPanic.String())
	}
}
