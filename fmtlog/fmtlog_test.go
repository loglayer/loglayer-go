package fmtlog_test

import (
	"errors"
	"testing"

	"go.loglayer.dev"
	"go.loglayer.dev/fmtlog"
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
	return loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true}), lib
}

func TestLevels(t *testing.T) {
	cases := []struct {
		name  string
		emit  func(*loglayer.LogLayer)
		level loglayer.LogLevel
		want  string
	}{
		{"Debugf", func(l *loglayer.LogLayer) { fmtlog.Debugf(l, "n=%d", 7) }, loglayer.LogLevelDebug, "n=7"},
		{"Infof", func(l *loglayer.LogLayer) { fmtlog.Infof(l, "user %s", "alice") }, loglayer.LogLevelInfo, "user alice"},
		{"Warnf", func(l *loglayer.LogLayer) { fmtlog.Warnf(l, "%d/%d", 3, 4) }, loglayer.LogLevelWarn, "3/4"},
		{"Errorf", func(l *loglayer.LogLayer) { fmtlog.Errorf(l, "boom: %v", errors.New("x")) }, loglayer.LogLevelError, "boom: x"},
		{"Fatalf", func(l *loglayer.LogLayer) { fmtlog.Fatalf(l, "halt %d", 1) }, loglayer.LogLevelFatal, "halt 1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			log, lib := newLogger(t)
			c.emit(log)
			line := lib.PopLine()
			if line == nil {
				t.Fatal("expected a line")
			}
			if line.Level != c.level {
				t.Errorf("level: got %v, want %v", line.Level, c.level)
			}
			if got, _ := line.Messages[0].(string); got != c.want {
				t.Errorf("message: got %q, want %q", got, c.want)
			}
		})
	}
}

// TestComposesWithBuilder shows the documented pattern: the helper
// stays out of the builder chain; users format inline when combining
// with WithMetadata/WithError.
func TestComposesWithBuilder(t *testing.T) {
	log, lib := newLogger(t)
	log.WithMetadata(loglayer.Metadata{"k": "v"}).
		Info("user 7 signed in") // pre-formatted inline
	line := lib.PopLine()
	if got, _ := line.Messages[0].(string); got != "user 7 signed in" {
		t.Errorf("message: got %q", got)
	}
	m := line.Metadata.(loglayer.Metadata)
	if m["k"] != "v" {
		t.Errorf("metadata not preserved: got %v", m)
	}
}
