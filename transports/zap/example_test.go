package zap_test

import (
	"io"

	"go.loglayer.dev/transports/zap/v2"
	"go.loglayer.dev/v2"
)

// New wraps a *zap.Logger. When Logger is nil a default logger is
// built with a JSON encoder over Writer (stderr by default).
func ExampleNew() {
	t := zap.New(zap.Config{Writer: io.Discard})
	log := loglayer.New(loglayer.Config{
		Transport:        t,
		DisableFatalExit: true,
	})
	log.Info("served")
}
