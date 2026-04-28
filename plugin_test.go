package loglayer_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
	lltest "go.loglayer.dev/transports/testing"
)

func TestPlugin_OnBeforeDataOut_AddsKeys(t *testing.T) {
	log, lib := setup(t)
	log.AddPlugin(loglayer.Plugin{
		ID: "add-keys",
		OnBeforeDataOut: func(p loglayer.BeforeDataOutParams) loglayer.Data {
			return loglayer.Data{"plugin_key": "plugin_value"}
		},
	})

	log.Info("hi")
	line := lib.PopLine()
	if line.Data["plugin_key"] != "plugin_value" {
		t.Errorf("plugin should add key: got %v", line.Data)
	}
}

func TestPlugin_OnBeforeMessageOut_RewritesMessages(t *testing.T) {
	log, lib := setup(t)
	log.AddPlugin(loglayer.Plugin{
		ID: "uppercase",
		OnBeforeMessageOut: func(p loglayer.BeforeMessageOutParams) []any {
			return []any{"REWRITTEN"}
		},
	})

	log.Info("original")
	line := lib.PopLine()
	if line.Messages[0] != "REWRITTEN" {
		t.Errorf("messages should be rewritten: got %v", line.Messages)
	}
}

func TestPlugin_TransformLogLevel_OverridesLevel(t *testing.T) {
	log, lib := setup(t)
	log.AddPlugin(loglayer.Plugin{
		ID: "promote-to-warn",
		TransformLogLevel: func(p loglayer.TransformLogLevelParams) (loglayer.LogLevel, bool) {
			if p.LogLevel == loglayer.LogLevelInfo {
				return loglayer.LogLevelWarn, true
			}
			return 0, false
		},
	})

	log.Info("originally info")
	line := lib.PopLine()
	if line.Level != loglayer.LogLevelWarn {
		t.Errorf("plugin should promote info → warn, got %s", line.Level)
	}
}

func TestPlugin_ShouldSend_VetoesPerTransport(t *testing.T) {
	lib1 := &lltest.TestLoggingLibrary{}
	lib2 := &lltest.TestLoggingLibrary{}
	t1 := lltest.New(lltest.Config{BaseConfig: transport.BaseConfig{ID: "keep"}, Library: lib1})
	t2 := lltest.New(lltest.Config{BaseConfig: transport.BaseConfig{ID: "drop"}, Library: lib2})
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transports:       []loglayer.Transport{t1, t2},
	})

	log.AddPlugin(loglayer.Plugin{
		ID: "skip-drop-transport",
		ShouldSend: func(p loglayer.ShouldSendParams) bool {
			return p.TransportID != "drop"
		},
	})

	log.Info("selective")
	if lib1.Len() != 1 {
		t.Errorf("keep transport should receive: got %d", lib1.Len())
	}
	if lib2.Len() != 0 {
		t.Errorf("drop transport should be skipped: got %d", lib2.Len())
	}
}

func TestPlugin_OnMetadataCalled_RewritesMetadata(t *testing.T) {
	log, lib := setup(t)
	log.AddPlugin(loglayer.Plugin{
		ID: "redact-password",
		OnMetadataCalled: func(metadata any) any {
			m, ok := metadata.(map[string]any)
			if !ok {
				return metadata
			}
			if _, has := m["password"]; has {
				m["password"] = "[REDACTED]"
			}
			return m
		},
	})

	log.WithMetadata(map[string]any{"username": "alice", "password": "secret"}).Info("login")
	line := lib.PopLine()
	m := line.Metadata.(map[string]any)
	if m["password"] != "[REDACTED]" {
		t.Errorf("password should be redacted: got %v", m["password"])
	}
	if m["username"] != "alice" {
		t.Errorf("username should be preserved: got %v", m["username"])
	}
}

func TestPlugin_OnMetadataCalled_NilDropsMetadata(t *testing.T) {
	log, lib := setup(t)
	log.AddPlugin(loglayer.Plugin{
		ID: "drop-all",
		OnMetadataCalled: func(metadata any) any {
			return nil
		},
	})

	log.WithMetadata(map[string]any{"k": "v"}).Info("dropped")
	line := lib.PopLine()
	if line.Metadata != nil {
		t.Errorf("metadata should be nil after plugin drop: got %v", line.Metadata)
	}
}

func TestPlugin_OnFieldsCalled_RewritesFields(t *testing.T) {
	log, lib := setup(t)
	log.AddPlugin(loglayer.Plugin{
		ID: "uppercase-keys",
		OnFieldsCalled: func(fields loglayer.Fields) loglayer.Fields {
			out := make(loglayer.Fields, len(fields))
			for k, v := range fields {
				out["U_"+k] = v
			}
			return out
		},
	})

	log = log.WithFields(loglayer.Fields{"requestId": "abc"})
	log.Info("hi")
	line := lib.PopLine()
	if line.Data["U_requestId"] != "abc" {
		t.Errorf("plugin should rewrite field key: got %v", line.Data)
	}
	if _, present := line.Data["requestId"]; present {
		t.Errorf("original field key should be gone: got %v", line.Data)
	}
}

func TestPlugin_MultiplePlugins_RunInRegistrationOrder(t *testing.T) {
	log, lib := setup(t)
	log.AddPlugin(loglayer.Plugin{
		ID: "first",
		OnBeforeMessageOut: func(p loglayer.BeforeMessageOutParams) []any {
			return []any{"first"}
		},
	})
	log.AddPlugin(loglayer.Plugin{
		ID: "second",
		OnBeforeMessageOut: func(p loglayer.BeforeMessageOutParams) []any {
			// Should see "first" as the input.
			if s, ok := p.Messages[0].(string); !ok || s != "first" {
				t.Errorf("second plugin should see first plugin's output, got %v", p.Messages)
			}
			return []any{"second"}
		},
	})

	log.Info("ignored")
	line := lib.PopLine()
	if line.Messages[0] != "second" {
		t.Errorf("last plugin's output should win: got %v", line.Messages)
	}
}

func TestPlugin_ReplaceByID(t *testing.T) {
	log, _ := setup(t)
	log.AddPlugin(loglayer.Plugin{ID: "p1"})
	log.AddPlugin(loglayer.Plugin{ID: "p1"}) // replace, not add

	if log.PluginCount() != 1 {
		t.Errorf("re-adding same ID should replace: got %d plugins", log.PluginCount())
	}
}

func TestPlugin_RemovePlugin(t *testing.T) {
	log, _ := setup(t)
	log.AddPlugin(loglayer.Plugin{ID: "to-remove"})

	if !log.RemovePlugin("to-remove") {
		t.Error("RemovePlugin should return true for known ID")
	}
	if log.RemovePlugin("nonexistent") {
		t.Error("RemovePlugin should return false for unknown ID")
	}
	if log.PluginCount() != 0 {
		t.Errorf("expected 0 plugins after remove, got %d", log.PluginCount())
	}
}

func TestPlugin_GetPlugin(t *testing.T) {
	log, _ := setup(t)
	log.AddPlugin(loglayer.Plugin{ID: "lookup"})

	p, ok := log.GetPlugin("lookup")
	if !ok {
		t.Fatal("GetPlugin should find registered plugin")
	}
	if p.ID != "lookup" {
		t.Errorf("returned plugin ID: got %q", p.ID)
	}
	if _, ok := log.GetPlugin("missing"); ok {
		t.Error("GetPlugin should return ok=false for unknown ID")
	}
}

func TestPlugin_AddPlugin_AutoGeneratesEmptyID(t *testing.T) {
	log, _ := setup(t)
	before := log.PluginCount()
	log.AddPlugin(loglayer.Plugin{
		OnBeforeDataOut: func(_ loglayer.BeforeDataOutParams) loglayer.Data { return nil },
	})
	if log.PluginCount() != before+1 {
		t.Fatalf("plugin count: got %d, want %d", log.PluginCount(), before+1)
	}
	// The auto-generated ID isn't directly returned, but the plugin should be
	// registered and reachable via the catalog (one new entry, prefixed).
}

func TestBuild_PluginEmptyID_AutoGenerates(t *testing.T) {
	tr := discardTransport{}
	log, err := loglayer.Build(loglayer.Config{
		Transport: tr,
		Plugins: []loglayer.Plugin{{
			OnBeforeDataOut: func(_ loglayer.BeforeDataOutParams) loglayer.Data { return nil },
		}},
	})
	if err != nil {
		t.Fatalf("Build with empty plugin ID should succeed; got %v", err)
	}
	if log.PluginCount() != 1 {
		t.Errorf("plugin count: got %d, want 1", log.PluginCount())
	}
}

func TestPlugin_ChildInheritsPlugins(t *testing.T) {
	parent, lib := setup(t)
	parent.AddPlugin(loglayer.Plugin{
		ID: "marker",
		OnBeforeDataOut: func(p loglayer.BeforeDataOutParams) loglayer.Data {
			return loglayer.Data{"from_plugin": true}
		},
	})

	child := parent.Child()
	child.Info("from child")
	line := lib.PopLine()
	if line.Data["from_plugin"] != true {
		t.Errorf("child should inherit parent's plugin: got %v", line.Data)
	}
}

func TestPlugin_ChildPluginIsolation(t *testing.T) {
	parent, lib := setup(t)
	child := parent.Child()
	child.AddPlugin(loglayer.Plugin{
		ID: "child-only",
		OnBeforeDataOut: func(p loglayer.BeforeDataOutParams) loglayer.Data {
			return loglayer.Data{"from_child": true}
		},
	})

	parent.Info("parent log")
	line := lib.PopLine()
	if line.Data["from_child"] != nil {
		t.Errorf("parent should not see child-only plugin: got %v", line.Data)
	}
}

func TestPlugin_TransformLogLevel_DroppedByLevelFilter(t *testing.T) {
	log, lib := setup(t)
	log.SetLevel(loglayer.LogLevelError)
	log.AddPlugin(loglayer.Plugin{
		ID: "demote",
		TransformLogLevel: func(p loglayer.TransformLogLevelParams) (loglayer.LogLevel, bool) {
			return loglayer.LogLevelInfo, true
		},
	})

	// The log starts at Error and would normally emit, but the plugin
	// demotes it to Info. Note: level filtering happens BEFORE the plugin
	// pipeline (the level method short-circuits if disabled), so the
	// transform happens after the filter check. This documents that
	// TransformLogLevel cannot save an entry that was already filtered out.
	log.Error("originally error")
	line := lib.PopLine()
	if line == nil {
		t.Fatal("expected line: filter happens before plugin transforms")
	}
	if line.Level != loglayer.LogLevelInfo {
		t.Errorf("plugin should have demoted to info: got %s", line.Level)
	}
}

func TestPlugin_BeforeDataOut_SeesError(t *testing.T) {
	log, lib := setup(t)
	var sawError bool
	log.AddPlugin(loglayer.Plugin{
		ID: "see-error",
		OnBeforeDataOut: func(p loglayer.BeforeDataOutParams) loglayer.Data {
			if p.Err != nil && p.Err.Error() == "boom" {
				sawError = true
			}
			return nil
		},
	})

	log.WithError(errors.New("boom")).Error("failed")
	_ = lib.PopLine()
	if !sawError {
		t.Error("plugin OnBeforeDataOut should see the error in params")
	}
}

func TestPlugin_DispatchHooksReceiveCtx(t *testing.T) {
	type ctxKey struct{}
	ctx := context.WithValue(context.Background(), ctxKey{}, "value")

	var dataCtx, msgCtx, lvlCtx, sendCtx context.Context

	log, _ := setup(t)
	log.AddPlugin(loglayer.Plugin{
		ID: "ctx-capture",
		OnBeforeDataOut: func(p loglayer.BeforeDataOutParams) loglayer.Data {
			dataCtx = p.Ctx
			return nil
		},
		OnBeforeMessageOut: func(p loglayer.BeforeMessageOutParams) []any {
			msgCtx = p.Ctx
			return nil
		},
		TransformLogLevel: func(p loglayer.TransformLogLevelParams) (loglayer.LogLevel, bool) {
			lvlCtx = p.Ctx
			return 0, false
		},
		ShouldSend: func(p loglayer.ShouldSendParams) bool {
			sendCtx = p.Ctx
			return true
		},
	})

	log.WithCtx(ctx).Info("hello")

	for name, got := range map[string]context.Context{
		"OnBeforeDataOut":    dataCtx,
		"OnBeforeMessageOut": msgCtx,
		"TransformLogLevel":  lvlCtx,
		"ShouldSend":         sendCtx,
	} {
		if got == nil {
			t.Errorf("%s: Ctx not propagated (nil)", name)
			continue
		}
		if v, _ := got.Value(ctxKey{}).(string); v != "value" {
			t.Errorf("%s: Ctx didn't carry the user's value", name)
		}
	}
}

func TestPlugin_PostRegistrationMutationIsNoOp(t *testing.T) {
	log, lib := setup(t)
	called := false
	p := loglayer.Plugin{
		ID: "mutate",
		OnBeforeDataOut: func(p loglayer.BeforeDataOutParams) loglayer.Data {
			called = true
			return loglayer.Data{"original": true}
		},
	}
	log.AddPlugin(p)

	// Mutating after AddPlugin should have no effect on the registered plugin.
	p.OnBeforeDataOut = func(p loglayer.BeforeDataOutParams) loglayer.Data {
		return loglayer.Data{"mutated": true}
	}

	log.Info("ok")
	line := lib.PopLine()
	if !called {
		t.Fatal("registered hook did not fire")
	}
	if line.Data["mutated"] == true {
		t.Errorf("post-registration mutation took effect: %v", line.Data)
	}
	if line.Data["original"] != true {
		t.Errorf("original hook output missing: %v", line.Data)
	}
}

func TestPlugin_ConcurrentAddAndEmit(t *testing.T) {
	log, _ := setup(t)
	log.AddPlugin(loglayer.Plugin{
		ID: "always",
		OnBeforeMessageOut: func(p loglayer.BeforeMessageOutParams) []any {
			return p.Messages
		},
	})

	const emitters = 8
	const adders = 4
	const iters = 100

	var wg sync.WaitGroup
	var stop atomic.Bool

	wg.Add(emitters)
	for g := 0; g < emitters; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				log.Info("traffic")
				if stop.Load() {
					return
				}
			}
		}()
	}

	wg.Add(adders)
	for g := 0; g < adders; g++ {
		gid := g
		go func() {
			defer wg.Done()
			for i := 0; i < iters/4; i++ {
				id := "p_" + string(rune('a'+gid)) + "_" + string(rune('0'+i%10))
				log.AddPlugin(loglayer.Plugin{
					ID: id,
					OnBeforeMessageOut: func(p loglayer.BeforeMessageOutParams) []any {
						return p.Messages
					},
				})
				log.RemovePlugin(id)
			}
		}()
	}

	wg.Wait()
	stop.Store(true)
}

// OnBeforeDataOut returning nil leaves the assembled data unchanged. The
// plugin's no-op path is documented but untested.
func TestPlugin_OnBeforeDataOut_NilPreservesData(t *testing.T) {
	log, lib := setup(t)
	log.AddPlugin(loglayer.Plugin{
		ID: "no-op",
		OnBeforeDataOut: func(p loglayer.BeforeDataOutParams) loglayer.Data {
			return nil
		},
	})
	log = log.WithFields(loglayer.Fields{"req": "abc"})
	log.Info("emit")
	line := lib.PopLine()
	if line.Data["req"] != "abc" {
		t.Errorf("OnBeforeDataOut returning nil should preserve assembled data: %v", line.Data)
	}
}

// OnFieldsCalled returning nil drops the WithFields call entirely
// (parallel to TestPlugin_OnMetadataCalled_NilDropsMetadata).
func TestPlugin_OnFieldsCalled_NilDropsFields(t *testing.T) {
	log, lib := setup(t)
	log.AddPlugin(loglayer.Plugin{
		ID: "drop-fields",
		OnFieldsCalled: func(fields loglayer.Fields) loglayer.Fields {
			return nil
		},
	})
	log = log.WithFields(loglayer.Fields{"k": "v"})
	log.Info("emit")
	line := lib.PopLine()
	if line.Data["k"] != nil {
		t.Errorf("OnFieldsCalled returning nil should drop the WithFields call: %v", line.Data)
	}
}

// Multiple OnFieldsCalled plugins chain, each seeing the previous output.
func TestPlugin_OnFieldsCalled_Chain(t *testing.T) {
	log, lib := setup(t)
	log.AddPlugin(loglayer.Plugin{
		ID: "first",
		OnFieldsCalled: func(fields loglayer.Fields) loglayer.Fields {
			out := make(loglayer.Fields, len(fields))
			for k, v := range fields {
				out["A_"+k] = v
			}
			return out
		},
	})
	log.AddPlugin(loglayer.Plugin{
		ID: "second",
		OnFieldsCalled: func(fields loglayer.Fields) loglayer.Fields {
			out := make(loglayer.Fields, len(fields))
			for k, v := range fields {
				out["B_"+k] = v
			}
			return out
		},
	})
	log = log.WithFields(loglayer.Fields{"k": "v"})
	log.Info("emit")
	line := lib.PopLine()
	if line.Data["B_A_k"] != "v" {
		t.Errorf("plugins should chain OnFieldsCalled: got %v", line.Data)
	}
}

// When OnMetadataCalled returns nil, the chain short-circuits: subsequent
// plugins do not fire.
func TestPlugin_OnMetadataCalled_NilShortCircuits(t *testing.T) {
	log, _ := setup(t)
	secondCalled := false
	log.AddPlugin(loglayer.Plugin{
		ID: "drop",
		OnMetadataCalled: func(metadata any) any {
			return nil
		},
	})
	log.AddPlugin(loglayer.Plugin{
		ID: "after-drop",
		OnMetadataCalled: func(metadata any) any {
			secondCalled = true
			return metadata
		},
	})
	log.WithMetadata(map[string]any{"k": "v"}).Info("hi")
	if secondCalled {
		t.Error("plugin chain should short-circuit when an earlier hook returns nil")
	}
}

// Same short-circuit behavior for OnFieldsCalled.
func TestPlugin_OnFieldsCalled_NilShortCircuits(t *testing.T) {
	log, _ := setup(t)
	secondCalled := false
	log.AddPlugin(loglayer.Plugin{
		ID: "drop",
		OnFieldsCalled: func(fields loglayer.Fields) loglayer.Fields {
			return nil
		},
	})
	log.AddPlugin(loglayer.Plugin{
		ID: "after-drop",
		OnFieldsCalled: func(fields loglayer.Fields) loglayer.Fields {
			secondCalled = true
			return fields
		},
	})
	log = log.WithFields(loglayer.Fields{"k": "v"})
	_ = log
	if secondCalled {
		t.Error("OnFieldsCalled chain should short-circuit on nil")
	}
}

// Pin the dispatch-time hook ordering: OnBeforeDataOut runs first, then
// OnBeforeMessageOut, then TransformLogLevel, and ShouldSend last (per
// transport).
func TestPlugin_DispatchHookOrdering(t *testing.T) {
	log, _ := setup(t)
	var calls []string

	log.AddPlugin(loglayer.Plugin{
		ID: "ordering",
		OnBeforeDataOut: func(p loglayer.BeforeDataOutParams) loglayer.Data {
			calls = append(calls, "data")
			return nil
		},
		OnBeforeMessageOut: func(p loglayer.BeforeMessageOutParams) []any {
			calls = append(calls, "messages")
			return nil
		},
		TransformLogLevel: func(p loglayer.TransformLogLevelParams) (loglayer.LogLevel, bool) {
			calls = append(calls, "level")
			return 0, false
		},
		ShouldSend: func(p loglayer.ShouldSendParams) bool {
			calls = append(calls, "send")
			return true
		},
	})
	log.Info("ordered")

	want := []string{"data", "messages", "level", "send"}
	if len(calls) != len(want) {
		t.Fatalf("got %d hook calls, want %d: %v", len(calls), len(want), calls)
	}
	for i, w := range want {
		if calls[i] != w {
			t.Errorf("hook %d: got %q, want %q (full sequence: %v)", i, calls[i], w, calls)
		}
	}
}

// TransformLogLevel must receive Metadata and Err on its params (not just
// LogLevel and Data).
func TestPlugin_TransformLogLevel_SeesMetadataAndErr(t *testing.T) {
	log, _ := setup(t)
	var sawMeta any
	var sawErr error
	log.AddPlugin(loglayer.Plugin{
		ID: "inspect",
		TransformLogLevel: func(p loglayer.TransformLogLevelParams) (loglayer.LogLevel, bool) {
			sawMeta = p.Metadata
			sawErr = p.Err
			return 0, false
		},
	})
	log.WithMetadata(map[string]any{"k": "v"}).WithError(errors.New("boom")).Error("explode")

	m, ok := sawMeta.(map[string]any)
	if !ok || m["k"] != "v" {
		t.Errorf("TransformLogLevel should see Metadata: got %T %v", sawMeta, sawMeta)
	}
	if sawErr == nil || sawErr.Error() != "boom" {
		t.Errorf("TransformLogLevel should see Err: got %v", sawErr)
	}
}

// Raw entries flow through the plugin pipeline.
func TestPlugin_Raw_RunsPipeline(t *testing.T) {
	log, lib := setup(t)
	log.AddPlugin(loglayer.Plugin{
		ID: "raw-mutator",
		OnBeforeDataOut: func(p loglayer.BeforeDataOutParams) loglayer.Data {
			return loglayer.Data{"plugin_added": true}
		},
		OnBeforeMessageOut: func(p loglayer.BeforeMessageOutParams) []any {
			return []any{"rewritten"}
		},
	})
	log.Raw(loglayer.RawLogEntry{
		LogLevel: loglayer.LogLevelInfo,
		Messages: []any{"original"},
	})
	line := lib.PopLine()
	if line.Data["plugin_added"] != true {
		t.Errorf("OnBeforeDataOut should run on Raw entries: %v", line.Data)
	}
	if len(line.Messages) != 1 || line.Messages[0] != "rewritten" {
		t.Errorf("OnBeforeMessageOut should run on Raw entries: %v", line.Messages)
	}
}

// The framework recovers panics in plugin hooks so a buggy plugin can't
// tear down the caller's goroutine. Plugin.OnError surfaces the
// recovered panic for the plugin to observe.
func TestPlugin_HookPanicRecovered(t *testing.T) {
	log, lib := setup(t)

	hooks := []struct {
		name   string
		plugin loglayer.Plugin
		emit   func()
	}{
		{
			name: "OnBeforeDataOut",
			plugin: loglayer.Plugin{
				ID:              "p-bdo",
				OnBeforeDataOut: func(p loglayer.BeforeDataOutParams) loglayer.Data { panic("bdo") },
			},
			emit: func() { log.Info("ok") },
		},
		{
			name: "OnBeforeMessageOut",
			plugin: loglayer.Plugin{
				ID:                 "p-bmo",
				OnBeforeMessageOut: func(p loglayer.BeforeMessageOutParams) []any { panic("bmo") },
			},
			emit: func() { log.Info("ok") },
		},
		{
			name: "TransformLogLevel",
			plugin: loglayer.Plugin{
				ID:                "p-tll",
				TransformLogLevel: func(p loglayer.TransformLogLevelParams) (loglayer.LogLevel, bool) { panic("tll") },
			},
			emit: func() { log.Info("ok") },
		},
		{
			name: "ShouldSend",
			plugin: loglayer.Plugin{
				ID:         "p-ss",
				ShouldSend: func(p loglayer.ShouldSendParams) bool { panic("ss") },
			},
			emit: func() { log.Info("ok") },
		},
		{
			name: "OnMetadataCalled",
			plugin: loglayer.Plugin{
				ID:               "p-omc",
				OnMetadataCalled: func(metadata any) any { panic("omc") },
			},
			emit: func() { log.WithMetadata(loglayer.Metadata{"k": "v"}).Info("ok") },
		},
		{
			name: "OnFieldsCalled",
			plugin: loglayer.Plugin{
				ID:             "p-ofc",
				OnFieldsCalled: func(f loglayer.Fields) loglayer.Fields { panic("ofc") },
			},
			// WithFields is what triggers OnFieldsCalled. The result
			// logger is then used for the emission so we have something
			// to observe.
			emit: func() { log.WithFields(loglayer.Fields{"a": 1}).Info("ok") },
		},
	}

	for _, h := range hooks {
		t.Run(h.name, func(t *testing.T) {
			var caught error
			plugin := h.plugin
			plugin.OnError = func(err error) { caught = err }
			log.AddPlugin(plugin)
			defer log.RemovePlugin(plugin.ID)

			lib.ClearLines()
			h.emit() // must not panic

			if caught == nil {
				t.Fatalf("%s: OnError should have been called", h.name)
			}
			if !strings.Contains(caught.Error(), h.name) {
				t.Errorf("%s: error message should name the hook: got %q", h.name, caught.Error())
			}
		})
	}
}

// When OnError is nil, hook panics are silently recovered (logging
// continues). The framework MUST NOT propagate the panic to the caller.
func TestPlugin_HookPanicSilentWhenNoOnError(t *testing.T) {
	log, lib := setup(t)
	log.AddPlugin(loglayer.Plugin{
		ID:              "panicker",
		OnBeforeDataOut: func(p loglayer.BeforeDataOutParams) loglayer.Data { panic("boom") },
		// OnError nil
	})

	// Must not panic.
	log.Info("entry")
	if lib.Len() != 1 {
		t.Errorf("entry should still emit even when hook panics silently: got %d lines", lib.Len())
	}
}

// ShouldSend fails open: a panicking gate doesn't drop the entry. This is
// the safer default for a logging library: silent dropping would mask
// plugin bugs as data loss.
func TestPlugin_ShouldSendPanicFailsOpen(t *testing.T) {
	log, lib := setup(t)
	log.AddPlugin(loglayer.Plugin{
		ID:         "panicker",
		ShouldSend: func(p loglayer.ShouldSendParams) bool { panic("gate-broken") },
	})
	log.Info("survived")
	if lib.Len() != 1 {
		t.Errorf("ShouldSend panic should not drop the entry; got %d lines", lib.Len())
	}
}

// The error passed to OnError is a *RecoveredPanicError that exposes the
// hook name and the original recovered value, and unwraps to the original
// error when the panic value implemented error.
func TestPlugin_RecoveredPanicErrorTypedInspection(t *testing.T) {
	log, _ := setup(t)
	originalErr := errors.New("boom")
	var caught error
	log.AddPlugin(loglayer.Plugin{
		ID:              "panicker",
		OnBeforeDataOut: func(p loglayer.BeforeDataOutParams) loglayer.Data { panic(originalErr) },
		OnError:         func(err error) { caught = err },
	})
	log.Info("trigger")

	var rpe *loglayer.RecoveredPanicError
	if !errors.As(caught, &rpe) {
		t.Fatalf("caught error should be *RecoveredPanicError, got %T (%v)", caught, caught)
	}
	if rpe.Hook != "OnBeforeDataOut" {
		t.Errorf("Hook: got %q, want %q", rpe.Hook, "OnBeforeDataOut")
	}
	if rpe.Value != error(originalErr) {
		t.Errorf("Value: got %v, want %v", rpe.Value, originalErr)
	}
	if !errors.Is(caught, originalErr) {
		t.Errorf("errors.Is should reach the wrapped panic value")
	}
}

// When the panic value is a non-error (string, int, custom struct),
// the Value field exposes it. Unwrap returns nil because there's no
// error chain to follow.
func TestPlugin_RecoveredPanicErrorWithNonErrorValue(t *testing.T) {
	log, _ := setup(t)
	var caught error
	log.AddPlugin(loglayer.Plugin{
		ID:              "panicker",
		OnBeforeDataOut: func(p loglayer.BeforeDataOutParams) loglayer.Data { panic("string-panic") },
		OnError:         func(err error) { caught = err },
	})
	log.Info("trigger")

	var rpe *loglayer.RecoveredPanicError
	if !errors.As(caught, &rpe) {
		t.Fatalf("caught error should be *RecoveredPanicError")
	}
	if rpe.Value != "string-panic" {
		t.Errorf("Value: got %v, want %q", rpe.Value, "string-panic")
	}
	if errors.Unwrap(caught) != nil {
		t.Errorf("Unwrap should be nil for non-error panic values")
	}
}
