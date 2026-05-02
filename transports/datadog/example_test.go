package datadog_test

import (
	"go.loglayer.dev/transports/datadog/v2"
	"go.loglayer.dev/v2"
)

// New ships log entries to the Datadog Logs HTTP intake. APIKey is
// required; Site selects the regional intake (defaults to SiteUS1). The
// transport spawns a worker goroutine; call Close on shutdown to flush
// pending entries.
func ExampleNew() {
	t := datadog.New(datadog.Config{
		APIKey:  "your-datadog-api-key",
		Site:    datadog.SiteUS1,
		Service: "checkout-api",
		Source:  "go",
	})
	defer t.Close()

	log := loglayer.New(loglayer.Config{
		Transport:        t,
		DisableFatalExit: true,
	})
	log.Info("served")
}
