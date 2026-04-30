package structured_test

import (
	"go.loglayer.dev"
	"go.loglayer.dev/transports/structured"
)

// New builds a structured-JSON transport. DateFn returns a fixed
// string here so the example output is deterministic.
func ExampleNew() {
	t := structured.New(structured.Config{
		DateFn: func() string { return "2026-04-26T12:00:00Z" },
	})
	log := loglayer.New(loglayer.Config{
		Transport:        t,
		DisableFatalExit: true,
	})
	log.WithMetadata(loglayer.Metadata{"durationMs": 42}).Info("served")
	// Output: {"level":"info","time":"2026-04-26T12:00:00Z","msg":"served","durationMs":42}
}
