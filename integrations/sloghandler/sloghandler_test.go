package sloghandler_test

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"go.loglayer.dev"
	"go.loglayer.dev/integrations/sloghandler"
	lltest "go.loglayer.dev/transports/testing"
)

func newSlog(t *testing.T) (*slog.Logger, *lltest.TestLoggingLibrary, *loglayer.LogLayer) {
	t.Helper()
	lib := &lltest.TestLoggingLibrary{}
	log := loglayer.New(loglayer.Config{
		Transport:        lltest.New(lltest.Config{Library: lib}),
		DisableFatalExit: true,
	})
	return slog.New(sloghandler.New(log)), lib, log
}

func TestBasicLevels(t *testing.T) {
	t.Parallel()
	l, lib, _ := newSlog(t)

	l.Debug("d")
	l.Info("i")
	l.Warn("w")
	l.Error("e")

	lines := lib.Lines()
	if len(lines) != 4 {
		t.Fatalf("expected 4 lines, got %d", len(lines))
	}
	got := []loglayer.LogLevel{lines[0].Level, lines[1].Level, lines[2].Level, lines[3].Level}
	want := []loglayer.LogLevel{loglayer.LogLevelDebug, loglayer.LogLevelInfo, loglayer.LogLevelWarn, loglayer.LogLevelError}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line[%d] level: got %v, want %v", i, got[i], want[i])
		}
	}
}

// slog has no Fatal level; values at or above slog.LevelError pin to
// LogLevelError so a custom slog level can never accidentally exit the
// process via loglayer.LogLevelFatal.
func TestSlogHighLevelDoesNotMapToFatal(t *testing.T) {
	t.Parallel()
	l, lib, _ := newSlog(t)

	l.Log(context.Background(), slog.Level(99), "above-error")
	line := lib.PopLine()
	if line.Level != loglayer.LogLevelError {
		t.Errorf("custom high level should pin to LogLevelError, got %v", line.Level)
	}
}

func TestDefaultLogger_GoesThroughHandler(t *testing.T) {
	t.Parallel()
	l, lib, _ := newSlog(t)

	// Don't actually call slog.SetDefault here; that would race with other
	// parallel tests. Use the constructed logger directly.
	l.Info("via slog")

	line := lib.PopLine()
	if line.Messages[0] != "via slog" {
		t.Errorf("message should pass through unchanged: got %v", line.Messages[0])
	}
}

func TestInlineAttrs_BecomePerCallFields(t *testing.T) {
	t.Parallel()
	l, lib, _ := newSlog(t)

	l.Info("hello", "userId", 42, "tenant", "acme")

	line := lib.PopLine()
	if line.Data["userId"] != int64(42) {
		t.Errorf("userId: got %v (%T), want int64(42)", line.Data["userId"], line.Data["userId"])
	}
	if line.Data["tenant"] != "acme" {
		t.Errorf("tenant: got %v", line.Data["tenant"])
	}
}

func TestWithAttrs_PersistsAcrossCalls(t *testing.T) {
	t.Parallel()
	l, lib, _ := newSlog(t)

	scoped := l.With("requestId", "abc")
	scoped.Info("first")
	scoped.Info("second")

	for i, line := range lib.Lines() {
		if line.Data["requestId"] != "abc" {
			t.Errorf("line[%d] missing persistent requestId: %v", i, line.Data)
		}
	}
}

func TestWithAttrs_DerivedHandlersAreIndependent(t *testing.T) {
	t.Parallel()
	l, lib, _ := newSlog(t)

	a := l.With("scope", "a")
	b := l.With("scope", "b")
	a.Info("from-a")
	b.Info("from-b")
	l.Info("from-base")

	lines := lib.Lines()
	if lines[0].Data["scope"] != "a" {
		t.Errorf("a: %v", lines[0].Data)
	}
	if lines[1].Data["scope"] != "b" {
		t.Errorf("b: %v", lines[1].Data)
	}
	if _, ok := lines[2].Data["scope"]; ok {
		t.Errorf("base logger should not see derived attrs: %v", lines[2].Data)
	}
}

func TestWithGroup_NestsAttrsUnderGroupName(t *testing.T) {
	t.Parallel()
	l, lib, _ := newSlog(t)

	l.WithGroup("http").Info("served", "method", "GET", "status", 200)

	line := lib.PopLine()
	httpGroup, ok := line.Data["http"].(map[string]any)
	if !ok {
		t.Fatalf("expected http to be a nested map, got %T: %v", line.Data["http"], line.Data["http"])
	}
	if httpGroup["method"] != "GET" {
		t.Errorf("method under http: %v", httpGroup["method"])
	}
	if httpGroup["status"] != int64(200) {
		t.Errorf("status under http: %v", httpGroup["status"])
	}
}

func TestWithGroup_StackedGroupsNest(t *testing.T) {
	t.Parallel()
	l, lib, _ := newSlog(t)

	l.WithGroup("http").WithGroup("request").Info("rx", "path", "/x")

	line := lib.PopLine()
	httpG, ok := line.Data["http"].(map[string]any)
	if !ok {
		t.Fatalf("missing http group: %v", line.Data)
	}
	reqG, ok := httpG["request"].(map[string]any)
	if !ok {
		t.Fatalf("missing http.request group: %v", httpG)
	}
	if reqG["path"] != "/x" {
		t.Errorf("path under http.request: %v", reqG["path"])
	}
}

func TestWithGroup_PreGroupAttrsStayAtRoot(t *testing.T) {
	t.Parallel()
	l, lib, _ := newSlog(t)

	// "a" was added before WithGroup; "b" after.
	l.With("a", 1).WithGroup("g").With("b", 2).Info("hi")

	line := lib.PopLine()
	if line.Data["a"] != int64(1) {
		t.Errorf("pre-group attr should stay at root: got %v in %v", line.Data["a"], line.Data)
	}
	g, ok := line.Data["g"].(map[string]any)
	if !ok {
		t.Fatalf("expected g to be a nested map: %v", line.Data)
	}
	if g["b"] != int64(2) {
		t.Errorf("post-group attr should nest: got %v in %v", g["b"], g)
	}
}

func TestEmptyGroupName_IsNoOp(t *testing.T) {
	t.Parallel()
	l, lib, _ := newSlog(t)

	l.WithGroup("").Info("hi", "k", "v")

	line := lib.PopLine()
	if line.Data["k"] != "v" {
		t.Errorf("empty group should be a no-op; expected k=v at root, got %v", line.Data)
	}
}

func TestEmptySlogGroupAttr_IsDropped(t *testing.T) {
	t.Parallel()
	l, lib, _ := newSlog(t)

	l.Info("hi", slog.Group("g"))

	line := lib.PopLine()
	if _, ok := line.Data["g"]; ok {
		t.Errorf("empty slog.Group should be dropped, got %v", line.Data)
	}
}

func TestEmptyKeyGroupAttr_InlinesMembers(t *testing.T) {
	t.Parallel()
	l, lib, _ := newSlog(t)

	// slog convention: a Group with key "" inlines its members at the parent.
	l.Info("hi", slog.Group("", slog.String("a", "1"), slog.Int("b", 2)))

	line := lib.PopLine()
	if line.Data["a"] != "1" {
		t.Errorf("inlined a: got %v", line.Data["a"])
	}
	if line.Data["b"] != int64(2) {
		t.Errorf("inlined b: got %v", line.Data["b"])
	}
}

type testValuer struct{ id int }

func (t testValuer) LogValue() slog.Value { return slog.IntValue(t.id) }

func TestLogValuer_IsResolved(t *testing.T) {
	t.Parallel()
	l, lib, _ := newSlog(t)

	l.Info("hi", "user", testValuer{id: 7})

	line := lib.PopLine()
	if line.Data["user"] != int64(7) {
		t.Errorf("LogValuer should resolve to its value: got %v (%T)", line.Data["user"], line.Data["user"])
	}
}

func TestEmptyKeyAttr_IsDropped(t *testing.T) {
	t.Parallel()
	l, lib, _ := newSlog(t)

	l.Info("hi", "", "ignored", "real", "kept")

	line := lib.PopLine()
	if _, ok := line.Data[""]; ok {
		t.Errorf("empty-key attr should be dropped: %v", line.Data)
	}
	if line.Data["real"] != "kept" {
		t.Errorf("non-empty attr should pass: %v", line.Data)
	}
}

func TestNativeKinds_PreserveTypes(t *testing.T) {
	t.Parallel()
	l, lib, _ := newSlog(t)

	when := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	l.Info("kinds",
		slog.String("s", "x"),
		slog.Int64("i", 1<<40),
		slog.Uint64("u", 1<<40),
		slog.Float64("f", 1.5),
		slog.Bool("b", true),
		slog.Time("t", when),
		slog.Duration("d", 2*time.Second),
	)

	line := lib.PopLine()
	checks := map[string]any{
		"s": "x",
		"i": int64(1 << 40),
		"u": uint64(1 << 40),
		"f": 1.5,
		"b": true,
		"t": when,
		"d": 2 * time.Second,
	}
	for k, want := range checks {
		if line.Data[k] != want {
			t.Errorf("%s: got %v (%T), want %v (%T)", k, line.Data[k], line.Data[k], want, want)
		}
	}
}

func TestUnderlyingLoggerFields_ArePreserved(t *testing.T) {
	t.Parallel()
	lib := &lltest.TestLoggingLibrary{}
	log := loglayer.New(loglayer.Config{
		Transport:        lltest.New(lltest.Config{Library: lib}),
		DisableFatalExit: true,
	}).WithFields(loglayer.Fields{"app": "foo"})

	l := slog.New(sloghandler.New(log))
	l.Info("hi")
	l.Info("with-attr", "k", "v")

	lines := lib.Lines()
	if lines[0].Data["app"] != "foo" {
		t.Errorf("base logger fields lost on no-attr emission: %v", lines[0].Data)
	}
	if lines[1].Data["app"] != "foo" {
		t.Errorf("base logger fields lost when slog adds attrs: %v", lines[1].Data)
	}
	if lines[1].Data["k"] != "v" {
		t.Errorf("slog attr missing: %v", lines[1].Data)
	}
}

func TestLevelFilter_HonoredViaEnabled(t *testing.T) {
	t.Parallel()
	lib := &lltest.TestLoggingLibrary{}
	log := loglayer.New(loglayer.Config{
		Transport:        lltest.New(lltest.Config{Library: lib}),
		DisableFatalExit: true,
	}).SetLevel(loglayer.LogLevelWarn)

	l := slog.New(sloghandler.New(log))
	if l.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("Info should be disabled when underlying logger is at Warn")
	}
	if !l.Enabled(context.Background(), slog.LevelWarn) {
		t.Error("Warn should be enabled")
	}

	l.Info("filtered")
	l.Warn("kept")

	lines := lib.Lines()
	if len(lines) != 1 {
		t.Fatalf("expected 1 emission, got %d", len(lines))
	}
	if lines[0].Messages[0] != "kept" {
		t.Errorf("wrong line: %v", lines[0])
	}
}

func TestPluginPipeline_Runs(t *testing.T) {
	t.Parallel()
	lib := &lltest.TestLoggingLibrary{}
	log := loglayer.New(loglayer.Config{
		Transport:        lltest.New(lltest.Config{Library: lib}),
		DisableFatalExit: true,
	})

	// Inline DataHook plugin: rewrites any string value starting with
	// "secret-" to "[REDACTED]". This used to use the standalone redact
	// plugin, but pulling that in created a cross-module test
	// dependency we don't otherwise need. The test isn't really about
	// redact; it's verifying that values arriving via the slog adapter
	// participate in the plugin pipeline at all.
	log.AddPlugin(loglayer.NewDataHook("test-redactor", func(p loglayer.BeforeDataOutParams) loglayer.Data {
		if p.Data == nil {
			return nil
		}
		out := make(loglayer.Data, len(p.Data))
		for k, v := range p.Data {
			if s, ok := v.(string); ok && strings.HasPrefix(s, "secret-") {
				out[k] = "[REDACTED]"
			} else {
				out[k] = v
			}
		}
		return out
	}))

	l := slog.New(sloghandler.New(log))
	l.Info("auth", "token", "secret-12345")

	line := lib.PopLine()
	if line.Data["token"] != "[REDACTED]" {
		t.Errorf("plugin should rewrite values arriving from slog: got %v", line.Data["token"])
	}
}

func TestContext_ForwardedToTransport(t *testing.T) {
	t.Parallel()
	l, lib, _ := newSlog(t)

	type ctxKey struct{}
	ctx := context.WithValue(context.Background(), ctxKey{}, "marker")
	l.InfoContext(ctx, "hi")

	line := lib.PopLine()
	if line.Ctx == nil {
		t.Fatal("Ctx should be forwarded")
	}
	if got := line.Ctx.Value(ctxKey{}); got != "marker" {
		t.Errorf("ctx value lost: got %v", got)
	}
}

func TestErrorAttr_PassesThroughAsField(t *testing.T) {
	t.Parallel()
	l, lib, _ := newSlog(t)

	l.Info("oops", "err", errors.New("boom"))

	line := lib.PopLine()
	got, ok := line.Data["err"].(error)
	if !ok {
		t.Fatalf("err attr should arrive as error: got %T (%v)", line.Data["err"], line.Data["err"])
	}
	if got.Error() != "boom" {
		t.Errorf("unexpected error message: %v", got)
	}
}

// slog.Record.PC arrives when the slog frontend was constructed with
// AddSource: true (which is the typical production setup). The handler
// should forward it so loglayer renders source under SourceFieldName.
func TestSourceForwardedFromSlogPC(t *testing.T) {
	t.Parallel()
	lib := &lltest.TestLoggingLibrary{}
	log := loglayer.New(loglayer.Config{
		Transport:        lltest.New(lltest.Config{Library: lib}),
		DisableFatalExit: true,
	})

	// slog.New defaults to PC capture; slog.Logger.log() always populates
	// Record.PC via runtime.Callers. So Info() through a slog.Logger
	// backed by our handler should yield a Source that points back here.
	l := slog.New(sloghandler.New(log))
	l.Info("hi") // capture

	line := lib.PopLine()
	src, ok := line.Data["source"].(*loglayer.Source)
	if !ok || src == nil {
		t.Fatalf("expected source forwarded from slog Record.PC, got %v (%T)", line.Data["source"], line.Data["source"])
	}
	if src.File == "" || src.Line == 0 {
		t.Errorf("source should carry file+line, got %+v", src)
	}
	if !strings.Contains(src.Function, "TestSourceForwardedFromSlogPC") {
		t.Errorf("function: got %q, want containing TestSourceForwardedFromSlogPC", src.Function)
	}
}

func TestConcurrentEmission(t *testing.T) {
	t.Parallel()
	l, lib, _ := newSlog(t)

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			scoped := l.With("worker", i).WithGroup("op")
			for j := 0; j < 32; j++ {
				scoped.Info("tick", "j", j)
			}
		}(i)
	}
	wg.Wait()

	if got := lib.Len(); got != 16*32 {
		t.Errorf("expected %d lines, got %d", 16*32, got)
	}
}
