package loglayer_test

import (
	"errors"
	"testing"

	"go.loglayer.dev"
)

func TestNewMockReturnsUsableLogger(t *testing.T) {
	log := loglayer.NewMock()
	if log == nil {
		t.Fatal("NewMock returned nil")
	}
	// All level methods should be safe to call.
	log.Trace("trace")
	log.Debug("debug")
	log.Info("info")
	log.Warn("warn")
	log.Error("error")
	log.Fatal("fatal") // does not exit
}

func TestNewMockBuilderChain(t *testing.T) {
	log := loglayer.NewMock()
	log.WithMetadata(map[string]any{"k": "v"}).
		WithError(errors.New("boom")).
		Error("failed")
	// Reaching this line without panic is success.
}

func TestNewMockHonorsContextAndLevels(t *testing.T) {
	log := loglayer.NewMock()
	log = log.WithFields(loglayer.Fields{"requestId": "abc"})
	if log.GetFields()["requestId"] != "abc" {
		t.Errorf("mock should still track context: %v", log.GetFields())
	}
	log.SetLevel(loglayer.LogLevelWarn)
	if log.IsLevelEnabled(loglayer.LogLevelInfo) {
		t.Error("mock should still respect SetLevel: info should be disabled at warn threshold")
	}
}

func TestNewMockChildIsIndependent(t *testing.T) {
	log := loglayer.NewMock()
	log = log.WithFields(loglayer.Fields{"shared": "v"})
	child := log.Child()
	child = child.WithFields(loglayer.Fields{"only_child": "x"})

	if log.GetFields()["only_child"] != nil {
		t.Errorf("parent should not see child context: %v", log.GetFields())
	}
	if child.GetFields()["shared"] != "v" {
		t.Errorf("child should inherit parent context: %v", child.GetFields())
	}
}

func TestNewMockGetLoggerInstanceIsNil(t *testing.T) {
	log := loglayer.NewMock()
	if got := log.GetLoggerInstance("mock"); got != nil {
		t.Errorf("mock transport should expose nil underlying logger, got %T", got)
	}
}

func TestNewMockMetadataOnlyAndErrorOnly(t *testing.T) {
	log := loglayer.NewMock()
	log.MetadataOnly(map[string]any{"k": "v"})
	log.ErrorOnly(errors.New("only"))
	// No assertions — just verify these are no-op-safe with the discard transport.
}
