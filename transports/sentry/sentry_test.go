package sentrytransport_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/getsentry/sentry-go"
	"github.com/getsentry/sentry-go/attribute"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
	sentrytransport "go.loglayer.dev/transports/sentry"
)

// fakeLogger satisfies sentry.Logger and records every entry built
// against it. Each Trace/Debug/Info/Warn/Error/LFatal call returns a
// fresh fakeEntry that records attribute writes and the final Emit.
type fakeLogger struct {
	entries []*fakeEntry
	ctx     context.Context
}

type fakeEntry struct {
	level string
	ctx   context.Context
	attrs map[string]any
	msg   string
}

func newFakeLogger() *fakeLogger {
	return &fakeLogger{ctx: context.Background()}
}

func (f *fakeLogger) record(level string) sentry.LogEntry {
	e := &fakeEntry{level: level, ctx: f.ctx, attrs: map[string]any{}}
	f.entries = append(f.entries, e)
	return e
}

func (f *fakeLogger) Trace() sentry.LogEntry  { return f.record("trace") }
func (f *fakeLogger) Debug() sentry.LogEntry  { return f.record("debug") }
func (f *fakeLogger) Info() sentry.LogEntry   { return f.record("info") }
func (f *fakeLogger) Warn() sentry.LogEntry   { return f.record("warn") }
func (f *fakeLogger) Error() sentry.LogEntry  { return f.record("error") }
func (f *fakeLogger) Fatal() sentry.LogEntry  { return f.record("fatal") }
func (f *fakeLogger) Panic() sentry.LogEntry  { return f.record("panic") }
func (f *fakeLogger) LFatal() sentry.LogEntry { return f.record("lfatal") }

func (f *fakeLogger) Write(p []byte) (int, error)        { return len(p), nil }
func (f *fakeLogger) SetAttributes(...attribute.Builder) {}
func (f *fakeLogger) GetCtx() context.Context            { return f.ctx }

// fakeEntry implements sentry.LogEntry. Every typed setter mutates and
// returns the receiver so the loglayer transport's chain works.
func (e *fakeEntry) WithCtx(ctx context.Context) sentry.LogEntry { e.ctx = ctx; return e }

func (e *fakeEntry) String(k, v string) sentry.LogEntry                 { e.attrs[k] = v; return e }
func (e *fakeEntry) Int(k string, v int) sentry.LogEntry                { e.attrs[k] = v; return e }
func (e *fakeEntry) Int64(k string, v int64) sentry.LogEntry            { e.attrs[k] = v; return e }
func (e *fakeEntry) Float64(k string, v float64) sentry.LogEntry        { e.attrs[k] = v; return e }
func (e *fakeEntry) Bool(k string, v bool) sentry.LogEntry              { e.attrs[k] = v; return e }
func (e *fakeEntry) StringSlice(k string, v []string) sentry.LogEntry   { e.attrs[k] = v; return e }
func (e *fakeEntry) Int64Slice(k string, v []int64) sentry.LogEntry     { e.attrs[k] = v; return e }
func (e *fakeEntry) Float64Slice(k string, v []float64) sentry.LogEntry { e.attrs[k] = v; return e }
func (e *fakeEntry) BoolSlice(k string, v []bool) sentry.LogEntry       { e.attrs[k] = v; return e }

func (e *fakeEntry) Emit(args ...interface{}) {
	if len(args) == 1 {
		if s, ok := args[0].(string); ok {
			e.msg = s
			return
		}
	}
	// Fallback for non-string args; tests use the single-string form.
	e.msg = ""
}
func (e *fakeEntry) Emitf(format string, args ...interface{}) { e.msg = format }

func newLogger(t *testing.T, fake *fakeLogger) *loglayer.LogLayer {
	t.Helper()
	tr := sentrytransport.New(sentrytransport.Config{Logger: fake})
	return loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})
}

func TestSentry_Build_RequiresLogger(t *testing.T) {
	_, err := sentrytransport.Build(sentrytransport.Config{})
	if !errors.Is(err, sentrytransport.ErrLoggerRequired) {
		t.Errorf("Build with nil Logger: got %v, want ErrLoggerRequired", err)
	}
}

func TestSentry_New_PanicsWithoutLogger(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic when Logger is nil")
		}
		err, ok := r.(error)
		if !ok || !errors.Is(err, sentrytransport.ErrLoggerRequired) {
			t.Errorf("panic value: got %v, want ErrLoggerRequired", r)
		}
	}()
	_ = sentrytransport.New(sentrytransport.Config{})
}

func TestSentry_LevelMapping(t *testing.T) {
	fake := newFakeLogger()
	log := newLogger(t, fake)

	log.Trace("t")
	log.Debug("d")
	log.Info("i")
	log.Warn("w")
	log.Error("e")
	log.Fatal("f")
	func() {
		defer func() { _ = recover() }()
		log.Panic("p")
	}()

	// LogLevelFatal and LogLevelPanic both map to LFatal so Sentry
	// doesn't double-trigger termination.
	want := []string{"trace", "debug", "info", "warn", "error", "lfatal", "lfatal"}
	got := make([]string, 0, len(fake.entries))
	for _, e := range fake.entries {
		got = append(got, e.level)
	}
	if !slices.Equal(got, want) {
		t.Errorf("level sequence: got %v, want %v", got, want)
	}
}

func TestSentry_MessageJoinedAndEmitted(t *testing.T) {
	fake := newFakeLogger()
	log := newLogger(t, fake)
	log.Info("served", "request")

	if len(fake.entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(fake.entries))
	}
	if got := fake.entries[0].msg; got != "served request" {
		t.Errorf("msg: got %q, want %q", got, "served request")
	}
}

func TestSentry_FieldsAttachAsAttributes(t *testing.T) {
	fake := newFakeLogger()
	log := newLogger(t, fake)
	log = log.WithFields(loglayer.Fields{
		"requestId": "abc",
		"retries":   3,
		"duration":  1.5,
		"flag":      true,
		"tags":      []string{"a", "b"},
	})
	log.Info("hi")

	if len(fake.entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(fake.entries))
	}
	attrs := fake.entries[0].attrs
	if attrs["requestId"] != "abc" {
		t.Errorf("requestId: got %v", attrs["requestId"])
	}
	if attrs["retries"] != 3 {
		t.Errorf("retries: got %v", attrs["retries"])
	}
	if attrs["duration"] != 1.5 {
		t.Errorf("duration: got %v", attrs["duration"])
	}
	if attrs["flag"] != true {
		t.Errorf("flag: got %v", attrs["flag"])
	}
	if !slices.Equal(attrs["tags"].([]string), []string{"a", "b"}) {
		t.Errorf("tags: got %v", attrs["tags"])
	}
}

func TestSentry_ErrorAttachesUnderErrKey(t *testing.T) {
	fake := newFakeLogger()
	log := newLogger(t, fake)
	log.WithError(errors.New("boom")).Error("failed")

	if len(fake.entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(fake.entries))
	}
	// Default ErrorSerializer puts {"message": "boom"} in Data["err"].
	// The transport sees Data["err"] = map[string]any{"message": "boom"},
	// which falls through to the JSON-encoded path (no typed setter for
	// a map value).
	got := fake.entries[0].attrs["err"]
	want := `{"message":"boom"}`
	if got != want {
		t.Errorf("err attr: got %v, want %v", got, want)
	}
}

func TestSentry_MapMetadataFlattens(t *testing.T) {
	fake := newFakeLogger()
	log := newLogger(t, fake)
	log.WithMetadata(loglayer.Metadata{"durationMs": 42, "op": "load"}).Info("did")

	if len(fake.entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(fake.entries))
	}
	attrs := fake.entries[0].attrs
	if attrs["durationMs"] != 42 {
		t.Errorf("durationMs: got %v", attrs["durationMs"])
	}
	if attrs["op"] != "load" {
		t.Errorf("op: got %v", attrs["op"])
	}
}

func TestSentry_NonMapMetadataNestedUnderKey(t *testing.T) {
	fake := newFakeLogger()
	log := newLogger(t, fake)

	type ev struct{ Op string }
	log.WithMetadata(ev{Op: "load"}).Info("did")

	if len(fake.entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(fake.entries))
	}
	got, ok := fake.entries[0].attrs["metadata"].(string)
	if !ok {
		t.Fatalf("metadata: got %T, want string", fake.entries[0].attrs["metadata"])
	}
	if got != `{"Op":"load"}` {
		t.Errorf("metadata: got %q, want %q", got, `{"Op":"load"}`)
	}
}

// When Config.MetadataFieldName is set, both map and non-map metadata
// nest under that key as a JSON-encoded string.
func TestSentry_KeyedMetadata_NestsBothShapes(t *testing.T) {
	cases := []struct {
		name     string
		metadata any
		want     string
	}{
		{"map", loglayer.Metadata{"k": "v"}, `{"k":"v"}`},
		{"struct", struct{ Op string }{Op: "load"}, `{"Op":"load"}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fake := newFakeLogger()
			tr := sentrytransport.New(sentrytransport.Config{Logger: fake})
			log := loglayer.New(loglayer.Config{
				Transport:         tr,
				DisableFatalExit:  true,
				MetadataFieldName: "payload",
			})
			log.WithMetadata(c.metadata).Info("hi")

			if len(fake.entries) != 1 {
				t.Fatalf("expected 1 entry, got %d", len(fake.entries))
			}
			got, ok := fake.entries[0].attrs["payload"].(string)
			if !ok {
				t.Fatalf("payload: got %T, want string", fake.entries[0].attrs["payload"])
			}
			if got != c.want {
				t.Errorf("payload: got %q, want %q", got, c.want)
			}
		})
	}
}

// jsonString falls back to fmt.Sprintf when the value can't be marshalled
// (channels, funcs, etc.) so the dispatch path doesn't drop the value.
func TestSentry_UnmarshallableValueFallsBackToSprintf(t *testing.T) {
	fake := newFakeLogger()
	log := newLogger(t, fake)

	ch := make(chan int)
	log.WithMetadata(ch).Info("hi")

	if len(fake.entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(fake.entries))
	}
	got, ok := fake.entries[0].attrs["metadata"].(string)
	if !ok {
		t.Fatalf("metadata: got %T, want string", fake.entries[0].attrs["metadata"])
	}
	// fmt.Sprintf("%v", chan) renders as "0x..."; just assert non-empty so
	// any future Sprintf change still satisfies the no-silent-drop contract.
	if got == "" {
		t.Errorf("metadata: got empty string, want fmt.Sprintf rendering of the channel")
	}
}

func TestSentry_ContextForwarded(t *testing.T) {
	type ctxKey struct{}
	ctx := context.WithValue(context.Background(), ctxKey{}, "value")

	fake := newFakeLogger()
	log := newLogger(t, fake)
	log.WithContext(ctx).Info("hi")

	if len(fake.entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(fake.entries))
	}
	gotCtx := fake.entries[0].ctx
	if gotCtx == nil {
		t.Fatal("WithCtx not called: ctx is nil")
	}
	if v, _ := gotCtx.Value(ctxKey{}).(string); v != "value" {
		t.Errorf("ctx value: got %q, want value", v)
	}
}

func TestSentry_GetLoggerInstance_ReturnsConfiguredLogger(t *testing.T) {
	fake := newFakeLogger()
	tr := sentrytransport.New(sentrytransport.Config{Logger: fake})
	got := tr.GetLoggerInstance()
	if got != sentry.Logger(fake) {
		t.Errorf("GetLoggerInstance: got %v, want the configured logger", got)
	}
}

func TestSentry_BaseConfigPropagated(t *testing.T) {
	fake := newFakeLogger()
	tr := sentrytransport.New(sentrytransport.Config{
		BaseConfig: transport.BaseConfig{ID: "sentry-test"},
		Logger:     fake,
	})
	if got := tr.ID(); got != "sentry-test" {
		t.Errorf("ID: got %q, want sentry-test", got)
	}
}

func TestSentry_LevelFilteringRespected(t *testing.T) {
	fake := newFakeLogger()
	tr := sentrytransport.New(sentrytransport.Config{
		BaseConfig: transport.BaseConfig{Level: loglayer.LogLevelWarn},
		Logger:     fake,
	})
	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})
	log.Info("dropped")
	log.Warn("kept")

	if len(fake.entries) != 1 {
		t.Fatalf("expected 1 entry under Warn level, got %d", len(fake.entries))
	}
	if fake.entries[0].level != "warn" {
		t.Errorf("level: got %q", fake.entries[0].level)
	}
}
