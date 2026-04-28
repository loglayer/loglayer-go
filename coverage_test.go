package loglayer_test

// Tests filling coverage gaps in critical paths: every LogBuilder terminal
// level method, level parsing/stringification, and edge cases for prefix +
// unknown levels.

import (
	"context"
	"errors"
	"testing"

	"go.loglayer.dev"
)

func TestBuild_NoTransport(t *testing.T) {
	_, err := loglayer.Build(loglayer.Config{})
	if !errors.Is(err, loglayer.ErrNoTransport) {
		t.Errorf("got %v, want ErrNoTransport", err)
	}
}

func TestBuild_WithTransport(t *testing.T) {
	log, err := loglayer.Build(loglayer.Config{Transport: discardTransport{}})
	if err != nil || log == nil {
		t.Errorf("Build with transport: err=%v log=%v", err, log)
	}
}

func TestNew_PanicsWithoutTransport(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic with no transport")
		}
	}()
	loglayer.New(loglayer.Config{})
}

func TestBuild_TransportAndTransports_Errors(t *testing.T) {
	_, err := loglayer.Build(loglayer.Config{
		Transport:  discardTransport{},
		Transports: []loglayer.Transport{discardTransport{}},
	})
	if !errors.Is(err, loglayer.ErrTransportAndTransports) {
		t.Errorf("got %v, want ErrTransportAndTransports", err)
	}
}

func TestNew_TransportAndTransports_Panics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic when both Transport and Transports are set")
		}
		err, ok := r.(error)
		if !ok || !errors.Is(err, loglayer.ErrTransportAndTransports) {
			t.Errorf("panic value: got %v, want ErrTransportAndTransports", r)
		}
	}()
	loglayer.New(loglayer.Config{
		Transport:  discardTransport{},
		Transports: []loglayer.Transport{discardTransport{}},
	})
}

type discardTransport struct{}

func (discardTransport) ID() string                              { return "discard" }
func (discardTransport) IsEnabled() bool                         { return true }
func (discardTransport) SendToLogger(_ loglayer.TransportParams) {}
func (discardTransport) GetLoggerInstance() any                  { return nil }

func TestWithCtx_PassesThroughToTransport(t *testing.T) {
	log, lib := setup(t)
	type ctxKey struct{}
	ctx := context.WithValue(context.Background(), ctxKey{}, "trace-abc")

	log.WithCtx(ctx).Info("with ctx")
	line := lib.PopLine()
	if line == nil || line.Ctx == nil {
		t.Fatal("expected Ctx to be set on captured line")
	}
	if got := line.Ctx.Value(ctxKey{}); got != "trace-abc" {
		t.Errorf("ctx value not preserved: got %v", got)
	}
}

func TestWithCtx_BuilderChain(t *testing.T) {
	log, lib := setup(t)
	ctx := context.Background()

	log.WithCtx(ctx).
		WithMetadata(loglayer.Metadata{"k": "v"}).
		WithError(errors.New("boom")).
		Error("chained")

	line := lib.PopLine()
	if line == nil {
		t.Fatal("expected line")
	}
	if line.Ctx != ctx {
		t.Errorf("Ctx mismatch: got %v", line.Ctx)
	}
	if line.Data["err"] == nil {
		t.Errorf("err missing: %v", line.Data)
	}
	if m, _ := line.Metadata.(loglayer.Metadata); m["k"] != "v" {
		t.Errorf("metadata missing: %v", line.Metadata)
	}
}

func TestWithCtx_Raw(t *testing.T) {
	log, lib := setup(t)
	ctx := context.Background()
	log.Raw(loglayer.RawLogEntry{
		LogLevel: loglayer.LogLevelInfo,
		Messages: []any{"raw with ctx"},
		Ctx:      ctx,
	})
	line := lib.PopLine()
	if line == nil || line.Ctx != ctx {
		t.Errorf("Raw should propagate Ctx: got %v", line)
	}
}

// Raw with no messages still emits a line with the configured level and
// metadata. Edge case: empty Messages slice should not panic and should
// not be silently dropped.
func TestRaw_EmptyMessages(t *testing.T) {
	log, lib := setup(t)
	log.Raw(loglayer.RawLogEntry{
		LogLevel: loglayer.LogLevelWarn,
		Messages: []any{},
		Metadata: map[string]any{"k": "v"},
	})
	line := lib.PopLine()
	if line == nil {
		t.Fatal("Raw with empty messages should still emit")
	}
	if line.Level != loglayer.LogLevelWarn {
		t.Errorf("level: got %s, want warn", line.Level)
	}
	if len(line.Messages) != 0 {
		t.Errorf("expected empty messages, got %v", line.Messages)
	}
	if m, _ := line.Metadata.(map[string]any); m["k"] != "v" {
		t.Errorf("metadata should still flow: %v", line.Metadata)
	}
}

func TestWithoutCtx_NilOnTransport(t *testing.T) {
	log, lib := setup(t)
	log.Info("no ctx attached")
	line := lib.PopLine()
	if line == nil {
		t.Fatal("expected line")
	}
	if line.Ctx != nil {
		t.Errorf("Ctx should be nil when not attached: got %v", line.Ctx)
	}
}

// (*LogLayer).WithCtx binds a context to all subsequent emissions from
// the returned logger. Tests the persistent-ctx semantics added to
// distinguish from per-call (*LogBuilder).WithCtx.
func TestWithCtx_BindsToLogger(t *testing.T) {
	log, lib := setup(t)
	type ctxKey struct{}
	ctx := context.WithValue(context.Background(), ctxKey{}, "trace-xyz")

	bound := log.WithCtx(ctx)
	bound.Info("first")
	bound.Warn("second")
	bound.WithMetadata(loglayer.Metadata{"k": "v"}).Info("third")

	for i, want := range []string{"first", "second", "third"} {
		line := lib.PopLine()
		if line == nil {
			t.Fatalf("line %d (%q): missing", i, want)
		}
		if line.Ctx == nil {
			t.Fatalf("line %d (%q): bound ctx not propagated", i, want)
		}
		if got := line.Ctx.Value(ctxKey{}); got != "trace-xyz" {
			t.Errorf("line %d (%q): ctx value: got %v", i, want, got)
		}
	}
}

// Per-call (*LogBuilder).WithCtx still overrides the bound ctx for that
// emission only. The bound ctx applies again for subsequent emissions.
func TestWithCtx_PerCallOverridesBound(t *testing.T) {
	log, lib := setup(t)
	type ctxKey struct{}
	bound := context.WithValue(context.Background(), ctxKey{}, "BOUND")
	override := context.WithValue(context.Background(), ctxKey{}, "OVERRIDE")

	logger := log.WithCtx(bound)

	logger.Info("uses bound")
	logger.WithCtx(override).Info("uses override")
	logger.Info("back to bound")

	want := []string{"BOUND", "OVERRIDE", "BOUND"}
	for i, expect := range want {
		line := lib.PopLine()
		got, _ := line.Ctx.Value(ctxKey{}).(string)
		if got != expect {
			t.Errorf("line %d: got ctx value %q, want %q", i, got, expect)
		}
	}
}

// WithCtx returns a derived logger; the receiver's behavior is unchanged.
func TestWithCtx_ReceiverUnchanged(t *testing.T) {
	log, lib := setup(t)
	ctx := context.Background()
	_ = log.WithCtx(ctx)

	log.Info("from base logger") // should not have any ctx attached
	line := lib.PopLine()
	if line.Ctx != nil {
		t.Errorf("base logger should not have ctx after WithCtx call: got %v", line.Ctx)
	}
}

// Child() inherits the parent's bound ctx; later mutations on either
// side don't bleed.
func TestWithCtx_ChildInheritsAndIsolates(t *testing.T) {
	log, lib := setup(t)
	type ctxKey struct{}
	ctx := context.WithValue(context.Background(), ctxKey{}, "parent")

	parent := log.WithCtx(ctx)
	child := parent.Child()

	child.Info("inherits parent's bound ctx")
	if got := lib.PopLine().Ctx.Value(ctxKey{}); got != "parent" {
		t.Errorf("child should inherit parent's bound ctx: got %v", got)
	}

	other := context.WithValue(context.Background(), ctxKey{}, "rebound")
	rebound := child.WithCtx(other)
	rebound.Info("child rebinds")
	if got := lib.PopLine().Ctx.Value(ctxKey{}); got != "rebound" {
		t.Errorf("child rebind: got %v", got)
	}

	parent.Info("parent unaffected by child rebind")
	if got := lib.PopLine().Ctx.Value(ctxKey{}); got != "parent" {
		t.Errorf("parent's bound ctx should be unchanged: got %v", got)
	}
}

// Passing nil to WithCtx clears any bound ctx (returns a logger with
// no ctx bound).
func TestWithCtx_NilClears(t *testing.T) {
	log, lib := setup(t)
	bound := log.WithCtx(context.Background())
	//lint:ignore SA1012 nil is intentional: test the clears-binding path.
	cleared := bound.WithCtx(nil)

	cleared.Info("no ctx")
	line := lib.PopLine()
	if line.Ctx != nil {
		t.Errorf("WithCtx(nil) should clear bound ctx: got %v", line.Ctx)
	}
}

func TestLogBuilder_AllTerminals(t *testing.T) {
	cases := []struct {
		name  string
		fn    func(*loglayer.LogLayer)
		level loglayer.LogLevel
	}{
		{"Debug", func(l *loglayer.LogLayer) { l.WithMetadata(loglayer.Metadata{"k": "v"}).Debug("msg") }, loglayer.LogLevelDebug},
		{"Info", func(l *loglayer.LogLayer) { l.WithMetadata(loglayer.Metadata{"k": "v"}).Info("msg") }, loglayer.LogLevelInfo},
		{"Warn", func(l *loglayer.LogLayer) { l.WithMetadata(loglayer.Metadata{"k": "v"}).Warn("msg") }, loglayer.LogLevelWarn},
		{"Error", func(l *loglayer.LogLayer) { l.WithMetadata(loglayer.Metadata{"k": "v"}).Error("msg") }, loglayer.LogLevelError},
		{"Fatal", func(l *loglayer.LogLayer) { l.WithMetadata(loglayer.Metadata{"k": "v"}).Fatal("msg") }, loglayer.LogLevelFatal},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			log, lib := setup(t)
			c.fn(log)
			line := lib.PopLine()
			if line == nil {
				t.Fatalf("%s: expected line", c.name)
			}
			if line.Level != c.level {
				t.Errorf("%s: level got %s, want %s", c.name, line.Level, c.level)
			}
			m, ok := line.Metadata.(loglayer.Metadata)
			if !ok || m["k"] != "v" {
				t.Errorf("%s: metadata not preserved through builder: %+v", c.name, line.Metadata)
			}
		})
	}
}

func TestLogBuilder_LevelFiltered_NoEntry(t *testing.T) {
	log, lib := setup(t)
	log.SetLevel(loglayer.LogLevelError)
	log.WithMetadata(loglayer.Metadata{"x": 1}).Info("dropped")
	log.WithError(errors.New("e")).Warn("dropped")
	if lib.Len() != 0 {
		t.Errorf("expected no captured lines (level filtered), got %d", lib.Len())
	}
}

// TestFAndMAliases pins that loglayer.F is interchangeable with
// loglayer.Fields and loglayer.M with loglayer.Metadata. The aliases
// exist for shorter call sites; they MUST stay aliases (not distinct
// types) so existing code that passes loglayer.Fields keeps working.
func TestFAndMAliases(t *testing.T) {
	log, lib := setup(t)
	log = log.WithFields(loglayer.F{"requestId": "abc"})
	log.WithMetadata(loglayer.M{"duration": 150}).Info("done")

	line := lib.PopLine()
	if line == nil {
		t.Fatal("expected a captured line")
	}
	if got, _ := line.Data["requestId"].(string); got != "abc" {
		t.Errorf("F should land in Fields path: got %v", line.Data)
	}
	m, _ := line.Metadata.(loglayer.Metadata)
	if m["duration"] != 150 {
		t.Errorf("M should land in Metadata path: got %v", line.Metadata)
	}
}

func TestLogLevel_String(t *testing.T) {
	cases := map[loglayer.LogLevel]string{
		loglayer.LogLevelDebug: "debug",
		loglayer.LogLevelInfo:  "info",
		loglayer.LogLevelWarn:  "warn",
		loglayer.LogLevelError: "error",
		loglayer.LogLevelFatal: "fatal",
		loglayer.LogLevel(999): "unknown",
	}
	for level, want := range cases {
		if got := level.String(); got != want {
			t.Errorf("LogLevel(%d).String(): got %q, want %q", level, got, want)
		}
	}
}

func TestParseLogLevel(t *testing.T) {
	cases := []struct {
		in   string
		want loglayer.LogLevel
		ok   bool
	}{
		{"debug", loglayer.LogLevelDebug, true},
		{"info", loglayer.LogLevelInfo, true},
		{"warn", loglayer.LogLevelWarn, true},
		{"error", loglayer.LogLevelError, true},
		{"fatal", loglayer.LogLevelFatal, true},
		{"INFO", loglayer.LogLevelInfo, false}, // case-sensitive
		{"", loglayer.LogLevelInfo, false},
		{"unknown", loglayer.LogLevelInfo, false},
	}
	for _, c := range cases {
		got, ok := loglayer.ParseLogLevel(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("ParseLogLevel(%q) = (%s, %v), want (%s, %v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestLevelState_UnknownLevelIsNoop(t *testing.T) {
	log, lib := setup(t)
	// Unknown levels should not panic and should be safe.
	log.EnableLevel(loglayer.LogLevel(0))
	log.DisableLevel(loglayer.LogLevel(999))
	log.SetLevel(loglayer.LogLevel(-5)) // should not blow up
	// Standard levels should still work after weird calls.
	log.Info("after weird levels")
	if lib.Len() != 1 {
		t.Errorf("expected 1 line after edge-case level changes, got %d", lib.Len())
	}
}

func TestEnableLevel_AfterSetLevel(t *testing.T) {
	log, lib := setup(t)
	log.SetLevel(loglayer.LogLevelError)
	log.Info("dropped")
	log.EnableLevel(loglayer.LogLevelInfo)
	log.Info("emitted")
	if lib.Len() != 1 {
		t.Errorf("expected 1 line after re-enabling info, got %d", lib.Len())
	}
}

func TestUnmuteMetadata(t *testing.T) {
	log, lib := setup(t)
	log.MuteMetadata()
	log.WithMetadata(loglayer.Metadata{"a": 1}).Info("muted")
	first := lib.PopLine()
	if first == nil || first.Metadata != nil {
		t.Errorf("muted metadata should be nil, got %+v", first)
	}
	log.UnmuteMetadata()
	log.WithMetadata(loglayer.Metadata{"a": 1}).Info("unmuted")
	second := lib.PopLine()
	if second == nil || second.Metadata == nil {
		t.Errorf("unmuted metadata should be present, got %+v", second)
	}
}

func TestPrefix_NonStringFirstArg(t *testing.T) {
	log, lib := setupWithConfig(t, loglayer.Config{Prefix: "[app]"})
	// Non-string first arg should be left alone (no prefix prepended).
	log.Info(42, "context")
	line := lib.PopLine()
	if line == nil || line.Messages[0] != 42 {
		t.Errorf("non-string first arg should be untouched, got %+v", line.Messages)
	}
}

func TestPrefix_EmptyMessages(t *testing.T) {
	log, lib := setupWithConfig(t, loglayer.Config{Prefix: "[app]"})
	// Empty messages slice should not panic.
	log.Info()
	line := lib.PopLine()
	if line == nil {
		t.Fatal("expected entry even with empty messages")
	}
	if len(line.Messages) != 0 {
		t.Errorf("expected empty messages, got %+v", line.Messages)
	}
}
