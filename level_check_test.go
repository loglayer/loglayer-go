package loglayer_test

import (
	"testing"

	"go.loglayer.dev/v3"
)

// LevelFiltering with Config.Level set at construction: the core's own
// level state must filter before any transport sees the entry, so a
// non-filtering transport cannot leak below-threshold entries. Pins the
// contract suite's LevelFiltering case (transporttest.RunContract).
func TestConfigLevel_FiltersBeforeDispatch(t *testing.T) {
	log, lib := setupWithConfig(t, loglayer.Config{Level: loglayer.LogLevelError})
	log.Warn("dropped")
	if lib.Len() != 0 {
		t.Fatalf("warn should be filtered by core level state, got %d lines", lib.Len())
	}
	log.Error("passes")
	line := lib.PopLine()
	if line == nil {
		t.Fatal("expected error line to be emitted")
	}
}
