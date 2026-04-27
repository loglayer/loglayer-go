package loglayer_test

import (
	"testing"

	"go.loglayer.dev"
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
