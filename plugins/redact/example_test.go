package redact_test

import (
	"fmt"

	"go.loglayer.dev"
	"go.loglayer.dev/plugins/redact"
	lltesting "go.loglayer.dev/transports/testing"
)

// New returns a plugin that replaces values for keys listed in
// Config.Keys with Censor (default "[REDACTED]") wherever they appear,
// at any depth. Caller-owned input is never mutated.
func ExampleNew() {
	tr := lltesting.New(lltesting.Config{})
	log := loglayer.New(loglayer.Config{
		Transport:        tr,
		DisableFatalExit: true,
		Plugins: []loglayer.Plugin{
			redact.New(redact.Config{Keys: []string{"password"}}),
		},
	})

	log.WithMetadata(loglayer.Metadata{
		"user":     "alice",
		"password": "hunter2",
	}).Info("login")

	line := tr.Library.PopLine()
	md := line.Metadata.(loglayer.Metadata)
	fmt.Println(md["user"], md["password"])
	// Output: alice [REDACTED]
}
