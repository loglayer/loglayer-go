package oteltrace_test

import (
	"context"
	"testing"

	"go.loglayer.dev/plugins/oteltrace/v2"
	"go.loglayer.dev/plugins/plugintest/v2"
	"go.loglayer.dev/v2"
	"go.opentelemetry.io/otel/baggage"
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

func TestInjectsIDsWhenSpanPresent(t *testing.T) {
	t.Parallel()
	log, lib := plugintest.Install(t, oteltrace.New(oteltrace.Config{}))

	log.WithContext(ctxWithSpan(fixedTraceID, fixedSpanID, true)).Info("served")

	line := lib.PopLine()
	if line.Data["trace_id"] != fixedTraceIDHex {
		t.Errorf("trace_id: got %v, want %s", line.Data["trace_id"], fixedTraceIDHex)
	}
	if line.Data["span_id"] != fixedSpanIDHex {
		t.Errorf("span_id: got %v, want %s", line.Data["span_id"], fixedSpanIDHex)
	}
}

func TestNoCtxNoInjection(t *testing.T) {
	t.Parallel()
	log, lib := plugintest.Install(t, oteltrace.New(oteltrace.Config{}))

	log.Info("plain") // no WithContext

	line := lib.PopLine()
	if _, has := line.Data["trace_id"]; has {
		t.Errorf("trace_id should be absent when no Ctx: %v", line.Data)
	}
}

func TestNoSpanNoInjection(t *testing.T) {
	t.Parallel()
	log, lib := plugintest.Install(t, oteltrace.New(oteltrace.Config{}))

	// Background ctx has no span context attached.
	log.WithContext(context.Background()).Info("ctx but no span")

	line := lib.PopLine()
	if _, has := line.Data["trace_id"]; has {
		t.Errorf("trace_id should be absent when no span: %v", line.Data)
	}
}

func TestInvalidSpanContextNoInjection(t *testing.T) {
	t.Parallel()
	// Zero-valued TraceID/SpanID make the SpanContext invalid.
	ctx := ctxWithSpan(otrace.TraceID{}, otrace.SpanID{}, false)
	log, lib := plugintest.Install(t, oteltrace.New(oteltrace.Config{}))

	log.WithContext(ctx).Info("invalid span")

	line := lib.PopLine()
	if _, has := line.Data["trace_id"]; has {
		t.Errorf("trace_id should be absent for invalid span context: %v", line.Data)
	}
}

func TestCustomKeys(t *testing.T) {
	t.Parallel()
	log, lib := plugintest.Install(t, oteltrace.New(oteltrace.Config{
		TraceIDKey: "trace.id",
		SpanIDKey:  "span.id",
	}))

	log.WithContext(ctxWithSpan(fixedTraceID, fixedSpanID, true)).Info("ecs-style")

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
	t.Parallel()
	log, lib := plugintest.Install(t, oteltrace.New(oteltrace.Config{
		TraceFlagsKey: "trace_flags",
	}))

	log.WithContext(ctxWithSpan(fixedTraceID, fixedSpanID, true)).Info("sampled")

	line := lib.PopLine()
	if line.Data["trace_flags"] != int(otrace.FlagsSampled) {
		t.Errorf("trace_flags: got %v, want %d", line.Data["trace_flags"], int(otrace.FlagsSampled))
	}
}

func TestTraceFlagsOmittedByDefault(t *testing.T) {
	t.Parallel()
	log, lib := plugintest.Install(t, oteltrace.New(oteltrace.Config{}))

	log.WithContext(ctxWithSpan(fixedTraceID, fixedSpanID, true)).Info("no-flags")

	line := lib.PopLine()
	if _, has := line.Data["trace_flags"]; has {
		t.Errorf("trace_flags should be absent when TraceFlagsKey is empty: %v", line.Data["trace_flags"])
	}
}

func TestPreservesUserData(t *testing.T) {
	t.Parallel()
	log, lib := plugintest.Install(t, oteltrace.New(oteltrace.Config{}))
	log = log.WithFields(loglayer.Fields{"requestId": "abc-123"})

	log.WithContext(ctxWithSpan(fixedTraceID, fixedSpanID, true)).
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
	t.Parallel()
	p := oteltrace.New(oteltrace.Config{})
	if p.ID() != "otel-trace-injector" {
		t.Errorf("default ID: got %q, want %q", p.ID(), "otel-trace-injector")
	}
}

func TestCustomID(t *testing.T) {
	t.Parallel()
	p := oteltrace.New(oteltrace.Config{ID: "my-injector"})
	if p.ID() != "my-injector" {
		t.Errorf("custom ID: got %q", p.ID())
	}
}

// ctxWithSpanAndState attaches a span context with a parsed W3C
// TraceState. Construction goes through ParseTraceState so the value
// stays in canonical form on String().
func ctxWithSpanAndState(t *testing.T, ts string) context.Context {
	t.Helper()
	parsed, err := otrace.ParseTraceState(ts)
	if err != nil {
		t.Fatalf("ParseTraceState(%q): %v", ts, err)
	}
	sc := otrace.NewSpanContext(otrace.SpanContextConfig{
		TraceID:    fixedTraceID,
		SpanID:     fixedSpanID,
		TraceFlags: otrace.FlagsSampled,
		TraceState: parsed,
	})
	return otrace.ContextWithSpanContext(context.Background(), sc)
}

func TestTraceStateEmittedWhenConfigured(t *testing.T) {
	t.Parallel()
	log, lib := plugintest.Install(t, oteltrace.New(oteltrace.Config{
		TraceStateKey: "trace_state",
	}))

	log.WithContext(ctxWithSpanAndState(t, "vendor1=val1,vendor2=val2")).Info("hi")

	line := lib.PopLine()
	if line.Data["trace_state"] != "vendor1=val1,vendor2=val2" {
		t.Errorf("trace_state: got %v, want %q", line.Data["trace_state"], "vendor1=val1,vendor2=val2")
	}
}

func TestTraceStateOmittedByDefault(t *testing.T) {
	t.Parallel()
	log, lib := plugintest.Install(t, oteltrace.New(oteltrace.Config{}))

	log.WithContext(ctxWithSpanAndState(t, "vendor1=val1")).Info("hi")

	line := lib.PopLine()
	if _, has := line.Data["trace_state"]; has {
		t.Errorf("trace_state should be absent when TraceStateKey is empty: %v", line.Data["trace_state"])
	}
}

func TestTraceStateOmittedWhenEmpty(t *testing.T) {
	t.Parallel()
	// Key is configured but the trace state itself is empty: emit nothing.
	log, lib := plugintest.Install(t, oteltrace.New(oteltrace.Config{
		TraceStateKey: "trace_state",
	}))

	log.WithContext(ctxWithSpan(fixedTraceID, fixedSpanID, true)).Info("hi") // no trace state

	line := lib.PopLine()
	if _, has := line.Data["trace_state"]; has {
		t.Errorf("trace_state should be absent when state is empty: %v", line.Data["trace_state"])
	}
}

func TestBaggageEmittedWithPrefix(t *testing.T) {
	t.Parallel()
	log, lib := plugintest.Install(t, oteltrace.New(oteltrace.Config{
		BaggageKeyPrefix: "baggage.",
	}))

	user, _ := baggage.NewMember("user_id", "alice")
	tenant, _ := baggage.NewMember("tenant_id", "acme")
	bag, _ := baggage.New(user, tenant)
	ctx := baggage.ContextWithBaggage(ctxWithSpan(fixedTraceID, fixedSpanID, true), bag)

	log.WithContext(ctx).Info("hi")

	line := lib.PopLine()
	if line.Data["baggage.user_id"] != "alice" {
		t.Errorf("baggage.user_id: got %v, want %q", line.Data["baggage.user_id"], "alice")
	}
	if line.Data["baggage.tenant_id"] != "acme" {
		t.Errorf("baggage.tenant_id: got %v, want %q", line.Data["baggage.tenant_id"], "acme")
	}
}

func TestBaggageOmittedByDefault(t *testing.T) {
	t.Parallel()
	log, lib := plugintest.Install(t, oteltrace.New(oteltrace.Config{}))

	user, _ := baggage.NewMember("user_id", "alice")
	bag, _ := baggage.New(user)
	ctx := baggage.ContextWithBaggage(ctxWithSpan(fixedTraceID, fixedSpanID, true), bag)

	log.WithContext(ctx).Info("hi")

	line := lib.PopLine()
	for k := range line.Data {
		if len(k) > 8 && k[:8] == "baggage." {
			t.Errorf("baggage.* keys should not appear: %v", line.Data)
		}
	}
}

func TestBaggageEmittedWithoutSpan(t *testing.T) {
	t.Parallel()
	// Baggage rides independently of the trace span: a context with
	// baggage but no span should still surface baggage attributes.
	log, lib := plugintest.Install(t, oteltrace.New(oteltrace.Config{
		BaggageKeyPrefix: "baggage.",
	}))

	feature, _ := baggage.NewMember("feature_flag", "checkout-v2")
	bag, _ := baggage.New(feature)
	ctx := baggage.ContextWithBaggage(context.Background(), bag)

	log.WithContext(ctx).Info("no span")

	line := lib.PopLine()
	if line.Data["baggage.feature_flag"] != "checkout-v2" {
		t.Errorf("baggage.feature_flag: got %v", line.Data["baggage.feature_flag"])
	}
	// Trace IDs absent because the span context is invalid.
	if _, has := line.Data["trace_id"]; has {
		t.Errorf("trace_id should be absent without a span: %v", line.Data)
	}
}

func TestNoSpanNoBaggageNoInjection(t *testing.T) {
	t.Parallel()
	log, lib := plugintest.Install(t, oteltrace.New(oteltrace.Config{
		BaggageKeyPrefix: "baggage.",
	}))

	// Background ctx: no span, no baggage. Nothing to emit.
	log.WithContext(context.Background()).Info("plain")

	line := lib.PopLine()
	if _, has := line.Data["trace_id"]; has {
		t.Errorf("trace_id should be absent: %v", line.Data)
	}
	for k := range line.Data {
		if len(k) > 8 && k[:8] == "baggage." {
			t.Errorf("baggage.* should be absent: %v", line.Data)
		}
	}
}
