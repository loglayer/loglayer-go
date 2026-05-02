// Demonstrates loglayer's OpenTelemetry pieces end-to-end:
//
//   - transports/otellog ships log entries through the OTel logs pipeline
//     so they land alongside traces and metrics in the same exporter chain.
//   - plugins/oteltrace surfaces trace_id/span_id as flat fields on every
//     entry that carries an active span, so log/trace correlation works
//     even on transports that don't speak OTel.
//
// The example wires both an OTel logs and trace pipeline to stdout
// exporters so it's runnable without a real collector. You'll see two
// kinds of stdout output interleaved: OTel-formatted log records (from
// the logs pipeline) and OTel-formatted span records (from the trace
// pipeline). The flat-JSON-style trace_id/span_id from the plugin only
// matters for transports that aren't otellog; here both run side by
// side so you can compare.
//
// Run from the repo root:
//
//	go run ./examples/otel-end-to-end
package main

import (
	"context"
	"fmt"
	"os"

	"go.loglayer.dev/plugins/oteltrace/v2"
	"go.loglayer.dev/transports/otellog/v2"
	"go.loglayer.dev/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func main() {
	ctx := context.Background()

	// --- OTel logs pipeline (so transports/otellog has somewhere to ship to)
	logExporter, err := stdoutlog.New(stdoutlog.WithPrettyPrint())
	if err != nil {
		fail("stdoutlog: %v", err)
	}
	logProvider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
	)
	defer func() { _ = logProvider.Shutdown(ctx) }()

	// --- OTel trace pipeline (so the plugin and the SDK both have a real
	// span context to read from). Default sampler is AlwaysSample for
	// root spans.
	traceExporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		fail("stdouttrace: %v", err)
	}
	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
	)
	defer func() { _ = traceProvider.Shutdown(ctx) }()
	otel.SetTracerProvider(traceProvider)

	// --- Build the LogLayer logger with both an OTel transport and the
	// trace-injector plugin. The plugin is redundant alongside otellog
	// (the SDK already correlates), but we include it so the example
	// shows what its attributes look like.
	tr := otellog.New(otellog.Config{
		Name:           "otel-end-to-end-example",
		Version:        "0.0.1",
		LoggerProvider: logProvider,
	})
	plugin := oteltrace.New(oteltrace.Config{
		TraceFlagsKey:    "trace_flags",
		BaggageKeyPrefix: "baggage.",
	})
	log := loglayer.New(loglayer.Config{
		Transport: tr,
		Plugins:   []loglayer.Plugin{plugin},
	})

	// --- Emit some logs, both inside and outside a span.
	log.Info("server starting")

	tracer := otel.Tracer("otel-end-to-end-example")
	ctxOuter, outer := tracer.Start(ctx, "handle-request")
	requestLog := log.WithContext(ctxOuter)
	requestLog.WithFields(loglayer.Fields{"path": "/checkout"}).
		Info("request received")

	ctxInner, inner := tracer.Start(ctxOuter, "db-query")
	requestLog.WithContext(ctxInner).
		WithMetadata(loglayer.Metadata{"durationMs": 47}).
		Info("query completed")
	inner.End()

	requestLog.Info("request served")
	outer.End()

	// Drain both batchers so the deferred Shutdown calls don't lose
	// in-flight records. ForceFlush is the deterministic version of
	// "wait until everything is exported."
	_ = logProvider.ForceFlush(ctx)
	_ = traceProvider.ForceFlush(ctx)
	fmt.Println("\n(end of example: shutdown flushes remaining batches)")
}

func fail(msg string, args ...any) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
	os.Exit(1)
}
