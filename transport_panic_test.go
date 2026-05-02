package loglayer_test

import (
	"errors"
	"sync"
	"testing"

	"go.loglayer.dev/v2"
	"go.loglayer.dev/v2/internal/lltest"
	"go.loglayer.dev/v2/transport"
)

// panickingTransport panics on every SendToLogger.
type panickingTransport struct {
	transport.BaseTransport
	value any
}

func newPanickingTransport(id string, value any) *panickingTransport {
	return &panickingTransport{
		BaseTransport: transport.NewBaseTransport(transport.BaseConfig{ID: id}),
		value:         value,
	}
}

func (*panickingTransport) GetLoggerInstance() any { return nil }

func (p *panickingTransport) SendToLogger(_ loglayer.TransportParams) {
	panic(p.value)
}

// Default behavior: a panicking transport propagates up through the
// emission call. Matches Go logging convention (zerolog/zap/slog).
func TestTransportPanic_PropagatesByDefault(t *testing.T) {
	log := loglayer.New(loglayer.Config{
		Transport:        newPanickingTransport("boom", "bad transport"),
		DisableFatalExit: true,
	})

	var got any
	func() {
		defer func() { got = recover() }()
		log.Info("hi")
	}()

	if got == nil {
		t.Fatal("a panicking transport should propagate up by default; got no panic")
	}
	if got != "bad transport" {
		t.Errorf("panic value: got %v, want \"bad transport\"", got)
	}
}

// With OnTransportPanic set, the dispatch loop recovers and reports.
// The user's emission call returns normally; the panic surfaces via
// the callback as a *RecoveredPanicError matching the plugin shape.
func TestTransportPanic_RecoveredViaCallback(t *testing.T) {
	var (
		mu      sync.Mutex
		reports []*loglayer.RecoveredPanicError
	)
	log := loglayer.New(loglayer.Config{
		Transport:        newPanickingTransport("boom", "bad transport"),
		DisableFatalExit: true,
		OnTransportPanic: func(err *loglayer.RecoveredPanicError) {
			mu.Lock()
			defer mu.Unlock()
			reports = append(reports, err)
		},
	})

	// Should NOT panic; the dispatch loop recovers.
	log.Info("hi")

	mu.Lock()
	defer mu.Unlock()
	if len(reports) != 1 {
		t.Fatalf("expected 1 incident reported, got %d", len(reports))
	}
	rpe := reports[0]
	if rpe.Kind != loglayer.PanicKindTransport {
		t.Errorf("Kind: got %q, want %q", rpe.Kind, loglayer.PanicKindTransport)
	}
	if rpe.ID != "boom" {
		t.Errorf("ID (transport id): got %q, want \"boom\"", rpe.ID)
	}
	if rpe.Plugin != nil {
		t.Errorf("Plugin should be nil for transport panics, got %+v", rpe.Plugin)
	}
	if rpe.Value != "bad transport" {
		t.Errorf("Value: got %v, want \"bad transport\"", rpe.Value)
	}
}

// A panic in one transport must not stop dispatch to the others.
// Pair a panicking transport with a working one and confirm both
// the report and the surviving emission.
func TestTransportPanic_DoesNotSuppressOtherTransports(t *testing.T) {
	lib := &lltest.TestLoggingLibrary{}
	working := lltest.New(lltest.Config{
		BaseConfig: transport.BaseConfig{ID: "working"},
		Library:    lib,
	})
	bad := newPanickingTransport("bad", errors.New("oh no"))

	var reported int
	log := loglayer.New(loglayer.Config{
		Transports:       []loglayer.Transport{bad, working},
		DisableFatalExit: true,
		OnTransportPanic: func(*loglayer.RecoveredPanicError) { reported++ },
	})

	log.Info("survives")

	if reported != 1 {
		t.Errorf("expected one panic report, got %d", reported)
	}
	if lib.Len() != 1 {
		t.Errorf("the working transport should have received the entry: got %d", lib.Len())
	}
}

// A panic in OnTransportPanic itself is recovered (and dropped) so a
// buggy reporter can't take down the dispatch loop.
func TestTransportPanic_HandlerPanicSwallowed(t *testing.T) {
	lib := &lltest.TestLoggingLibrary{}
	working := lltest.New(lltest.Config{
		BaseConfig: transport.BaseConfig{ID: "working"},
		Library:    lib,
	})
	bad := newPanickingTransport("bad", "transport boom")

	log := loglayer.New(loglayer.Config{
		Transports:       []loglayer.Transport{bad, working},
		DisableFatalExit: true,
		OnTransportPanic: func(*loglayer.RecoveredPanicError) {
			panic("handler boom")
		},
	})

	// Outer log call must not panic. The panicking transport's panic
	// is recovered; the handler's panic is also recovered (dropped).
	log.Info("hi")

	if lib.Len() != 1 {
		t.Errorf("working transport should still receive the entry: got %d", lib.Len())
	}
}

// The same callback can absorb panics from either plugin hooks or
// transports because both surface a *RecoveredPanicError. Kind
// distinguishes the source. This test wires one observer into both
// pipelines and confirms it sees both events with the right Kind values.
func TestTransportPanic_UnifiedShapeWithPluginPanic(t *testing.T) {
	var (
		mu     sync.Mutex
		events []*loglayer.RecoveredPanicError
	)
	record := func(err *loglayer.RecoveredPanicError) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, err)
	}

	lib := &lltest.TestLoggingLibrary{}
	working := lltest.New(lltest.Config{
		BaseConfig: transport.BaseConfig{ID: "working"},
		Library:    lib,
	})
	log := loglayer.New(loglayer.Config{
		Transports:       []loglayer.Transport{newPanickingTransport("bad", "boom"), working},
		DisableFatalExit: true,
		OnTransportPanic: record,
	})

	// Plugin panic: route through a plugin with an ErrorReporter that
	// forwards to the same record() function so both Kinds end up in
	// the same observer.
	log.AddPlugin(loglayer.WithErrorReporter(
		loglayer.NewDataHook("panicker",
			func(loglayer.BeforeDataOutParams) loglayer.Data { panic("plugin boom") },
		),
		func(err error) {
			if rpe, ok := err.(*loglayer.RecoveredPanicError); ok {
				record(rpe)
			}
		},
	))

	log.Info("triggers both panics") // bad transport panics; plugin panics

	mu.Lock()
	defer mu.Unlock()
	var sawPlugin, sawTransport bool
	for _, e := range events {
		switch e.Kind {
		case loglayer.PanicKindPlugin:
			sawPlugin = true
			// Plugin panics carry both ID (which plugin) and Plugin
			// details (which hook method). Confirm both are populated.
			if e.ID != "panicker" {
				t.Errorf("plugin panic ID: got %q, want \"panicker\"", e.ID)
			}
			if e.Plugin == nil {
				t.Error("plugin panic should have non-nil Plugin details")
			} else if e.Plugin.Hook != "OnBeforeDataOut" {
				t.Errorf("plugin panic Plugin.Hook: got %q, want \"OnBeforeDataOut\"", e.Plugin.Hook)
			}
		case loglayer.PanicKindTransport:
			sawTransport = true
			// Transport panics have ID (transport ID) but no Plugin
			// details (Plugin is nil — typed absence rather than empty
			// string).
			if e.ID != "bad" {
				t.Errorf("transport panic ID: got %q, want \"bad\"", e.ID)
			}
			if e.Plugin != nil {
				t.Errorf("transport panic Plugin should be nil, got %+v", e.Plugin)
			}
		default:
			t.Errorf("unexpected Kind: %q", e.Kind)
		}
	}
	if !sawPlugin || !sawTransport {
		t.Errorf("expected to observe both panic kinds; got plugin=%v transport=%v", sawPlugin, sawTransport)
	}
}

// Default behavior with default-nil handler: hot path is a direct
// SendToLogger call (no recover overhead). This test pairs with the
// benchmark; the assertion here is just that the no-handler config
// works at all.
func TestTransportPanic_NoHandler_HotPathIsDirect(t *testing.T) {
	lib := &lltest.TestLoggingLibrary{}
	log := loglayer.New(loglayer.Config{
		Transport:        lltest.New(lltest.Config{Library: lib}),
		DisableFatalExit: true,
	})
	log.Info("hi")
	if lib.Len() != 1 {
		t.Errorf("default config should dispatch: got %d", lib.Len())
	}
}
