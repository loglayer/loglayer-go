package central_test

import (
	"go.loglayer.dev/transports/central/v2"
	"go.loglayer.dev/v2"
)

// New ships log entries to the LogLayer Central server's HTTP intake.
// Service is required; BaseURL defaults to http://localhost:9800. The
// transport spawns a worker goroutine; call Close on shutdown to flush
// pending entries.
func ExampleNew() {
	t := central.New(central.Config{
		Service:    "checkout-api",
		InstanceID: "checkout-api-1",
	})
	defer t.Close()

	log := loglayer.New(loglayer.Config{
		Transport:        t,
		DisableFatalExit: true,
	})
	log.Info("served")
}
