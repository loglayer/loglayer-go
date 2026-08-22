package newrelic_test

import (
	"go.loglayer.dev/transports/newrelic/v2"
	"go.loglayer.dev/v3"
)

// New ships log entries to the New Relic Logs HTTP intake. LicenseKey is
// required; Site selects the regional intake (defaults to SiteUS). The
// transport spawns a worker goroutine; call Close on shutdown to flush
// pending entries.
func ExampleNew() {
	t := newrelic.New(newrelic.Config{
		LicenseKey: "your-new-relic-license-key",
		Site:       newrelic.SiteUS,
	})
	defer t.Close()

	log := loglayer.New(loglayer.Config{
		Transport:        t,
		DisableFatalExit: true,
	})
	log.Info("served")
}
