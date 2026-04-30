package logrus_test

import (
	"io"

	"go.loglayer.dev"
	"go.loglayer.dev/transports/logrus"
)

// New wraps a *logrus.Logger. When Logger is nil a default logger is
// built that writes to Writer (stderr by default).
func ExampleNew() {
	t := logrus.New(logrus.Config{Writer: io.Discard})
	log := loglayer.New(loglayer.Config{
		Transport:        t,
		DisableFatalExit: true,
	})
	log.Info("served")
}
