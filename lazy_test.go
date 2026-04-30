package loglayer_test

import (
	"sync/atomic"
	"testing"

	"go.loglayer.dev"
)

// Lazy in WithFields resolves at emit time and the result lands in
// Data under the field key.
func TestLazy_WithFields_ResolvedAtEmit(t *testing.T) {
	log, lib := setup(t)
	log = log.WithFields(loglayer.Fields{
		"requestId": loglayer.Lazy(func() any { return "abc-123" }),
	})
	log.Info("served")
	line := lib.PopLine()
	if line.Data["requestId"] != "abc-123" {
		t.Errorf("requestId: got %v, want abc-123", line.Data["requestId"])
	}
}

// Lazy is root-only: a *LazyValue nested inside another value (a map,
// slice, struct field) is not resolved.
func TestLazy_OnlyResolvesAtFieldRoot(t *testing.T) {
	log, lib := setup(t)
	nested := loglayer.Lazy(func() any { return "deep" })
	log = log.WithFields(loglayer.Fields{
		"wrapped": map[string]any{"inner": nested},
	})
	log.Info("e")
	line := lib.PopLine()
	wrapped, ok := line.Data["wrapped"].(map[string]any)
	if !ok {
		t.Fatalf("wrapped: got %T", line.Data["wrapped"])
	}
	if wrapped["inner"] != nested {
		t.Errorf("nested *LazyValue should pass through; got %v (%T)", wrapped["inner"], wrapped["inner"])
	}
}

// Callback does not run when the level is filtered out.
func TestLazy_NotInvoked_WhenLevelDisabled(t *testing.T) {
	log, _ := setup(t)
	log.DisableLevel(loglayer.LogLevelDebug)

	var calls atomic.Int32
	log = log.WithFields(loglayer.Fields{
		"expensive": loglayer.Lazy(func() any {
			calls.Add(1)
			return "x"
		}),
	})
	log.Debug("won't fire")

	if calls.Load() != 0 {
		t.Errorf("lazy callback should not have run for filtered-out level; got %d invocations", calls.Load())
	}
}

// Callback fires on each emission. The wrapper persists in the
// logger's fields map; we read fresh state every time.
func TestLazy_ReEvaluatedPerEmission(t *testing.T) {
	log, lib := setup(t)

	var counter atomic.Int32
	log = log.WithFields(loglayer.Fields{
		"seq": loglayer.Lazy(func() any { return counter.Add(1) }),
	})

	log.Info("a")
	log.Info("b")
	log.Info("c")

	lines := lib.Lines()
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want 3", len(lines))
	}
	for i, want := range []int32{1, 2, 3} {
		got := lines[i].Data["seq"]
		if got != want {
			t.Errorf("line %d seq: got %v, want %d", i, got, want)
		}
	}
}

// Child loggers carry the wrapper, not a snapshot of the resolved
// value. Each child evaluates fresh at its own emission time.
func TestLazy_ChildInheritsWrapperNotResolvedValue(t *testing.T) {
	log, lib := setup(t)

	var n atomic.Int32
	parent := log.WithFields(loglayer.Fields{
		"counter": loglayer.Lazy(func() any { return n.Add(1) }),
	})

	parent.Info("from parent")
	child := parent.Child()
	child.Info("from child")

	lines := lib.Lines()
	if lines[0].Data["counter"] != int32(1) {
		t.Errorf("parent counter: got %v, want 1", lines[0].Data["counter"])
	}
	if lines[1].Data["counter"] != int32(2) {
		t.Errorf("child counter: got %v, want 2 (wrapper inherited, callback re-ran)", lines[1].Data["counter"])
	}
}

// A panicking callback substitutes [LazyEvalError]; the entry still
// emits so other fields aren't lost.
func TestLazy_PanickingCallback_SubstitutesPlaceholder(t *testing.T) {
	log, lib := setup(t)

	log.WithFields(loglayer.Fields{
		"safe": "value",
		"bad":  loglayer.Lazy(func() any { panic("boom") }),
	}).Info("event")

	line := lib.PopLine()
	if line == nil {
		t.Fatal("entry should still be emitted after a lazy panic")
	}
	if line.Data["safe"] != "value" {
		t.Errorf("safe field should survive the panic: got %v", line.Data["safe"])
	}
	if line.Data["bad"] != loglayer.LazyEvalError {
		t.Errorf("bad field: got %v, want %s", line.Data["bad"], loglayer.LazyEvalError)
	}
}

// Raw goes through the same Fields resolution path.
func TestLazy_Raw_ResolvesFields(t *testing.T) {
	log, lib := setup(t)
	log.Raw(loglayer.RawLogEntry{
		LogLevel: loglayer.LogLevelInfo,
		Messages: []any{"raw"},
		Fields:   loglayer.Fields{"f": loglayer.Lazy(func() any { return "F" })},
	})
	line := lib.PopLine()
	if line.Data["f"] != "F" {
		t.Errorf("raw fields lazy: got %v", line.Data["f"])
	}
}

// Non-lazy fields pass through unchanged.
func TestLazy_NoLazy_PassesThrough(t *testing.T) {
	log, lib := setup(t)
	log = log.WithFields(loglayer.Fields{"k": "v"})
	log.Info("e")
	if lib.PopLine().Data["k"] != "v" {
		t.Error("normal fields path broken")
	}
}

// Muted fields skip lazy resolution: the callback must not run when
// its output won't be emitted.
func TestLazy_MutedFields_CallbackSkipped(t *testing.T) {
	log, _ := setupWithConfig(t, loglayer.Config{MuteFields: true})
	var calls atomic.Int32
	log = log.WithFields(loglayer.Fields{
		"k": loglayer.Lazy(func() any { calls.Add(1); return "v" }),
	})
	log.Info("e")
	if calls.Load() != 0 {
		t.Errorf("callback ran %d times; expected 0 because MuteFields drops the output", calls.Load())
	}
}

// Derived loggers that no longer hold the lazy field stop invoking
// the callback (WithoutFields removes both the value and the
// derived-logger's lazy-tracking).
func TestLazy_DerivedLoggerWithoutLazy_NoCallback(t *testing.T) {
	log, _ := setup(t)

	var calls atomic.Int32
	withLazy := log.WithFields(loglayer.Fields{
		"k": loglayer.Lazy(func() any { calls.Add(1); return "v" }),
	})
	withLazy.Info("a")
	if calls.Load() != 1 {
		t.Fatalf("after WithFields(lazy).Info: calls=%d, want 1", calls.Load())
	}

	withoutLazy := withLazy.WithoutFields("k")
	withoutLazy.Info("b")
	if got := calls.Load(); got != 1 {
		t.Errorf("after WithoutFields(\"k\"): calls=%d, want 1 (lazy key removed; flag should be cleared on derived logger)", got)
	}

	cleared := withLazy.WithoutFields()
	cleared.Info("c")
	if got := calls.Load(); got != 1 {
		t.Errorf("after WithoutFields(): calls=%d, want 1 (all fields cleared on derived logger)", got)
	}
}

// Dispatch-time plugin hooks see the resolved value in Data, not the
// *LazyValue wrapper.
func TestLazy_DispatchHookSeesResolvedValue(t *testing.T) {
	var seenInData atomic.Value
	log, _ := setup(t)
	log.AddPlugin(loglayer.NewDataHook("capture", func(p loglayer.BeforeDataOutParams) loglayer.Data {
		if v, ok := p.Data["k"]; ok {
			seenInData.Store(v)
		}
		return p.Data
	}))

	log = log.WithFields(loglayer.Fields{
		"k": loglayer.Lazy(func() any { return "resolved" }),
	})
	log.Info("e")

	got := seenInData.Load()
	if got != "resolved" {
		t.Errorf("OnBeforeDataOut saw %v in Data['k'], want \"resolved\"", got)
	}
}
