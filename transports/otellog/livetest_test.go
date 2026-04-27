//go:build livetest

// Live integration tests for the otellog transport. These exercise the
// real OpenTelemetry SDK end-to-end: a real LoggerProvider with an
// in-memory Exporter for the log signal, and a real TracerProvider for
// the span context that drives log/trace correlation.
//
// Run:
//
//	go test -tags=livetest ./transports/otellog/
//
// The default `go test ./...` skips this file because of the build tag,
// keeping unit tests fast and dependency-light.
package otellog_test

import (
	"context"
	"sync"
	"testing"

	"go.loglayer.dev"
	"go.loglayer.dev/transports/otellog"
	otelapi "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// memExporter captures records exported by the SDK so tests can inspect
// what actually crossed the SDK boundary. The SDK reuses its records
// slice between calls, so we Clone each one per the Exporter contract.
type memExporter struct {
	mu      sync.Mutex
	records []sdklog.Record
}

func (e *memExporter) Export(_ context.Context, recs []sdklog.Record) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	for i := range recs {
		e.records = append(e.records, recs[i].Clone())
	}
	return nil
}

func (e *memExporter) Shutdown(_ context.Context) error   { return nil }
func (e *memExporter) ForceFlush(_ context.Context) error { return nil }

func (e *memExporter) all() []sdklog.Record {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]sdklog.Record, len(e.records))
	copy(out, e.records)
	return out
}

// sdkAttrs flattens an SDK record's attributes into a map for unordered
// lookup. The SDK's Record only exposes attributes via WalkAttributes
// (no slice accessor like the API/logtest record types).
func sdkAttrs(r sdklog.Record) map[string]otelapi.Value {
	out := make(map[string]otelapi.Value, r.AttributesLen())
	r.WalkAttributes(func(kv otelapi.KeyValue) bool {
		out[kv.Key] = kv.Value
		return true
	})
	return out
}

// newSDKLog builds a real LoggerProvider wired to an in-memory exporter
// via SimpleProcessor (synchronous; emission lands in the exporter
// before Emit returns).
func newSDKLog(t *testing.T) (*sdklog.LoggerProvider, *memExporter) {
	t.Helper()
	exp := &memExporter{}
	provider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewSimpleProcessor(exp)),
	)
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })
	return provider, exp
}

func TestLive_BasicEmissionThroughSDK(t *testing.T) {
	provider, exp := newSDKLog(t)
	tr := otellog.New(otellog.Config{
		Name:           "livetest",
		Version:        "0.0.1",
		LoggerProvider: provider,
	})
	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})

	log.WithFields(loglayer.Fields{"k": "v"}).Info("hello")

	got := exp.all()
	if len(got) != 1 {
		t.Fatalf("expected 1 exported record, got %d", len(got))
	}
	rec := got[0]
	if rec.Body().AsString() != "hello" {
		t.Errorf("body: got %q", rec.Body().AsString())
	}
	if rec.Severity() != otelapi.SeverityInfo {
		t.Errorf("severity: got %v, want SeverityInfo", rec.Severity())
	}
	if rec.SeverityText() != "info" {
		t.Errorf("severityText: got %q", rec.SeverityText())
	}

	scope := rec.InstrumentationScope()
	if scope.Name != "livetest" {
		t.Errorf("scope.Name: got %q", scope.Name)
	}
	if scope.Version != "0.0.1" {
		t.Errorf("scope.Version: got %q", scope.Version)
	}

	if got := sdkAttrs(rec)["k"].AsString(); got != "v" {
		t.Errorf("attr k: got %q", got)
	}
}

func TestLive_TraceCorrelation(t *testing.T) {
	// Real TracerProvider: default sampler is ParentBased(AlwaysSample),
	// so root spans get sampled and carry valid IDs.
	tp := sdktrace.NewTracerProvider()
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	tracer := tp.Tracer("livetest")

	provider, exp := newSDKLog(t)
	tr := otellog.New(otellog.Config{Name: "livetest", LoggerProvider: provider})
	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})

	ctx, span := tracer.Start(context.Background(), "op")
	log.WithCtx(ctx).Info("traced")
	span.End()

	got := exp.all()
	if len(got) != 1 {
		t.Fatalf("expected 1 exported record, got %d", len(got))
	}
	rec := got[0]

	wantTraceID := span.SpanContext().TraceID()
	wantSpanID := span.SpanContext().SpanID()
	if rec.TraceID() != wantTraceID {
		t.Errorf("trace_id: got %v, want %v", rec.TraceID(), wantTraceID)
	}
	if rec.SpanID() != wantSpanID {
		t.Errorf("span_id: got %v, want %v", rec.SpanID(), wantSpanID)
	}
	if !rec.TraceFlags().IsSampled() {
		t.Errorf("expected sampled flag set, got %v", rec.TraceFlags())
	}
}

func TestLive_NoCtxNoTraceCorrelation(t *testing.T) {
	provider, exp := newSDKLog(t)
	tr := otellog.New(otellog.Config{Name: "livetest", LoggerProvider: provider})
	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})

	log.Info("untraced") // no WithCtx

	got := exp.all()
	if len(got) != 1 {
		t.Fatalf("expected 1 record, got %d", len(got))
	}
	if got[0].TraceID().IsValid() {
		t.Errorf("trace ID should be zero when no ctx is bound: %v", got[0].TraceID())
	}
}

func TestLive_StructMetadataNestedThroughSDK(t *testing.T) {
	provider, exp := newSDKLog(t)
	tr := otellog.New(otellog.Config{Name: "livetest", LoggerProvider: provider})
	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})

	type user struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	log.WithMetadata(user{ID: 7, Name: "Alice"}).Info("hi")

	got := exp.all()
	if len(got) != 1 {
		t.Fatalf("expected 1 record, got %d", len(got))
	}
	meta := sdkAttrs(got[0])["metadata"]
	if meta.Kind() != otelapi.KindMap {
		t.Fatalf("metadata attr should be Map, got %v", meta.Kind())
	}
	nested := kvMap(meta.AsMap())
	// JSON roundtrip turns Go ints into float64.
	if nested["id"].AsFloat64() != 7 || nested["name"].AsString() != "Alice" {
		t.Errorf("metadata fields: got %v", nested)
	}
}

func TestLive_AllSeverities(t *testing.T) {
	cases := []struct {
		fn  func(*loglayer.LogLayer)
		sev otelapi.Severity
	}{
		{func(l *loglayer.LogLayer) { l.Trace("x") }, otelapi.SeverityTrace},
		{func(l *loglayer.LogLayer) { l.Debug("x") }, otelapi.SeverityDebug},
		{func(l *loglayer.LogLayer) { l.Info("x") }, otelapi.SeverityInfo},
		{func(l *loglayer.LogLayer) { l.Warn("x") }, otelapi.SeverityWarn},
		{func(l *loglayer.LogLayer) { l.Error("x") }, otelapi.SeverityError},
		{func(l *loglayer.LogLayer) { l.Fatal("x") }, otelapi.SeverityFatal},
	}
	for _, c := range cases {
		provider, exp := newSDKLog(t)
		tr := otellog.New(otellog.Config{Name: "livetest", LoggerProvider: provider})
		log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})
		c.fn(log)
		got := exp.all()
		if len(got) != 1 || got[0].Severity() != c.sev {
			t.Errorf("severity: got %v, want %v", got, c.sev)
		}
	}
}
