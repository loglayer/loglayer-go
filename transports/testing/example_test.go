package testing_test

import (
	"fmt"

	lltesting "go.loglayer.dev/transports/testing/v2"
	"go.loglayer.dev/v2"
)

// New builds an in-memory capture transport. Tests drive the logger,
// then pop captured lines from Transport.Library to assert on level,
// messages, and data.
func ExampleNew() {
	t := lltesting.New(lltesting.Config{})
	log := loglayer.New(loglayer.Config{
		Transport:        t,
		DisableFatalExit: true,
	})

	log.Info("served")

	line := t.Library.PopLine()
	fmt.Println(line.Level, line.Messages[0])
	// Output: info served
}
