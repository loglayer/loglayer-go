package lltest_test

import (
	"testing"

	"go.loglayer.dev/v2"
	"go.loglayer.dev/v2/internal/lltest"
	"go.loglayer.dev/v2/transport"
)

func newLogger() (*loglayer.LogLayer, *lltest.TestLoggingLibrary) {
	lib := &lltest.TestLoggingLibrary{}
	t := lltest.New(lltest.Config{
		BaseConfig: transport.BaseConfig{ID: "test"},
		Library:    lib,
	})
	return loglayer.New(loglayer.Config{DisableFatalExit: true, Transport: t}), lib
}

func TestTestTransportCaptures(t *testing.T) {
	log, lib := newLogger()
	log.Info("captured")
	if lib.Len() != 1 {
		t.Fatalf("expected 1 line, got %d", lib.Len())
	}
}

func TestTestLibraryGetLastLine(t *testing.T) {
	log, lib := newLogger()
	log.Info("first")
	log.Warn("second")
	line := lib.GetLastLine()
	if line == nil {
		t.Fatal("expected a line")
	}
	if line.Level != loglayer.LogLevelWarn {
		t.Errorf("GetLastLine: got %s, want warn", line.Level)
	}
	if lib.Len() != 2 {
		t.Error("GetLastLine should not remove the line")
	}
}

func TestTestLibraryPopLine(t *testing.T) {
	log, lib := newLogger()
	log.Info("pop me")
	line := lib.PopLine()
	if line == nil {
		t.Fatal("expected a line")
	}
	if lib.Len() != 0 {
		t.Error("PopLine should remove the line")
	}
}

func TestTestLibraryClearLines(t *testing.T) {
	log, lib := newLogger()
	log.Info("a")
	log.Info("b")
	lib.ClearLines()
	if lib.Len() != 0 {
		t.Errorf("ClearLines: expected 0 lines, got %d", lib.Len())
	}
}

func TestTestLibraryNilOnEmpty(t *testing.T) {
	lib := &lltest.TestLoggingLibrary{}
	if lib.GetLastLine() != nil {
		t.Error("GetLastLine on empty should return nil")
	}
	if lib.PopLine() != nil {
		t.Error("PopLine on empty should return nil")
	}
}

func TestTestTransportFieldsExposed(t *testing.T) {
	log, lib := newLogger()
	log = log.WithFields(loglayer.Fields{"ctx_key": "ctx_val"})
	log.WithMetadata(map[string]any{"meta_key": "meta_val"}).Info("msg")
	line := lib.PopLine()
	if line == nil {
		t.Fatal("expected a line")
	}
	if len(line.Messages) != 1 || line.Messages[0] != "msg" {
		t.Errorf("Messages: got %v", line.Messages)
	}
	if len(line.Data) == 0 {
		t.Error("Data should be populated when context is set")
	}
	if line.Data["ctx_key"] != "ctx_val" {
		t.Errorf("Data ctx_key: got %v", line.Data["ctx_key"])
	}
	m, ok := line.Metadata.(map[string]any)
	if !ok {
		t.Fatalf("Metadata: expected map, got %T", line.Metadata)
	}
	if m["meta_key"] != "meta_val" {
		t.Errorf("Metadata meta_key: got %v", m["meta_key"])
	}
}

func TestAutoCreatedLibrary(t *testing.T) {
	trans := lltest.New(lltest.Config{BaseConfig: transport.BaseConfig{ID: "auto"}})
	log := loglayer.New(loglayer.Config{DisableFatalExit: true, Transport: trans})
	log.Info("auto lib")
	if trans.Library.Len() != 1 {
		t.Errorf("auto-created library should capture logs, got %d", trans.Library.Len())
	}
}
