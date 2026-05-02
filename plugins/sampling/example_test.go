package sampling_test

import (
	"fmt"

	"go.loglayer.dev/plugins/sampling/v2"
	lltesting "go.loglayer.dev/transports/testing/v2"
	"go.loglayer.dev/v2"
)

// FixedRate keeps the given fraction of emissions. rate >= 1 keeps
// every entry (the gate is a no-op); rate <= 0 drops every entry; in
// between is a per-emission Bernoulli draw.
func ExampleFixedRate() {
	tr := lltesting.New(lltesting.Config{})
	log := loglayer.New(loglayer.Config{
		Transport:        tr,
		DisableFatalExit: true,
		Plugins:          []loglayer.Plugin{sampling.FixedRate(1.0)},
	})

	log.Info("served")
	fmt.Println(tr.Library.Len())
	// Output: 1
}
