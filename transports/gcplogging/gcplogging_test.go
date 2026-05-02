package gcplogging

import (
	"errors"
	"testing"
	"time"

	"cloud.google.com/go/logging"

	"go.loglayer.dev/v2"
)

func TestSeverityFor(t *testing.T) {
	cases := []struct {
		in   loglayer.LogLevel
		want logging.Severity
	}{
		{loglayer.LogLevelTrace, logging.Debug},
		{loglayer.LogLevelDebug, logging.Debug},
		{loglayer.LogLevelInfo, logging.Info},
		{loglayer.LogLevelWarn, logging.Warning},
		{loglayer.LogLevelError, logging.Error},
		{loglayer.LogLevelFatal, logging.Critical},
		{loglayer.LogLevelPanic, logging.Alert},
		{loglayer.LogLevel(999), logging.Default},
	}
	for _, c := range cases {
		if got := severityFor(c.in); got != c.want {
			t.Errorf("severityFor(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestBuild_NilLoggerReturnsError(t *testing.T) {
	_, err := Build(Config{})
	if !errors.Is(err, ErrLoggerRequired) {
		t.Errorf("got %v, want ErrLoggerRequired", err)
	}
}

func TestNew_NilLoggerPanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic")
		}
		err, ok := r.(error)
		if !ok || !errors.Is(err, ErrLoggerRequired) {
			t.Errorf("panic value: got %v, want ErrLoggerRequired", r)
		}
	}()
	New(Config{})
}

// fakeLogger is a non-nil *logging.Logger so Build's nil-check passes.
// Only buildEntry is exercised below; SendToLogger / LogSync paths that
// would dereference Logger are not invoked.
var fakeLogger = &logging.Logger{}

func TestBuildEntry_PayloadShape(t *testing.T) {
	tr, err := Build(Config{Logger: fakeLogger})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	params := loglayer.TransportParams{
		LogLevel: loglayer.LogLevelInfo,
		Messages: []any{"served"},
		Data:     loglayer.Data{"requestId": "abc"},
		Metadata: loglayer.Metadata{"durationMs": 42},
	}
	entry := tr.buildEntry(params)

	if entry.Severity != logging.Info {
		t.Errorf("Severity: got %v, want Info", entry.Severity)
	}
	if entry.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}
	payload, ok := entry.Payload.(map[string]any)
	if !ok {
		t.Fatalf("Payload type: %T, want map[string]any", entry.Payload)
	}
	if payload["message"] != "served" {
		t.Errorf("message: got %v, want %q", payload["message"], "served")
	}
	if payload["requestId"] != "abc" {
		t.Errorf("requestId: got %v, want %q", payload["requestId"], "abc")
	}
	if payload["durationMs"] != 42 {
		t.Errorf("durationMs: got %v, want 42", payload["durationMs"])
	}
}

func TestBuildEntry_CustomMessageField(t *testing.T) {
	tr, err := Build(Config{Logger: fakeLogger, MessageField: "msg"})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	entry := tr.buildEntry(loglayer.TransportParams{
		LogLevel: loglayer.LogLevelInfo,
		Messages: []any{"hello"},
	})
	payload := entry.Payload.(map[string]any)
	if payload["msg"] != "hello" {
		t.Errorf("msg: got %v, want %q", payload["msg"], "hello")
	}
	if _, exists := payload["message"]; exists {
		t.Error("default 'message' key should not be set when MessageField is overridden")
	}
}

func TestBuildEntry_RootEntryFieldsCarryThrough(t *testing.T) {
	tr, err := Build(Config{
		Logger: fakeLogger,
		RootEntry: logging.Entry{
			Labels: map[string]string{"env": "prod"},
			Trace:  "projects/x/traces/abc",
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	entry := tr.buildEntry(loglayer.TransportParams{
		LogLevel: loglayer.LogLevelInfo,
		Messages: []any{"hi"},
	})
	if entry.Labels["env"] != "prod" {
		t.Errorf("Labels not carried through: %v", entry.Labels)
	}
	if entry.Trace != "projects/x/traces/abc" {
		t.Errorf("Trace not carried through: %v", entry.Trace)
	}
}

func TestBuildEntry_EntryFnRunsLast(t *testing.T) {
	called := false
	tr, err := Build(Config{
		Logger: fakeLogger,
		EntryFn: func(p loglayer.TransportParams, e *logging.Entry) {
			called = true
			e.Labels = map[string]string{"runtime": "via-EntryFn"}
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	entry := tr.buildEntry(loglayer.TransportParams{
		LogLevel: loglayer.LogLevelInfo,
		Messages: []any{"x"},
		Schema:   loglayer.Schema{},
	})
	if !called {
		t.Error("EntryFn was not invoked")
	}
	if entry.Labels["runtime"] != "via-EntryFn" {
		t.Errorf("Labels not set by EntryFn: %v", entry.Labels)
	}
}

func TestBuildEntry_NonMapMetadataNestsUnderKey(t *testing.T) {
	tr, err := Build(Config{Logger: fakeLogger})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	entry := tr.buildEntry(loglayer.TransportParams{
		LogLevel: loglayer.LogLevelInfo,
		Messages: []any{"x"},
		Metadata: 42, // scalar, not a map
	})
	payload := entry.Payload.(map[string]any)
	if payload["metadata"] != 42 {
		t.Errorf("scalar metadata should nest under 'metadata': got %v", payload["metadata"])
	}
}

func TestBuildEntry_TimestampIsRecent(t *testing.T) {
	tr, _ := Build(Config{Logger: fakeLogger})
	before := time.Now()
	entry := tr.buildEntry(loglayer.TransportParams{
		LogLevel: loglayer.LogLevelInfo,
		Messages: []any{"x"},
	})
	after := time.Now()
	if entry.Timestamp.Before(before) || entry.Timestamp.After(after) {
		t.Errorf("Timestamp out of expected window: %v not in [%v,%v]", entry.Timestamp, before, after)
	}
}
