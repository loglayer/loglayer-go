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
	log, lib := plugintest.Install(t, loglayer.NewDataHook("tag",
		func(_ loglayer.BeforeDataOutParams) loglayer.Data {
			return loglayer.Data{"tagged": true}
		}))
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
		func(captureFn func(error)) loglayer.Plugin {
			return &panicPlugin{
				id:      "boom",
				panicFn: func() { panic("kaboom") },
				onError: captureFn,
			}
		},
		func(log *loglayer.LogLayer) { log.Info("trigger") },
	)
	if rpe == nil {
		t.Fatal("expected RecoveredPanicError, got nil")
	}
	if rpe.Plugin == nil {
		t.Fatal("Plugin should be non-nil for plugin panic")
	}
	if rpe.Plugin.Hook != "OnBeforeDataOut" {
		t.Errorf("Plugin.Hook: got %q, want OnBeforeDataOut", rpe.Plugin.Hook)
	}
	if msg, _ := rpe.Value.(string); !strings.Contains(msg, "kaboom") {
		t.Errorf("Value: got %v, want 'kaboom'", rpe.Value)
	}
}

func TestAssertPanicRecovered_PanicWithErrorValue(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("sentinel")
	rpe := plugintest.AssertPanicRecovered(t,
		func(captureFn func(error)) loglayer.Plugin {
			return &panicMetadataPlugin{
				id:      "panic-err",
				panicFn: func() { panic(sentinel) },
				onError: captureFn,
			}
		},
		func(log *loglayer.LogLayer) { log.WithMetadata(loglayer.Metadata{"k": "v"}).Info("x") },
	)
	if !errors.Is(rpe, sentinel) {
		t.Errorf("errors.Is(rpe, sentinel) should be true; rpe=%v", rpe)
	}
}

// panicPlugin implements DataHook + ErrorReporter. Used to drive the
// AssertPanicRecovered helper: panicFn fires from OnBeforeDataOut, the
// framework recovers it, and onError captures the wrapped error.
type panicPlugin struct {
	id      string
	panicFn func()
	onError func(error)
}

func (p *panicPlugin) ID() string { return p.id }
func (p *panicPlugin) OnBeforeDataOut(loglayer.BeforeDataOutParams) loglayer.Data {
	p.panicFn()
	return nil
}
func (p *panicPlugin) OnError(err error) { p.onError(err) }

// panicMetadataPlugin is the same idea but for an OnMetadataCalled panic.
type panicMetadataPlugin struct {
	id      string
	panicFn func()
	onError func(error)
}

func (p *panicMetadataPlugin) ID() string { return p.id }
func (p *panicMetadataPlugin) OnMetadataCalled(any) any {
	p.panicFn()
	return nil
}
func (p *panicMetadataPlugin) OnError(err error) { p.onError(err) }

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
