// Package plugintest provides helpers for testing LogLayer plugins.
//
// Use [Install] to wire a plugin into a fresh logger backed by the in-memory
// testing transport, then drive scenarios through the logger and assert
// against the captured [transports/testing.LogLine] entries.
//
// Use [AssertNoMutation] to verify a plugin's input-side hook
// (OnFieldsCalled / OnMetadataCalled) doesn't mutate caller-owned input,
// which is the contract the framework expects.
//
// Use [AssertPanicRecovered] to verify the framework recovers a panicking
// hook and surfaces it via Plugin.OnError as a *loglayer.RecoveredPanicError.
package plugintest

import (
	"errors"
	"reflect"
	"testing"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
	lltest "go.loglayer.dev/transports/testing"
	"go.loglayer.dev/utils/maputil"
)

// Install builds a fresh *loglayer.LogLayer with p installed and the
// in-memory testing transport attached. Returns the logger and the capture
// library; drive arbitrary scenarios through the logger and pop entries from
// the library.
//
//	log, lib := plugintest.Install(t, myplugin.New(...))
//	log.WithMetadata(loglayer.Metadata{"k": "v"}).Info("event")
//	line := lib.PopLine()
//	if line.Data["enriched"] != "yes" { ... }
//
// DisableFatalExit is set so log.Fatal() doesn't terminate the test process.
func Install(t testing.TB, p loglayer.Plugin) (*loglayer.LogLayer, *lltest.TestLoggingLibrary) {
	t.Helper()
	lib := &lltest.TestLoggingLibrary{}
	tr := lltest.New(lltest.Config{
		BaseConfig: transport.BaseConfig{ID: "plugintest"},
		Library:    lib,
	})
	log := loglayer.New(loglayer.Config{
		Transport:        tr,
		DisableFatalExit: true,
		Plugins:          []loglayer.Plugin{p},
	})
	return log, lib
}

// AssertNoMutation verifies that running an input-side hook does not mutate
// the caller-owned input. It deep-clones the input, runs the hook, and
// reports a test failure if the original differs from the clone afterward.
//
// Use it for OnFieldsCalled and OnMetadataCalled hooks. The hook function is
// passed in directly rather than via a Plugin, so the test focuses on hook
// behavior without needing to install or emit through a logger.
//
//	plugintest.AssertNoMutation(t, redactPlugin.OnMetadataCalled,
//	    map[string]any{"password": "hunter2", "user": "alice"})
func AssertNoMutation[T any](t testing.TB, hook func(T) T, input T) {
	t.Helper()
	if hook == nil {
		t.Fatal("AssertNoMutation: hook is nil")
	}
	cloner := &maputil.Cloner{}
	snapshot, ok := cloner.Clone(input).(T)
	if !ok {
		t.Fatalf("AssertNoMutation: snapshot type mismatch (cloner returned %T, want %T)",
			cloner.Clone(input), input)
		return
	}
	_ = hook(input)
	if !reflect.DeepEqual(input, snapshot) {
		t.Errorf("hook mutated caller-owned input.\n  before: %#v\n  after:  %#v", snapshot, input)
	}
}

// AssertPanicRecovered installs plugin into a fresh logger, drives the
// supplied emit callback, and asserts the framework caught a hook panic and
// forwarded a *loglayer.RecoveredPanicError. The plugin's OnError is
// replaced with a capturing closure for the duration of the call.
//
//	rpe := plugintest.AssertPanicRecovered(t,
//	    loglayer.Plugin{
//	        ID: "boom",
//	        OnBeforeDataOut: func(loglayer.BeforeDataOutParams) loglayer.Data {
//	            panic("boom")
//	        },
//	    },
//	    func(log *loglayer.LogLayer) { log.Info("trigger") },
//	)
//
// Returns the captured *RecoveredPanicError so callers can assert on
// Hook / Value if they want.
func AssertPanicRecovered(
	t testing.TB,
	plugin loglayer.Plugin,
	emit func(*loglayer.LogLayer),
) *loglayer.RecoveredPanicError {
	t.Helper()
	var captured error
	plugin.OnError = func(err error) { captured = err }
	log, _ := Install(t, plugin)
	emit(log)
	if captured == nil {
		t.Fatal("expected a recovered panic to reach OnError; got nil")
		return nil
	}
	var rpe *loglayer.RecoveredPanicError
	if !errors.As(captured, &rpe) {
		t.Errorf("OnError got %T (%v); want *loglayer.RecoveredPanicError", captured, captured)
		return nil
	}
	return rpe
}
