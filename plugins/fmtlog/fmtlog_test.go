package fmtlog_test

import (
	"errors"
	"testing"

	"go.loglayer.dev"
	"go.loglayer.dev/plugins/fmtlog"
	"go.loglayer.dev/transport"
	lltest "go.loglayer.dev/transports/testing"
)

func newLogger(t *testing.T) (*loglayer.LogLayer, *lltest.TestLoggingLibrary) {
	t.Helper()
	lib := &lltest.TestLoggingLibrary{}
	tr := lltest.New(lltest.Config{
		BaseConfig: transport.BaseConfig{ID: "t"},
		Library:    lib,
	})
	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})
	log.AddPlugin(fmtlog.New())
	return log, lib
}

func firstMessage(line *lltest.LogLine) string {
	s, _ := line.Messages[0].(string)
	return s
}

func TestFormatString(t *testing.T) {
	log, lib := newLogger(t)
	log.Info("user %d signed in", 42)
	if got := firstMessage(lib.PopLine()); got != "user 42 signed in" {
		t.Errorf("got %q", got)
	}
}

func TestFormatString_AcrossLevels(t *testing.T) {
	cases := []struct {
		name string
		emit func(*loglayer.LogLayer)
		want string
	}{
		{"Debug", func(l *loglayer.LogLayer) { l.Debug("n=%d", 7) }, "n=7"},
		{"Info", func(l *loglayer.LogLayer) { l.Info("user %s", "alice") }, "user alice"},
		{"Warn", func(l *loglayer.LogLayer) { l.Warn("%d/%d", 3, 4) }, "3/4"},
		{"Error", func(l *loglayer.LogLayer) { l.Error("boom: %v", errors.New("x")) }, "boom: x"},
		{"Fatal", func(l *loglayer.LogLayer) { l.Fatal("halt %d", 1) }, "halt 1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			log, lib := newLogger(t)
			c.emit(log)
			if got := firstMessage(lib.PopLine()); got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

// TestSingleArgUntouched confirms one-arg calls bypass the plugin: a
// literal "%" stays as-is, no Sprintf is applied.
func TestSingleArgUntouched(t *testing.T) {
	log, lib := newLogger(t)
	log.Info("100% complete")
	if got := firstMessage(lib.PopLine()); got != "100% complete" {
		t.Errorf("got %q", got)
	}
}

// TestNonStringFirstArgUntouched confirms calls whose first arg isn't
// a string bypass the plugin: the messages slice passes through
// untouched (the transport sees both elements, will space-join via
// JoinMessages at render time).
func TestNonStringFirstArgUntouched(t *testing.T) {
	log, lib := newLogger(t)
	log.Info(42, "extra")
	line := lib.PopLine()
	if len(line.Messages) != 2 {
		t.Fatalf("messages should pass through unchanged, got %v", line.Messages)
	}
}

// TestComposesWithBuilder shows the format-string semantics flow
// through the builder chain unchanged.
func TestComposesWithBuilder(t *testing.T) {
	log, lib := newLogger(t)
	log.WithMetadata(loglayer.Metadata{"k": "v"}).
		WithError(errors.New("timeout")).
		Error("request %s failed", "abc-123")
	line := lib.PopLine()
	if got := firstMessage(line); got != "request abc-123 failed" {
		t.Errorf("message: got %q", got)
	}
	m := line.Metadata.(loglayer.Metadata)
	if m["k"] != "v" {
		t.Errorf("metadata not preserved: got %v", m)
	}
}

// TestWithoutPlugin pins the framework's default behavior when fmtlog
// isn't registered: messages pass through untouched as a 2-element
// slice. JoinMessages would space-join them at render time.
func TestWithoutPlugin(t *testing.T) {
	lib := &lltest.TestLoggingLibrary{}
	tr := lltest.New(lltest.Config{
		BaseConfig: transport.BaseConfig{ID: "t"},
		Library:    lib,
	})
	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})
	log.Info("user %d", 42)
	line := lib.PopLine()
	if len(line.Messages) != 2 {
		t.Errorf("messages should pass through unchanged, got %v", line.Messages)
	}
}
