package pretty_test

import (
	"os"
	"time"

	"go.loglayer.dev/transports/pretty/v2"
	"go.loglayer.dev/v2"
)

// New builds the colorized terminal transport. NoColor and a fixed
// TimestampFn make the example output deterministic.
func ExampleNew() {
	t := pretty.New(pretty.Config{
		Writer:      os.Stdout,
		NoColor:     true,
		TimestampFn: func(_ time.Time) string { return "12:00:00.000" },
	})
	log := loglayer.New(loglayer.Config{
		Transport:        t,
		DisableFatalExit: true,
	})
	log.Info("served")
	// Output: 12:00:00.000 ▶ INFO served
}
