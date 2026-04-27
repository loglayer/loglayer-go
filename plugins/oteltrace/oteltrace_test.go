package oteltrace_test

import (
	"context"
	"testing"

	"go.loglayer.dev"
	"go.loglayer.dev/plugins/oteltrace"
	"go.loglayer.dev/transport"
	lltest "go.loglayer.dev/transports/testing"
	otrace "go.opentelemetry.io/otel/trace"
)

// fixedTraceID / fixedSpanID give deterministic hex string outputs we
// can assert against. The hex form is what trace.TraceID.String returns.
var (
	fixedTraceID = otrace.TraceID{0x4b, 0xf9, 0x2f, 0x35, 0x77, 0xb3, 0x4d, 0xa6, 0xa3, 0xce, 0x92, 0x9d, 0x0e, 0x0e, 0x47, 0x36}
	fixedSpanID  = otrace.SpanID{0x00, 0xf0, 0x67, 0xaa, 0x0b, 0xa9, 0x02, 0xb7}
)

const (
	fixedTraceIDHex = "4bf92f3577b34da6a3ce929d0e0e4736"
	fixedSpanIDHex  = "00f067aa0ba902b7"
)

func ctxWithSpan(traceID otrace.TraceID, spanID otrace.SpanID, sampled bool) context.Context {
	cfg := otrace.SpanContextConfig{
		TraceID: traceID,
		SpanID:  spanID,
	}
	if sampled {
		cfg.TraceFlags = otrace.FlagsSampled
	}
	sc := otrace.NewSpanContext(cfg)
	return otrace.ContextWithSpanContext(context.Background(), sc)
}

func setup(t *testing.T, plugin loglayer.Plugin) (*loglayer.LogLayer, *lltest.TestLoggingLibrary) {
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

func TestInjectsIDsWhenSpanPresent(t *testing.T) {
	log, lib := setup(t, oteltrace.New(oteltrace.Config{}))

	log.WithCtx(ctxWithSpan(fixedTraceID, fixedSpanID, true)).Info("served")

	line := lib.PopLine()
	if line.Data["trace_id"] != fixedTraceIDHex {
		t.Errorf("trace_id: got %v, want %s", line.Data["trace_id"], fixedTraceIDHex)
	}
	if line.Data["span_id"] != fixedSpanIDHex {
		t.Errorf("span_id: got %v, want %s", line.Data["span_id"], fixedSpanIDHex)
	}
}

func TestNoCtxNoInjection(t *testing.T) {
	log, lib := setup(t, oteltrace.New(oteltrace.Config{}))

	log.Info("plain") // no WithCtx

	line := lib.PopLine()
	if _, has := line.Data["trace_id"]; has {
		t.Errorf("trace_id should be absent when no Ctx: %v", line.Data)
	}
}

func TestNoSpanNoInjection(t *testing.T) {
	log, lib := setup(t, oteltrace.New(oteltrace.Config{}))

	// Background ctx has no span context attached.
	log.WithCtx(context.Background()).Info("ctx but no span")

	line := lib.PopLine()
	if _, has := line.Data["trace_id"]; has {
		t.Errorf("trace_id should be absent when no span: %v", line.Data)
	}
}

func TestInvalidSpanContextNoInjection(t *testing.T) {
	// Zero-valued TraceID/SpanID make the SpanContext invalid.
	ctx := ctxWithSpan(otrace.TraceID{}, otrace.SpanID{}, false)
	log, lib := setup(t, oteltrace.New(oteltrace.Config{}))

	log.WithCtx(ctx).Info("invalid span")

	line := lib.PopLine()
	if _, has := line.Data["trace_id"]; has {
		t.Errorf("trace_id should be absent for invalid span context: %v", line.Data)
	}
}

func TestCustomKeys(t *testing.T) {
	log, lib := setup(t, oteltrace.New(oteltrace.Config{
		TraceIDKey: "trace.id",
		SpanIDKey:  "span.id",
	}))

	log.WithCtx(ctxWithSpan(fixedTraceID, fixedSpanID, true)).Info("ecs-style")

	line := lib.PopLine()
	if line.Data["trace.id"] != fixedTraceIDHex {
		t.Errorf("trace.id: got %v", line.Data["trace.id"])
	}
	if line.Data["span.id"] != fixedSpanIDHex {
		t.Errorf("span.id: got %v", line.Data["span.id"])
	}
	// Default keys should NOT also appear.
	if _, has := line.Data["trace_id"]; has {
		t.Errorf("default trace_id should not appear when TraceIDKey is overridden")
	}
}

func TestTraceFlagsEmittedWhenConfigured(t *testing.T) {
	log, lib := setup(t, oteltrace.New(oteltrace.Config{
		TraceFlagsKey: "trace_flags",
	}))

	log.WithCtx(ctxWithSpan(fixedTraceID, fixedSpanID, true)).Info("sampled")

	line := lib.PopLine()
	if line.Data["trace_flags"] != int(otrace.FlagsSampled) {
		t.Errorf("trace_flags: got %v, want %d", line.Data["trace_flags"], int(otrace.FlagsSampled))
	}
}

func TestTraceFlagsOmittedByDefault(t *testing.T) {
	log, lib := setup(t, oteltrace.New(oteltrace.Config{}))

	log.WithCtx(ctxWithSpan(fixedTraceID, fixedSpanID, true)).Info("no-flags")

	line := lib.PopLine()
	if _, has := line.Data["trace_flags"]; has {
		t.Errorf("trace_flags should be absent when TraceFlagsKey is empty: %v", line.Data["trace_flags"])
	}
}

func TestPreservesUserData(t *testing.T) {
	log, lib := setup(t, oteltrace.New(oteltrace.Config{}))
	log = log.WithFields(loglayer.Fields{"requestId": "abc-123"})

	log.WithCtx(ctxWithSpan(fixedTraceID, fixedSpanID, true)).
		WithMetadata(loglayer.Metadata{"durationMs": 42}).
		Info("served")

	line := lib.PopLine()
	if line.Data["requestId"] != "abc-123" {
		t.Errorf("user field should pass through: %v", line.Data)
	}
	if line.Data["trace_id"] != fixedTraceIDHex {
		t.Errorf("trace_id: got %v", line.Data["trace_id"])
	}
	if m, _ := line.Metadata.(loglayer.Metadata); m["durationMs"] != 42 {
		t.Errorf("metadata should pass through: %v", line.Metadata)
	}
}

func TestDefaultID(t *testing.T) {
	p := oteltrace.New(oteltrace.Config{})
	if p.ID != "otel-trace-injector" {
		t.Errorf("default ID: got %q, want %q", p.ID, "otel-trace-injector")
	}
}

func TestCustomID(t *testing.T) {
	p := oteltrace.New(oteltrace.Config{ID: "my-injector"})
	if p.ID != "my-injector" {
		t.Errorf("custom ID: got %q", p.ID)
	}
}
