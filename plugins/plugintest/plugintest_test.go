package plugintest_test

import (
	"errors"
	"strings"
	"testing"

	"go.loglayer.dev"
	"go.loglayer.dev/plugins/plugintest"
)

func TestInstall_PluginRunsAndCaptures(t *testing.T) {
	t.Parallel()
	log, lib := plugintest.Install(t, loglayer.Plugin{
		ID: "tag",
		OnBeforeDataOut: func(_ loglayer.BeforeDataOutParams) loglayer.Data {
			return loglayer.Data{"tagged": true}
		},
	})
	log.Info("hello")
	line := lib.PopLine()
	if line.Data["tagged"] != true {
		t.Errorf("plugin output not in captured line: %v", line.Data)
	}
}

func TestAssertNoMutation_PassesForCleanHook(t *testing.T) {
	t.Parallel()
	hook := func(m any) any {
		// Builds a fresh map; doesn't touch the input.
		if in, ok := m.(map[string]any); ok {
			out := make(map[string]any, len(in))
			for k, v := range in {
				out[k] = v
			}
			out["added"] = true
			return out
		}
		return m
	}
	plugintest.AssertNoMutation[any](t, hook, map[string]any{"k": "v", "n": 1})
}

func TestAssertNoMutation_FailsForMutatingHook(t *testing.T) {
	t.Parallel()
	mt := &mockT{TB: t}
	hook := func(m any) any {
		if in, ok := m.(map[string]any); ok {
			in["mutated"] = true // ❌ mutates the caller's map
		}
		return m
	}
	plugintest.AssertNoMutation[any](mt, hook, map[string]any{"k": "v"})
	if !mt.failed {
		t.Error("AssertNoMutation should have flagged the mutating hook")
	}
}

func TestAssertPanicRecovered_PluginPanics(t *testing.T) {
	t.Parallel()
	rpe := plugintest.AssertPanicRecovered(t,
		loglayer.Plugin{
			ID: "boom",
			OnBeforeDataOut: func(loglayer.BeforeDataOutParams) loglayer.Data {
				panic("kaboom")
			},
		},
		func(log *loglayer.LogLayer) { log.Info("trigger") },
	)
	if rpe == nil {
		t.Fatal("expected RecoveredPanicError, got nil")
	}
	if rpe.Hook != "OnBeforeDataOut" {
		t.Errorf("Hook: got %q, want OnBeforeDataOut", rpe.Hook)
	}
	if msg, _ := rpe.Value.(string); !strings.Contains(msg, "kaboom") {
		t.Errorf("Value: got %v, want 'kaboom'", rpe.Value)
	}
}

func TestAssertPanicRecovered_PanicWithErrorValue(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("sentinel")
	rpe := plugintest.AssertPanicRecovered(t,
		loglayer.Plugin{
			ID:               "panic-err",
			OnMetadataCalled: func(any) any { panic(sentinel) },
		},
		func(log *loglayer.LogLayer) { log.WithMetadata(loglayer.Metadata{"k": "v"}).Info("x") },
	)
	if !errors.Is(rpe, sentinel) {
		t.Errorf("errors.Is(rpe, sentinel) should be true; rpe=%v", rpe)
	}
}

// mockT is a minimal testing.TB that records whether Errorf or Fatalf was
// called, used to verify AssertNoMutation's failure path.
type mockT struct {
	testing.TB
	failed bool
}

func (m *mockT) Errorf(string, ...any) { m.failed = true }
func (m *mockT) Fatalf(string, ...any) { m.failed = true }
func (m *mockT) Fatal(...any)          { m.failed = true }
func (m *mockT) Helper()               {}
