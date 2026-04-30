package fmtlog_test

import (
	"fmt"

	"go.loglayer.dev"
	"go.loglayer.dev/plugins/fmtlog"
	lltesting "go.loglayer.dev/transports/testing"
)

// New returns a plugin that rewrites multi-argument log messages via
// fmt.Sprintf when the first message is a format string. Single-message
// calls are untouched.
func ExampleNew() {
	tr := lltesting.New(lltesting.Config{})
	log := loglayer.New(loglayer.Config{
		Transport:        tr,
		DisableFatalExit: true,
		Plugins:          []loglayer.Plugin{fmtlog.New()},
	})

	log.Info("user %d signed in", 42)

	line := tr.Library.PopLine()
	fmt.Println(line.Messages[0])
	// Output: user 42 signed in
}
