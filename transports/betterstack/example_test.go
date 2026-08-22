package betterstack

import (
	"go.loglayer.dev/v3"
)

// ExampleNew shows how to create a Better Stack transport and ship logs.
func ExampleNew() {
	t := New(Config{
		SourceToken: "your-source-token",
		URL:         "https://in.logs.betterstack.com",
	})
	defer t.Close()

	log := loglayer.New(loglayer.Config{
		Transport:        t,
		DisableFatalExit: true,
	})
	log.Info("served")
}
