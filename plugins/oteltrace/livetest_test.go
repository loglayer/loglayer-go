//go:build livetest

// Live integration test for the oteltrace plugin. Spins up a real
// OpenTelemetry TracerProvider and starts real spans, then asserts the
// plugin emits the SDK-produced trace and span IDs in canonical hex
// form.
//
// Run:
//
//	go test -tags=livetest ./plugins/oteltrace/
package oteltrace_test

import (
	"context"
	"testing"

	"go.loglayer.dev"
	"go.loglayer.dev/plugins/oteltrace"
	lltest "go.loglayer.dev/transports/testing"
	"go.opentelemetry.io/otel/baggage"
	otrace "go.opentelemetry.io/otel/trace"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func newLiveSetup(t *testing.T, plugin loglayer.Plugin, tpOpts ...sdktrace.TracerProviderOption) (*loglayer.LogLayer, *lltest.TestLoggingLibrary, otrace.Tracer) {
	t.Helper()
	log, lib := setup(t, plugin)
	tp := sdktrace.NewTracerProvider(tpOpts...)
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	return log, lib, tp.Tracer("livetest")
}

func TestLive_RealSpanInjectsRealIDs(t *testing.T) {
	log, lib, tracer := newLiveSetup(t, oteltrace.New(oteltrace.Config{}))

	ctx, span := tracer.Start(context.Background(), "op")
	log.WithCtx(ctx).Info("served")
	span.End()

	line := lib.PopLine()
	wantTraceID := span.SpanContext().TraceID().String()
	wantSpanID := span.SpanContext().SpanID().String()
	if line.Data["trace_id"] != wantTraceID {
		t.Errorf("trace_id: got %v, want %s", line.Data["trace_id"], wantTraceID)
	}
	if line.Data["span_id"] != wantSpanID {
		t.Errorf("span_id: got %v, want %s", line.Data["span_id"], wantSpanID)
	}
}

func TestLive_NestedSpansEachInjectOwnIDs(t *testing.T) {
	log, lib, tracer := newLiveSetup(t, oteltrace.New(oteltrace.Config{}))

	ctxOuter, outer := tracer.Start(context.Background(), "outer")
	log.WithCtx(ctxOuter).Info("outer entry")
	outerLine := lib.PopLine()

	ctxInner, inner := tracer.Start(ctxOuter, "inner")
	log.WithCtx(ctxInner).Info("inner entry")
	innerLine := lib.PopLine()

	inner.End()
	outer.End()

	if outerLine.Data["span_id"] != outer.SpanContext().SpanID().String() {
		t.Errorf("outer span_id mismatch: got %v, want %s",
			outerLine.Data["span_id"], outer.SpanContext().SpanID().String())
	}
	if innerLine.Data["span_id"] != inner.SpanContext().SpanID().String() {
		t.Errorf("inner span_id mismatch: got %v, want %s",
			innerLine.Data["span_id"], inner.SpanContext().SpanID().String())
	}
	// Both spans share the trace ID (inner is a child of outer).
	if outerLine.Data["trace_id"] != innerLine.Data["trace_id"] {
		t.Errorf("nested spans should share trace_id; outer=%v, inner=%v",
			outerLine.Data["trace_id"], innerLine.Data["trace_id"])
	}
	// And the IDs differ between spans.
	if outerLine.Data["span_id"] == innerLine.Data["span_id"] {
		t.Errorf("nested spans should have different span_ids: %v", outerLine.Data["span_id"])
	}
}

func TestLive_SampledFlagEmitted(t *testing.T) {
	// Default TracerProvider uses ParentBased(AlwaysSample), so root
	// spans are sampled.
	log, lib, tracer := newLiveSetup(t, oteltrace.New(oteltrace.Config{
		TraceFlagsKey: "trace_flags",
	}))

	ctx, span := tracer.Start(context.Background(), "op")
	log.WithCtx(ctx).Info("sampled")
	span.End()

	line := lib.PopLine()
	flags, ok := line.Data["trace_flags"].(int)
	if !ok {
		t.Fatalf("trace_flags should be int, got %T (%v)", line.Data["trace_flags"], line.Data["trace_flags"])
	}
	// Bit 0 is the sampled flag.
	if flags&int(otrace.FlagsSampled) == 0 {
		t.Errorf("trace_flags should have the sampled bit set; got %d", flags)
	}
}

func TestLive_NeverSampleProducesValidIDs(t *testing.T) {
	// NeverSample still produces valid trace/span IDs; the sampled bit
	// just isn't set. The plugin should still emit the IDs since the
	// span context is valid.
	log, lib, tracer := newLiveSetup(t,
		oteltrace.New(oteltrace.Config{TraceFlagsKey: "trace_flags"}),
		sdktrace.WithSampler(sdktrace.NeverSample()),
	)

	ctx, span := tracer.Start(context.Background(), "op")
	log.WithCtx(ctx).Info("not sampled")
	span.End()

	line := lib.PopLine()
	if line.Data["trace_id"] != span.SpanContext().TraceID().String() {
		t.Errorf("trace_id should be emitted even when not sampled: got %v", line.Data["trace_id"])
	}
	flags, _ := line.Data["trace_flags"].(int)
	if flags&int(otrace.FlagsSampled) != 0 {
		t.Errorf("sampled bit should be unset under NeverSample; got flags=%d", flags)
	}
}

func TestLive_TraceStatePropagatesFromParent(t *testing.T) {
	// SDK tracers propagate W3C trace state from parent to child spans.
	// Build a parent SpanContext with trace state set, then start a real
	// child span; the plugin should read the inherited trace state.
	log, lib, tracer := newLiveSetup(t, oteltrace.New(oteltrace.Config{
		TraceStateKey: "trace_state",
	}))

	parentState, err := otrace.ParseTraceState("vendor1=val1,vendor2=val2")
	if err != nil {
		t.Fatalf("ParseTraceState: %v", err)
	}
	parentSC := otrace.NewSpanContext(otrace.SpanContextConfig{
		TraceID:    otrace.TraceID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10},
		SpanID:     otrace.SpanID{0xa1, 0xa2, 0xa3, 0xa4, 0xa5, 0xa6, 0xa7, 0xa8},
		TraceFlags: otrace.FlagsSampled,
		TraceState: parentState,
		Remote:     true,
	})
	parentCtx := otrace.ContextWithSpanContext(context.Background(), parentSC)

	ctx, span := tracer.Start(parentCtx, "child")
	log.WithCtx(ctx).Info("traced")
	span.End()

	line := lib.PopLine()
	if line.Data["trace_state"] != "vendor1=val1,vendor2=val2" {
		t.Errorf("trace_state: got %v, want %q",
			line.Data["trace_state"], "vendor1=val1,vendor2=val2")
	}
}

func TestLive_BaggageEmitted(t *testing.T) {
	log, lib, tracer := newLiveSetup(t, oteltrace.New(oteltrace.Config{
		BaggageKeyPrefix: "baggage.",
	}))

	user, err := baggage.NewMember("user_id", "alice")
	if err != nil {
		t.Fatalf("NewMember: %v", err)
	}
	tenant, err := baggage.NewMember("tenant_id", "acme")
	if err != nil {
		t.Fatalf("NewMember: %v", err)
	}
	bag, err := baggage.New(user, tenant)
	if err != nil {
		t.Fatalf("New baggage: %v", err)
	}
	ctx := baggage.ContextWithBaggage(context.Background(), bag)

	ctx, span := tracer.Start(ctx, "op")
	log.WithCtx(ctx).Info("served")
	span.End()

	line := lib.PopLine()
	if line.Data["baggage.user_id"] != "alice" {
		t.Errorf("baggage.user_id: got %v", line.Data["baggage.user_id"])
	}
	if line.Data["baggage.tenant_id"] != "acme" {
		t.Errorf("baggage.tenant_id: got %v", line.Data["baggage.tenant_id"])
	}
	// Trace IDs also present (we started a real span on the bagged ctx).
	if _, has := line.Data["trace_id"]; !has {
		t.Errorf("trace_id should be present alongside baggage: %v", line.Data)
	}
}
