package oteltrace_test

import (
	"go.loglayer.dev/plugins/oteltrace/v2"
	lltesting "go.loglayer.dev/transports/testing/v2"
	"go.loglayer.dev/v2"
)

// New returns a plugin that injects the active OTel span's trace_id
// and span_id (read from the per-call context attached via WithContext)
// onto every entry. Use it when shipping logs to a non-OTel backend
// that still needs trace correlation.
func ExampleNew() {
	t := lltesting.New(lltesting.Config{})
	log := loglayer.New(loglayer.Config{
		Transport:        t,
		DisableFatalExit: true,
		Plugins:          []loglayer.Plugin{oteltrace.New(oteltrace.Config{})},
	})
	log.Info("served")
}
