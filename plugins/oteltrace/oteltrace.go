// Package oteltrace provides a LogLayer plugin that injects OpenTelemetry
// trace and span IDs into log entries, enabling log/trace correlation
// for any transport (structured, HTTP, Datadog, etc.).
//
// Use this plugin when your service runs OpenTelemetry tracing but you
// ship logs to a destination other than the OTel logs pipeline. The
// plugin reads the active span from each entry's per-call context (the
// one attached via WithContext) and emits trace_id and span_id, plus an
// optional trace_flags, as fields on the entry. Downstream UIs (Grafana,
// Tempo, Honeycomb, custom dashboards) can use these to link a log line
// back to its trace.
//
// If you DO ship logs to the OTel pipeline via transports/otellog, you
// don't need this plugin — the OTel SDK reads the active span from the
// emission context and embeds the trace IDs on the exported log.Record
// automatically. Use this plugin when the destination doesn't speak OTel.
//
// Usage:
//
//	import (
//	    "go.loglayer.dev/v2"
//	    "go.loglayer.dev/plugins/oteltrace/v2"
//	    "go.loglayer.dev/transports/structured/v2"
//	)
//
//	log := loglayer.New(loglayer.Config{
//	    Transport: structured.New(structured.Config{}),
//	    Plugins:   []loglayer.Plugin{oteltrace.New(oteltrace.Config{})},
//	})
//
//	// Inside a handler whose context carries an OTel span:
//	handlerLog := log.WithContext(r.Context())
//	handlerLog.Info("served")
//	// {"level":"info","msg":"served","trace_id":"4bf...","span_id":"00f..."}
//
// See https://go.loglayer.dev for usage guides and the full API reference.
package oteltrace

import (
	"go.loglayer.dev/v2"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/trace"
)

// Config holds plugin configuration.
type Config struct {
	// ID for the plugin. Defaults to "otel-trace-injector".
	ID string

	// TraceIDKey is the data key under which the trace ID is emitted.
	// Defaults to "trace_id" (matching OTel JSON serialization). Common
	// alternatives: "trace.id" (ECS), "traceId" (camelCase backends).
	TraceIDKey string

	// SpanIDKey is the data key under which the span ID is emitted.
	// Defaults to "span_id".
	SpanIDKey string

	// TraceFlagsKey, when non-empty, also emits the trace flags byte
	// (an int 0-255; bit 0 is the sampled flag) under that key. Empty
	// means "don't emit" (the default).
	TraceFlagsKey string

	// TraceStateKey, when non-empty, emits the W3C trace state (vendor-
	// specific routing/sampling information that rides with the trace
	// context) as a single string under that key. Canonical form:
	// "vendor1=val1,vendor2=val2". Empty means "don't emit" (the default).
	// Skipped when the trace state is empty even if the key is set.
	TraceStateKey string

	// BaggageKeyPrefix, when non-empty, emits each W3C baggage member
	// from p.Ctx as a separate attribute keyed "<prefix><member-key>".
	// Common choice: "baggage." → baggage.user_id, baggage.tenant_id.
	// Empty means "don't emit" (the default).
	//
	// Baggage is read independently of the trace span: a context can
	// carry baggage with no active span, in which case those values are
	// emitted on the entry without trace_id / span_id.
	BaggageKeyPrefix string

	// OnError is forwarded to the resulting Plugin's OnError. The
	// LogLayer framework recovers extractor panics centrally; if you
	// pass a non-nil OnError here, you'll see the recovered panic.
	// Defaults to silent.
	OnError func(err error)
}

// New constructs the plugin. The returned plugin implements
// [loglayer.DataHook]; when cfg.OnError is set it's wrapped via
// [loglayer.WithErrorReporter] so recovered hook panics reach the
// caller-supplied callback instead of the framework's stderr default.
func New(cfg Config) loglayer.Plugin {
	if cfg.ID == "" {
		cfg.ID = "otel-trace-injector"
	}
	if cfg.TraceIDKey == "" {
		cfg.TraceIDKey = "trace_id"
	}
	if cfg.SpanIDKey == "" {
		cfg.SpanIDKey = "span_id"
	}
	return loglayer.WithErrorReporter(&plugin{cfg: cfg}, cfg.OnError)
}

type plugin struct {
	cfg Config
}

func (p *plugin) ID() string { return p.cfg.ID }

func (p *plugin) OnBeforeDataOut(bp loglayer.BeforeDataOutParams) loglayer.Data {
	if bp.Ctx == nil {
		return nil
	}
	var data loglayer.Data
	if sc := trace.SpanContextFromContext(bp.Ctx); sc.IsValid() {
		data = loglayer.Data{
			p.cfg.TraceIDKey: sc.TraceID().String(),
			p.cfg.SpanIDKey:  sc.SpanID().String(),
		}
		if p.cfg.TraceFlagsKey != "" {
			data[p.cfg.TraceFlagsKey] = int(sc.TraceFlags())
		}
		if p.cfg.TraceStateKey != "" {
			if ts := sc.TraceState(); ts.Len() > 0 {
				data[p.cfg.TraceStateKey] = ts.String()
			}
		}
	}
	if p.cfg.BaggageKeyPrefix != "" {
		if bag := baggage.FromContext(bp.Ctx); bag.Len() > 0 {
			if data == nil {
				data = make(loglayer.Data, bag.Len())
			}
			for _, m := range bag.Members() {
				data[p.cfg.BaggageKeyPrefix+m.Key()] = m.Value()
			}
		}
	}
	return data
}
