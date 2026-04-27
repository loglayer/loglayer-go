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
