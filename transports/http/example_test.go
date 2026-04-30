package httptransport_test

import (
	"go.loglayer.dev"
	httptransport "go.loglayer.dev/transports/http"
)

// New POSTs JSON batches to URL. The worker goroutine starts at
// construction time; call Close on shutdown to flush pending entries
// and stop the worker.
func ExampleNew() {
	t := httptransport.New(httptransport.Config{
		URL: "https://logs.example.com/ingest",
	})
	defer t.Close()

	log := loglayer.New(loglayer.Config{
		Transport:        t,
		DisableFatalExit: true,
	})
	log.Info("served")
}
