package loglayer_test

import (
	"errors"
	"testing"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
	lltest "go.loglayer.dev/transports/testing"
)

// dispatch_edge_test.go covers edge cases of the processLog dispatch path
// that the integration-style tests don't isolate cleanly. Pulled out so
// the cases are findable when something regresses in the hot path.

// twoTransport returns a logger with two test transports identified as
// "a" and "b", and the libraries each transport drains into.
func twoTransport(t *testing.T) (*loglayer.LogLayer, []*lltest.TestLoggingLibrary) {
	t.Helper()
	libs := []*lltest.TestLoggingLibrary{{}, {}}
	transports := []loglayer.Transport{
		lltest.New(lltest.Config{BaseConfig: transport.BaseConfig{ID: "a"}, Library: libs[0]}),
		lltest.New(lltest.Config{BaseConfig: transport.BaseConfig{ID: "b"}, Library: libs[1]}),
	}
	log := loglayer.New(loglayer.Config{
		Transports:       transports,
		DisableFatalExit: true,
	})
	return log, libs
}

// When a plugin's ShouldSend returns false for every transport, the
// entry is dropped everywhere. This is the "kill switch" semantic and
// must not silently leak entries into a default sink.
func TestDispatchEdge_AllShouldSendFalse_DropsEverywhere(t *testing.T) {
	log, libs := twoTransport(t)
	log.AddPlugin(loglayer.NewSendGate("deny-all", func(p loglayer.ShouldSendParams) bool { return false }))

	log.Info("dropped")

	if libs[0].Len() != 0 || libs[1].Len() != 0 {
		t.Errorf("ShouldSend=false on every transport should drop everywhere; got a=%d b=%d",
			libs[0].Len(), libs[1].Len())
	}
}

// A group whose listed transports are all Disabled should not deliver
// the entry. If the group is the only tag on the entry, the entry is
// dropped (it does not fall through to UngroupedRouting; "group with
// disabled transports" is "explicitly off," distinct from "no tag").
func TestDispatchEdge_DisabledGroupDoesNotFallThrough(t *testing.T) {
	libs := []*lltest.TestLoggingLibrary{{}, {}}
	log := loglayer.New(loglayer.Config{
		Transports: []loglayer.Transport{
			lltest.New(lltest.Config{BaseConfig: transport.BaseConfig{ID: "a"}, Library: libs[0]}),
			lltest.New(lltest.Config{BaseConfig: transport.BaseConfig{ID: "b"}, Library: libs[1]}),
		},
		Groups: map[string]loglayer.LogGroup{
			"silenced": {Transports: []string{"a", "b"}, Disabled: true},
		},
		// Default UngroupedToAll would route untagged entries to both
		// transports; this test tags the entry, so this only matters
		// to prove the disabled-group path doesn't accidentally fall
		// back into the ungrouped routing.
		UngroupedRouting: loglayer.UngroupedRouting{Mode: loglayer.UngroupedToAll},
		DisableFatalExit: true,
	})

	log.WithGroup("silenced").Info("ignored")

	if libs[0].Len() != 0 || libs[1].Len() != 0 {
		t.Errorf("disabled group should not deliver; got a=%d b=%d", libs[0].Len(), libs[1].Len())
	}
}

// A group whose listed transports are all valid but the tag references
// an *undefined* group falls through to UngroupedRouting. This is the
// counterpart to the disabled-group case: undefined ≠ disabled.
func TestDispatchEdge_UndefinedGroupFallsThrough(t *testing.T) {
	libs := []*lltest.TestLoggingLibrary{{}, {}}
	log := loglayer.New(loglayer.Config{
		Transports: []loglayer.Transport{
			lltest.New(lltest.Config{BaseConfig: transport.BaseConfig{ID: "a"}, Library: libs[0]}),
			lltest.New(lltest.Config{BaseConfig: transport.BaseConfig{ID: "b"}, Library: libs[1]}),
		},
		// Note: no group named "ghost" defined.
		Groups: map[string]loglayer.LogGroup{
			"defined": {Transports: []string{"a"}},
		},
		UngroupedRouting: loglayer.UngroupedRouting{
			Mode:       loglayer.UngroupedToTransports,
			Transports: []string{"b"},
		},
		DisableFatalExit: true,
	})

	log.WithGroup("ghost").Info("only-tagged-with-undefined")

	if libs[0].Len() != 0 {
		t.Errorf("undefined-group entry should not reach 'a' (which only matches the defined group)")
	}
	if libs[1].Len() != 1 {
		t.Errorf("undefined-group entry should fall through to UngroupedRouting -> 'b'; got %d", libs[1].Len())
	}
}

// A custom ErrorSerializer returning nil drops the err key entirely.
// The dispatch path must accept this and not crash on a nil map.
func TestDispatchEdge_ErrorSerializerReturningNil(t *testing.T) {
	lib := &lltest.TestLoggingLibrary{}
	tr := lltest.New(lltest.Config{BaseConfig: transport.BaseConfig{ID: "test"}, Library: lib})
	log := loglayer.New(loglayer.Config{
		Transport:        tr,
		DisableFatalExit: true,
		ErrorSerializer:  func(err error) map[string]any { return nil },
	})

	log.WithError(errors.New("boom")).Error("failed")
	line := lib.PopLine()
	if line == nil {
		t.Fatal("entry should still emit even when ErrorSerializer returns nil")
	}
	if _, has := line.Data["err"]; has {
		t.Errorf("err key should be absent when ErrorSerializer returns nil: %v", line.Data)
	}
}

// A custom ErrorSerializer returning an empty map adds an empty err
// object. The dispatch path must accept this without crashing.
func TestDispatchEdge_ErrorSerializerReturningEmptyMap(t *testing.T) {
	lib := &lltest.TestLoggingLibrary{}
	tr := lltest.New(lltest.Config{BaseConfig: transport.BaseConfig{ID: "test"}, Library: lib})
	log := loglayer.New(loglayer.Config{
		Transport:        tr,
		DisableFatalExit: true,
		ErrorSerializer:  func(err error) map[string]any { return map[string]any{} },
	})

	log.WithError(errors.New("boom")).Error("failed")
	line := lib.PopLine()
	errVal, has := line.Data["err"]
	if !has {
		t.Fatalf("err key should be present even when ErrorSerializer returns empty: %v", line.Data)
	}
	m, ok := errVal.(map[string]any)
	if !ok {
		t.Fatalf("err should be a map[string]any, got %T", errVal)
	}
	if len(m) != 0 {
		t.Errorf("err map should be empty: %v", m)
	}
}

// Multiple TransformLogLevel plugins: per the documented contract, the
// last plugin returning ok=true wins. Earlier ok=true returns are
// overridden.
func TestDispatchEdge_TransformLogLevel_LastTrueWins(t *testing.T) {
	log, lib := twoTransportSingleLib(t)

	// Three plugins:
	//  - A always returns Warn (ok=true).
	//  - B returns Error (ok=true). Should override A.
	//  - C returns nothing (ok=false). Leaves Error in place.
	log.AddPlugin(loglayer.NewLevelHook("first-warn", func(p loglayer.TransformLogLevelParams) (loglayer.LogLevel, bool) {
		return loglayer.LogLevelWarn, true
	}))
	log.AddPlugin(loglayer.NewLevelHook("second-error", func(p loglayer.TransformLogLevelParams) (loglayer.LogLevel, bool) {
		return loglayer.LogLevelError, true
	}))
	log.AddPlugin(loglayer.NewLevelHook("third-passthrough", func(p loglayer.TransformLogLevelParams) (loglayer.LogLevel, bool) {
		return 0, false
	}))

	log.Info("input-info")

	line := lib.PopLine()
	if line.Level != loglayer.LogLevelError {
		t.Errorf("last ok=true plugin should win; got %v want Error", line.Level)
	}
}

// twoTransportSingleLib returns a logger with one test transport so a
// caller asserting on "the level the transport saw" doesn't have to
// pick which of two libraries to read from.
func twoTransportSingleLib(t *testing.T) (*loglayer.LogLayer, *lltest.TestLoggingLibrary) {
	t.Helper()
	lib := &lltest.TestLoggingLibrary{}
	tr := lltest.New(lltest.Config{BaseConfig: transport.BaseConfig{ID: "test"}, Library: lib})
	log := loglayer.New(loglayer.Config{
		Transport:        tr,
		DisableFatalExit: true,
	})
	return log, lib
}
