package gcplogging_test

import (
	"cloud.google.com/go/logging"

	"go.loglayer.dev/transports/gcplogging/v2"
	"go.loglayer.dev/v2"
)

// New forwards entries to a caller-supplied *logging.Logger from
// cloud.google.com/go/logging, typically constructed via
// logging.NewClient(ctx, projectID).Logger(logID).
func ExampleNew() {
	// gcpLogger is nil in this compile-only example; real code passes a
	// constructed *logging.Logger and uses gcplogging.New, which panics
	// on nil instead of returning the error path.
	var gcpLogger *logging.Logger
	t, err := gcplogging.Build(gcplogging.Config{
		Logger: gcpLogger,
		RootEntry: logging.Entry{
			Labels: map[string]string{"env": "prod"},
		},
	})
	if err != nil {
		return
	}
	_ = loglayer.New(loglayer.Config{
		Transport:        t,
		DisableFatalExit: true,
	})
}

// EntryFn lifts values from per-call metadata onto typed Entry fields
// (Trace, SpanID, Labels, HTTPRequest, ...) instead of leaving them
// flattened into the JSON payload.
func ExampleConfig_EntryFn() {
	var gcpLogger *logging.Logger
	t, err := gcplogging.Build(gcplogging.Config{
		Logger: gcpLogger,
		EntryFn: func(p loglayer.TransportParams, e *logging.Entry) {
			md, ok := p.Metadata.(loglayer.Metadata)
			if !ok {
				return
			}
			if trace, ok := md["trace"].(string); ok {
				e.Trace = trace
			}
			if spanID, ok := md["spanId"].(string); ok {
				e.SpanID = spanID
			}
		},
	})
	if err != nil {
		return
	}
	_ = loglayer.New(loglayer.Config{
		Transport:        t,
		DisableFatalExit: true,
	})
}
