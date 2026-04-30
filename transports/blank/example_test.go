package blank_test

import (
	"fmt"

	"go.loglayer.dev"
	"go.loglayer.dev/transports/blank"
)

// New wraps a callback. ShipToLogger receives every TransportParams
// that passes the level filter; the callback owns dispatch (write to a
// queue, push to a metrics system, forward to another logger).
func ExampleNew() {
	t := blank.New(blank.Config{
		ShipToLogger: func(p loglayer.TransportParams) {
			fmt.Println(p.Messages[0])
		},
	})
	log := loglayer.New(loglayer.Config{
		Transport:        t,
		DisableFatalExit: true,
	})
	log.Info("served")
	// Output: served
}
