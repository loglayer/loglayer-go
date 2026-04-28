// Package datadogtrace provides a LogLayer plugin that injects Datadog
// APM trace and span IDs into log entries, enabling Datadog's
// log/trace correlation feature.
//
// The plugin reads the active span via a user-supplied Extract function
// from the per-call context.Context (attached via WithCtx). It then
// emits dd.trace_id, dd.span_id, plus the optional dd.service /
// dd.env / dd.version reserved attributes.
//
// The plugin does not depend on any specific Datadog tracer library:
// the user wires up the extractor for whichever version of dd-trace-go
// (v1, v2, or an OTel-bridged tracer) they already use. This keeps
// LogLayer's dependency footprint minimal for users who don't need
// Datadog APM correlation.
//
// Usage with dd-trace-go v1:
//
//	import (
//	    "context"
//	    ddtracer "gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
//	    "go.loglayer.dev"
//	    "go.loglayer.dev/plugins/datadogtrace"
//	    "go.loglayer.dev/transports/structured"
//	)
//
//	ddtracer.Start(ddtracer.WithService("checkout-api"))
//	defer ddtracer.Stop()
//
//	log := loglayer.New(loglayer.Config{
//	    Transport: structured.New(structured.Config{}),
//	    Plugins: []loglayer.Plugin{
//	        datadogtrace.New(datadogtrace.Config{
//	            Service: "checkout-api",
//	            Env:     "production",
//	            Extract: func(ctx context.Context) (uint64, uint64, bool) {
//	                if span, ok := ddtracer.SpanFromContext(ctx); ok {
//	                    sc := span.Context()
//	                    return sc.TraceID(), sc.SpanID(), true
//	                }
//	                return 0, 0, false
//	            },
//	        }),
//	    },
//	})
//
//	// Inside an HTTP handler whose context carries a span, bind once:
//	handlerLog := log.WithCtx(r.Context())
//	handlerLog.Info("served")
//	// Output includes dd.trace_id and dd.span_id, ready for log/trace
//	// correlation in the Datadog UI.
package datadogtrace

import (
	"context"
	"strconv"

	"go.loglayer.dev"
)

// Config holds plugin configuration.
type Config struct {
	// ID for the plugin. Defaults to "datadog-trace-injector".
	ID string

	// Extract returns the active span's trace and span IDs from the
	// per-call context. Returns ok=false when no span is attached or
	// when the IDs aren't yet available.
	//
	// Required. Wire it to whatever tracer library your service uses.
	// See the package doc for the standard dd-trace-go v1 wiring.
	Extract func(ctx context.Context) (traceID, spanID uint64, ok bool)

	// Service, Env, Version are optional Datadog reserved attributes
	// emitted on every entry as dd.service / dd.env / dd.version.
	// Set them once here to avoid duplicating them in every log call.
	// Empty values are omitted from the output.
	Service string
	Env     string
	Version string

	// OnError is forwarded to the resulting Plugin's OnError. The
	// LogLayer framework recovers extractor panics centrally; if you
	// pass a non-nil OnError here, you'll see the recovered panic.
	// Defaults to silent.
	OnError func(err error)
}

// New constructs the plugin. Panics if cfg.Extract is nil. The returned
// plugin implements [loglayer.DataHook]; when cfg.OnError is set it's
// wrapped via [loglayer.WithErrorReporter] so recovered hook panics
// reach the caller-supplied callback instead of the framework's stderr
// default.
func New(cfg Config) loglayer.Plugin {
	if cfg.Extract == nil {
		panic("loglayer/plugins/datadogtrace: Config.Extract is required")
	}
	if cfg.ID == "" {
		cfg.ID = "datadog-trace-injector"
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
	traceID, spanID, ok := p.cfg.Extract(bp.Ctx)
	if !ok {
		return nil
	}
	// Datadog log/trace correlation expects decimal-string IDs.
	data := loglayer.Data{
		"dd.trace_id": strconv.FormatUint(traceID, 10),
		"dd.span_id":  strconv.FormatUint(spanID, 10),
	}
	if p.cfg.Service != "" {
		data["dd.service"] = p.cfg.Service
	}
	if p.cfg.Env != "" {
		data["dd.env"] = p.cfg.Env
	}
	if p.cfg.Version != "" {
		data["dd.version"] = p.cfg.Version
	}
	return data
}
