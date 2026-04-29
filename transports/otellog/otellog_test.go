package otellog_test

import (
	"context"
	"errors"
	"testing"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
	"go.loglayer.dev/transports/otellog"
	otelapi "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/logtest"
)

func newLogger(t *testing.T, cfg otellog.Config) (*loglayer.LogLayer, *logtest.Recorder) {
	t.Helper()
	rec := logtest.NewRecorder()
	if cfg.Logger == nil && cfg.LoggerProvider == nil {
		cfg.LoggerProvider = rec
	}
	if cfg.Name == "" {
		cfg.Name = "otellog-test"
	}
	if cfg.BaseConfig.ID == "" {
		cfg.BaseConfig.ID = "otellog"
	}
	tr := otellog.New(cfg)
	return loglayer.New(loglayer.Config{
		Transport:        tr,
		DisableFatalExit: true,
	}), rec
}

// lastRecord returns the most recently recorded log entry from the
// recorder's single scope. Fails the test if zero or more than one scope
// is present (the standard test setup uses one Logger per test).
func lastRecord(t *testing.T, rec *logtest.Recorder) logtest.Record {
	t.Helper()
	res := rec.Result()
	if len(res) != 1 {
		t.Fatalf("expected 1 scope in recorder, got %d (%v)", len(res), res)
	}
	for _, recs := range res {
		if len(recs) == 0 {
			t.Fatal("scope has no records")
		}
		return recs[len(recs)-1]
	}
	return logtest.Record{}
}

// kvMap flattens a slice of OTel log key/values into a map for
// unordered lookup. Used by attrs and by tests that walk a nested
// MapValue (Value.AsMap returns []KeyValue).
func kvMap(kvs []otelapi.KeyValue) map[string]otelapi.Value {
	out := make(map[string]otelapi.Value, len(kvs))
	for _, kv := range kvs {
		out[kv.Key] = kv.Value
	}
	return out
}

// attrs flattens a record's attributes for unordered lookup.
func attrs(r logtest.Record) map[string]otelapi.Value { return kvMap(r.Attributes) }

func TestSimpleMessage(t *testing.T) {
	log, rec := newLogger(t, otellog.Config{})
	log.Info("hello")
	r := lastRecord(t, rec)
	if r.Body.AsString() != "hello" {
		t.Errorf("body: got %q", r.Body.AsString())
	}
	if r.Severity != otelapi.SeverityInfo {
		t.Errorf("severity: got %v", r.Severity)
	}
	if r.SeverityText != "info" {
		t.Errorf("severity text: got %q", r.SeverityText)
	}
}

func TestMultipleMessages(t *testing.T) {
	log, rec := newLogger(t, otellog.Config{})
	log.Info("part1", "part2")
	r := lastRecord(t, rec)
	if r.Body.AsString() != "part1 part2" {
		t.Errorf("body: got %q", r.Body.AsString())
	}
}

func TestLevels(t *testing.T) {
	cases := []struct {
		fn  func(*loglayer.LogLayer)
		sev otelapi.Severity
	}{
		{func(l *loglayer.LogLayer) { l.Debug("x") }, otelapi.SeverityDebug},
		{func(l *loglayer.LogLayer) { l.Info("x") }, otelapi.SeverityInfo},
		{func(l *loglayer.LogLayer) { l.Warn("x") }, otelapi.SeverityWarn},
		{func(l *loglayer.LogLayer) { l.Error("x") }, otelapi.SeverityError},
		{func(l *loglayer.LogLayer) { l.Fatal("x") }, otelapi.SeverityFatal},
	}
	for _, c := range cases {
		log, rec := newLogger(t, otellog.Config{})
		c.fn(log)
		if got := lastRecord(t, rec).Severity; got != c.sev {
			t.Errorf("got severity %v, want %v", got, c.sev)
		}
	}
}

func TestMapMetadataFlattens(t *testing.T) {
	log, rec := newLogger(t, otellog.Config{})
	log.WithMetadata(loglayer.Metadata{"requestId": "abc", "n": 42}).Info("req")
	a := attrs(lastRecord(t, rec))
	if a["requestId"].AsString() != "abc" {
		t.Errorf("requestId: got %v", a["requestId"])
	}
	if a["n"].AsInt64() != 42 {
		t.Errorf("n: got %v", a["n"])
	}
}

func TestStructMetadataNested(t *testing.T) {
	type user struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	log, rec := newLogger(t, otellog.Config{})
	log.WithMetadata(user{ID: 7, Name: "Alice"}).Info("hi")
	meta := attrs(lastRecord(t, rec))["metadata"]
	if meta.Kind() != otelapi.KindMap {
		t.Fatalf("metadata should be Map, got %v", meta.Kind())
	}
	got := kvMap(meta.AsMap())
	if got["id"].AsInt64() != 7 || got["name"].AsString() != "Alice" {
		t.Errorf("nested fields: got %v", got)
	}
}

func TestCustomMetadataFieldName(t *testing.T) {
	type user struct{ ID int }
	log, rec := newLogger(t, otellog.Config{MetadataFieldName: "user"})
	log.WithMetadata(user{ID: 9}).Info("hi")
	a := attrs(lastRecord(t, rec))
	if a["user"].Kind() != otelapi.KindMap {
		t.Errorf("expected 'user' map attribute, got %v", a)
	}
}

func TestFieldsMerged(t *testing.T) {
	log, rec := newLogger(t, otellog.Config{})
	log = log.WithFields(loglayer.Fields{"service": "api"})
	log.Info("hello")
	if attrs(lastRecord(t, rec))["service"].AsString() != "api" {
		t.Errorf("service: got %v", attrs(lastRecord(t, rec))["service"])
	}
}

func TestWithError(t *testing.T) {
	log, rec := newLogger(t, otellog.Config{})
	log.WithError(errors.New("boom")).Error("failed")
	errAttr := attrs(lastRecord(t, rec))["err"]
	if errAttr.Kind() != otelapi.KindMap {
		t.Fatalf("expected err nested map, got %v", errAttr.Kind())
	}
	msg, ok := kvMap(errAttr.AsMap())["message"]
	if !ok {
		t.Fatalf("err map missing 'message' key: %v", errAttr.AsMap())
	}
	if msg.AsString() != "boom" {
		t.Errorf("err.message: got %v", msg)
	}
}

func TestWithContextForwarded(t *testing.T) {
	type ctxKey struct{}
	parent := context.WithValue(context.Background(), ctxKey{}, "trace-123")

	log, rec := newLogger(t, otellog.Config{})
	log.WithContext(parent).Info("with ctx")
	got := lastRecord(t, rec).Context
	if got == nil || got.Value(ctxKey{}) != "trace-123" {
		t.Errorf("ctx not forwarded to OTel logger: %v", got)
	}
}

func TestNoCtxBecomesBackground(t *testing.T) {
	log, rec := newLogger(t, otellog.Config{})
	log.Info("plain")
	if lastRecord(t, rec).Context == nil {
		t.Errorf("Emit should always receive a non-nil context")
	}
}

func TestLevelFiltering(t *testing.T) {
	rec := logtest.NewRecorder()
	tr := otellog.New(otellog.Config{
		BaseConfig:     transport.BaseConfig{ID: "otellog", Level: loglayer.LogLevelError},
		Name:           "otellog-test",
		LoggerProvider: rec,
	})
	log := loglayer.New(loglayer.Config{
		Transport:        tr,
		DisableFatalExit: true,
	})
	log.Warn("dropped")
	for _, recs := range rec.Result() {
		if len(recs) != 0 {
			t.Errorf("warn should be filtered, got %v", recs)
		}
	}
	log.Error("kept")
	if r := lastRecord(t, rec); r.Body.AsString() != "kept" {
		t.Errorf("error should pass: got %q", r.Body.AsString())
	}
}

func TestGetLoggerInstance(t *testing.T) {
	rec := logtest.NewRecorder()
	logger := rec.Logger("otellog-test")
	tr := otellog.New(otellog.Config{
		BaseConfig: transport.BaseConfig{ID: "otellog"},
		Logger:     logger,
	})
	log := loglayer.New(loglayer.Config{
		Transport:        tr,
		DisableFatalExit: true,
	})
	if got := log.GetLoggerInstance("otellog"); got != otelapi.Logger(logger) {
		t.Errorf("expected the logger we passed, got %T", got)
	}
}

func TestProviderPath_BuildsLoggerWithOptions(t *testing.T) {
	rec := logtest.NewRecorder()
	tr := otellog.New(otellog.Config{
		Name:           "checkout-api",
		Version:        "1.2.3",
		SchemaURL:      "https://example.test/schemas/v1",
		LoggerProvider: rec,
	})
	log := loglayer.New(loglayer.Config{
		Transport:        tr,
		DisableFatalExit: true,
	})
	log.Info("ok")

	// Drive an entry through and inspect the recorded scope to verify
	// the LoggerProvider received the right instrumentation options.
	res := rec.Result()
	if len(res) != 1 {
		t.Fatalf("expected 1 scope, got %d", len(res))
	}
	for scope, recs := range res {
		if scope.Name != "checkout-api" {
			t.Errorf("scope.Name: got %q", scope.Name)
		}
		if scope.Version != "1.2.3" {
			t.Errorf("scope.Version: got %q", scope.Version)
		}
		if scope.SchemaURL != "https://example.test/schemas/v1" {
			t.Errorf("scope.SchemaURL: got %q", scope.SchemaURL)
		}
		if len(recs) != 1 {
			t.Errorf("expected 1 record, got %d", len(recs))
		}
	}
}

func TestBuild_ErrorsWhenNameMissing(t *testing.T) {
	_, err := otellog.Build(otellog.Config{}) // no Logger, no Name
	if !errors.Is(err, otellog.ErrNameRequired) {
		t.Errorf("got %v, want ErrNameRequired", err)
	}
}

func TestNew_PanicsWhenNameMissing(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when Name is missing and Logger is nil")
		}
	}()
	otellog.New(otellog.Config{})
}

func TestNew_DefaultsToGlobalProvider(t *testing.T) {
	// With no Logger, no LoggerProvider, but a Name set, construction
	// falls through to global.GetLoggerProvider() which returns a no-op
	// when nothing is registered. Construction must succeed and emission
	// must not panic.
	tr := otellog.New(otellog.Config{Name: "any"})
	if tr == nil {
		t.Fatal("expected non-nil transport")
	}
	log := loglayer.New(loglayer.Config{
		Transport:        tr,
		DisableFatalExit: true,
	})
	log.Info("noop")
}

func TestNestedSliceAndMapAttributes(t *testing.T) {
	log, rec := newLogger(t, otellog.Config{})
	log.WithMetadata(loglayer.Metadata{
		"tags":  []any{"a", "b", "c"},
		"inner": map[string]any{"k": 1, "ok": true},
	}).Info("nested")
	a := attrs(lastRecord(t, rec))

	tags := a["tags"]
	if tags.Kind() != otelapi.KindSlice {
		t.Fatalf("tags: kind %v, want KindSlice", tags.Kind())
	}
	tagSlice := tags.AsSlice()
	if len(tagSlice) != 3 || tagSlice[0].AsString() != "a" || tagSlice[2].AsString() != "c" {
		t.Errorf("tags: got %v", tagSlice)
	}
	inner := a["inner"]
	if inner.Kind() != otelapi.KindMap {
		t.Fatalf("inner: kind %v, want KindMap", inner.Kind())
	}
	got := kvMap(inner.AsMap())
	if got["k"].AsInt64() != 1 || got["ok"].AsBool() != true {
		t.Errorf("inner: got %v", got)
	}
}
