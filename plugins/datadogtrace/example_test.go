package datadogtrace_test

import (
	"context"

	"go.loglayer.dev"
	"go.loglayer.dev/plugins/datadogtrace"
	lltesting "go.loglayer.dev/transports/testing"
)

// New returns a plugin that injects dd.trace_id and dd.span_id onto
// every entry. Extract is required; wire it to whichever Datadog
// tracer your service uses (dd-trace-go v1, v2, or an OTel bridge).
func ExampleNew() {
	t := lltesting.New(lltesting.Config{})
	log := loglayer.New(loglayer.Config{
		Transport:        t,
		DisableFatalExit: true,
		Plugins: []loglayer.Plugin{
			datadogtrace.New(datadogtrace.Config{
				Service: "checkout-api",
				Env:     "production",
				Extract: func(ctx context.Context) (uint64, uint64, bool) {
					// Replace with ddtracer.SpanFromContext(ctx) etc.
					return 0, 0, false
				},
			}),
		},
	})
	log.Info("served")
}
