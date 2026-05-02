package charmlog_test

import (
	"io"

	"go.loglayer.dev/transports/charmlog/v2"
	"go.loglayer.dev/v2"
)

// New wraps a *charmbracelet/log.Logger. When Logger is nil a default
// logger is built that writes to Writer (stderr by default).
func ExampleNew() {
	t := charmlog.New(charmlog.Config{Writer: io.Discard})
	log := loglayer.New(loglayer.Config{
		Transport:        t,
		DisableFatalExit: true,
	})
	log.Info("served")
}
