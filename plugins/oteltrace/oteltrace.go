// Package oteltrace provides a LogLayer plugin that injects OpenTelemetry
// trace and span IDs into log entries, enabling log/trace correlation
// for any transport (structured, HTTP, Datadog, etc.).
//
// Use this plugin when your service runs OpenTelemetry tracing but you
// ship logs to a destination other than the OTel logs pipeline. The
// plugin reads the active span from each entry's per-call context (the
// one attached via WithCtx) and emits trace_id and span_id, plus an
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
//	    "go.loglayer.dev"
//	    "go.loglayer.dev/plugins/oteltrace"
//	    "go.loglayer.dev/transports/structured"
//	)
//
//	log := loglayer.New(loglayer.Config{
//	    Transport: structured.New(structured.Config{}),
//	    Plugins:   []loglayer.Plugin{oteltrace.New(oteltrace.Config{})},
//	})
//
//	// Inside a handler whose context carries an OTel span:
//	handlerLog := log.WithCtx(r.Context())
//	handlerLog.Info("served")
//	// {"level":"info","msg":"served","trace_id":"4bf...","span_id":"00f..."}
package oteltrace

import (
	"go.loglayer.dev"
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

	// OnError is forwarded to the resulting Plugin's OnError. The
	// LogLayer framework recovers extractor panics centrally; if you
	// pass a non-nil OnError here, you'll see the recovered panic.
	// Defaults to silent.
	OnError func(err error)
}

// New constructs the plugin.
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

	return loglayer.Plugin{
		ID:      cfg.ID,
		OnError: cfg.OnError,
		OnBeforeDataOut: func(p loglayer.BeforeDataOutParams) loglayer.Data {
			if p.Ctx == nil {
				return nil
			}
			sc := trace.SpanContextFromContext(p.Ctx)
			if !sc.IsValid() {
				return nil
			}
			data := loglayer.Data{
				cfg.TraceIDKey: sc.TraceID().String(),
				cfg.SpanIDKey:  sc.SpanID().String(),
			}
			if cfg.TraceFlagsKey != "" {
				data[cfg.TraceFlagsKey] = int(sc.TraceFlags())
			}
			return data
		},
	}
}
