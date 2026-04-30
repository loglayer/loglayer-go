package otellog_test

import (
	"go.loglayer.dev"
	"go.loglayer.dev/transports/otellog"
)

// New emits log entries to an OpenTelemetry log.Logger. Name is the
// instrumentation scope (required when LoggerProvider/Logger are nil);
// the transport falls back to the global LoggerProvider, which is a
// no-op until the OTel SDK registers a real provider.
func ExampleNew() {
	t := otellog.New(otellog.Config{
		Name:    "checkout-api",
		Version: "1.2.3",
	})
	log := loglayer.New(loglayer.Config{
		Transport:        t,
		DisableFatalExit: true,
	})
	log.Info("served")
}
