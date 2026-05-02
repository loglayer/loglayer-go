package sentrytransport_test

import (
	"context"

	"github.com/getsentry/sentry-go"

	sentrytransport "go.loglayer.dev/transports/sentry/v2"
	"go.loglayer.dev/v2"
)

// New forwards entries to a caller-supplied sentry.Logger. The Logger
// is required; obtain one from sentry.NewLogger after sentry.Init.
func ExampleNew() {
	// In production code, call sentry.Init first so the Logger ships
	// events to your project.
	t := sentrytransport.New(sentrytransport.Config{
		Logger: sentry.NewLogger(context.Background()),
	})
	_ = loglayer.New(loglayer.Config{
		Transport:        t,
		DisableFatalExit: true,
	})
}
