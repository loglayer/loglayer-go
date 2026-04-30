package console_test

import (
	"os"

	"go.loglayer.dev"
	"go.loglayer.dev/transports/console"
)

// New builds a logfmt-style console transport. DateFn returns a fixed
// string here so the example output is deterministic.
func ExampleNew() {
	t := console.New(console.Config{
		Writer:     os.Stdout,
		DateField:  "time",
		LevelField: "level",
		DateFn:     func() string { return "2026-04-26T12:00:00Z" },
	})
	log := loglayer.New(loglayer.Config{
		Transport:        t,
		DisableFatalExit: true,
	})
	log.WithMetadata(loglayer.Metadata{"durationMs": 42}).Info("served")
	// Output: served durationMs=42 level=info time=2026-04-26T12:00:00Z
}
