package newrelic_test

import (
	"go.loglayer.dev/transports/newrelic/v2"
	"go.loglayer.dev/v2"
)

// New ships log entries to the New Relic Log API. APIKey is required; Zone
// selects the regional endpoint (defaults to ZoneUS). The transport spawns a
// worker goroutine; call Close on shutdown to flush pending entries.
func ExampleNew() {
	t := newrelic.New(newrelic.Config{
		APIKey: "your-new-relic-api-key",
		Zone:   newrelic.ZoneUS,
	})
	defer t.Close()

	log := loglayer.New(loglayer.Config{
		Transport:        t,
		DisableFatalExit: true,
	})
	log.Info("served")
}
