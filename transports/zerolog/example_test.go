package zerolog_test

import (
	"io"

	"go.loglayer.dev"
	"go.loglayer.dev/transports/zerolog"
)

// New wraps a *zerolog.Logger. When Logger is nil a default logger is
// built that writes to Writer (stderr by default).
func ExampleNew() {
	t := zerolog.New(zerolog.Config{Writer: io.Discard})
	log := loglayer.New(loglayer.Config{
		Transport:        t,
		DisableFatalExit: true,
	})
	log.Info("served")
}
