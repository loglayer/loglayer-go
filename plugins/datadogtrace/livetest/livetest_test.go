// Live integration test for the datadogtrace plugin against the real
// dd-trace-go v2 tracer (via its in-process mocktracer). Validates that
// the documented extractor pattern produces IDs the plugin emits in the
// format Datadog ingestion expects (decimal-string uint64).
//
// This test lives in its own module so dd-trace-go's heavy transitive
// closure stays out of the main loglayer module. Run from this dir:
//
//	cd plugins/datadogtrace/livetest && go test ./...
//
// Or from the repo root:
//
//	go test ./plugins/datadogtrace/livetest/...
package livetest_test

import (
	"context"
	"strconv"
	"testing"

	"go.loglayer.dev/plugins/datadogtrace/v2"
	lltest "go.loglayer.dev/transports/testing/v2"
	"go.loglayer.dev/v2"
	"go.loglayer.dev/v2/transport"

	ddmocktracer "github.com/DataDog/dd-trace-go/v2/ddtrace/mocktracer"
	ddtracer "github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
)

// extractDDv2 is the canonical extractor pattern for dd-trace-go v2.
// v2's SpanContext.TraceID returns the full hex trace ID as a string;
// for log/trace correlation Datadog wants the lower 64 bits as a
// decimal string, which TraceIDLower returns directly.
func extractDDv2(ctx context.Context) (uint64, uint64, bool) {
	span, ok := ddtracer.SpanFromContext(ctx)
	if !ok {
		return 0, 0, false
	}
	sc := span.Context()
	return sc.TraceIDLower(), sc.SpanID(), true
}

func newLiveSetup(t *testing.T, plugin loglayer.Plugin) (*loglayer.LogLayer, *lltest.TestLoggingLibrary) {
	t.Helper()
	lib := &lltest.TestLoggingLibrary{}
	tr := lltest.New(lltest.Config{BaseConfig: transport.BaseConfig{ID: "test"}, Library: lib})
	log := loglayer.New(loglayer.Config{
		Transport:        tr,
		DisableFatalExit: true,
		Plugins:          []loglayer.Plugin{plugin},
	})
	return log, lib
}

func TestLive_RealDDSpanInjectsRealIDs(t *testing.T) {
	mt := ddmocktracer.Start()
	t.Cleanup(mt.Stop)

	log, lib := newLiveSetup(t, datadogtrace.New(datadogtrace.Config{
		Extract: extractDDv2,
	}))

	span, ctx := ddtracer.StartSpanFromContext(context.Background(), "operation")
	log.WithContext(ctx).Info("served")
	span.Finish()

	line := lib.PopLine()
	sc := span.Context()
	wantTraceID := strconv.FormatUint(sc.TraceIDLower(), 10)
	wantSpanID := strconv.FormatUint(sc.SpanID(), 10)
	if line.Data["dd.trace_id"] != wantTraceID {
		t.Errorf("dd.trace_id: got %v, want %s", line.Data["dd.trace_id"], wantTraceID)
	}
	if line.Data["dd.span_id"] != wantSpanID {
		t.Errorf("dd.span_id: got %v, want %s", line.Data["dd.span_id"], wantSpanID)
	}
}

func TestLive_NestedDDSpansEachInjectOwnIDs(t *testing.T) {
	mt := ddmocktracer.Start()
	t.Cleanup(mt.Stop)

	log, lib := newLiveSetup(t, datadogtrace.New(datadogtrace.Config{
		Extract: extractDDv2,
	}))

	outerSpan, ctxOuter := ddtracer.StartSpanFromContext(context.Background(), "outer")
	log.WithContext(ctxOuter).Info("outer entry")
	outerLine := lib.PopLine()

	innerSpan, ctxInner := ddtracer.StartSpanFromContext(ctxOuter, "inner")
	log.WithContext(ctxInner).Info("inner entry")
	innerLine := lib.PopLine()

	innerSpan.Finish()
	outerSpan.Finish()

	wantOuter := strconv.FormatUint(outerSpan.Context().SpanID(), 10)
	wantInner := strconv.FormatUint(innerSpan.Context().SpanID(), 10)

	if outerLine.Data["dd.span_id"] != wantOuter {
		t.Errorf("outer span_id: got %v, want %s", outerLine.Data["dd.span_id"], wantOuter)
	}
	if innerLine.Data["dd.span_id"] != wantInner {
		t.Errorf("inner span_id: got %v, want %s", innerLine.Data["dd.span_id"], wantInner)
	}
	if outerLine.Data["dd.trace_id"] != innerLine.Data["dd.trace_id"] {
		t.Errorf("nested spans should share dd.trace_id; outer=%v, inner=%v",
			outerLine.Data["dd.trace_id"], innerLine.Data["dd.trace_id"])
	}
	if outerLine.Data["dd.span_id"] == innerLine.Data["dd.span_id"] {
		t.Errorf("nested spans should have distinct dd.span_ids: %v", outerLine.Data["dd.span_id"])
	}
}

func TestLive_NoSpanOmitsIDs(t *testing.T) {
	mt := ddmocktracer.Start()
	t.Cleanup(mt.Stop)

	log, lib := newLiveSetup(t, datadogtrace.New(datadogtrace.Config{
		Extract: extractDDv2,
	}))

	// Background ctx has no active span; extractor returns ok=false.
	log.WithContext(context.Background()).Info("no span")

	line := lib.PopLine()
	if _, has := line.Data["dd.trace_id"]; has {
		t.Errorf("dd.trace_id should be absent: %v", line.Data)
	}
	if _, has := line.Data["dd.span_id"]; has {
		t.Errorf("dd.span_id should be absent: %v", line.Data)
	}
}

func TestLive_ServiceEnvVersionEmitted(t *testing.T) {
	mt := ddmocktracer.Start()
	t.Cleanup(mt.Stop)

	log, lib := newLiveSetup(t, datadogtrace.New(datadogtrace.Config{
		Extract: extractDDv2,
		Service: "checkout-api",
		Env:     "prod",
		Version: "1.2.3",
	}))

	span, ctx := ddtracer.StartSpanFromContext(context.Background(), "op")
	log.WithContext(ctx).Info("served")
	span.Finish()

	line := lib.PopLine()
	if line.Data["dd.service"] != "checkout-api" {
		t.Errorf("dd.service: got %v", line.Data["dd.service"])
	}
	if line.Data["dd.env"] != "prod" {
		t.Errorf("dd.env: got %v", line.Data["dd.env"])
	}
	if line.Data["dd.version"] != "1.2.3" {
		t.Errorf("dd.version: got %v", line.Data["dd.version"])
	}
}

func TestLive_DecimalIDsMatchDatadogIngestionFormat(t *testing.T) {
	mt := ddmocktracer.Start()
	t.Cleanup(mt.Stop)

	log, lib := newLiveSetup(t, datadogtrace.New(datadogtrace.Config{
		Extract: extractDDv2,
	}))

	span, ctx := ddtracer.StartSpanFromContext(context.Background(), "op")
	log.WithContext(ctx).Info("served")
	span.Finish()

	line := lib.PopLine()
	traceID, ok := line.Data["dd.trace_id"].(string)
	if !ok {
		t.Fatalf("dd.trace_id should be a string, got %T (%v)", line.Data["dd.trace_id"], line.Data["dd.trace_id"])
	}
	// Datadog log/trace correlation expects pure-decimal uint64 strings.
	if _, err := strconv.ParseUint(traceID, 10, 64); err != nil {
		t.Errorf("dd.trace_id should parse as decimal uint64: %q (%v)", traceID, err)
	}
	spanID, _ := line.Data["dd.span_id"].(string)
	if _, err := strconv.ParseUint(spanID, 10, 64); err != nil {
		t.Errorf("dd.span_id should parse as decimal uint64: %q (%v)", spanID, err)
	}
}
