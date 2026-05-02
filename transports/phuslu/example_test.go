package phuslu_test

import (
	"io"

	"go.loglayer.dev/transports/phuslu/v2"
	"go.loglayer.dev/v2"
)

// New wraps a *phuslu/log.Logger. When Logger is nil a default logger
// is built that writes to Writer (stderr by default).
//
// Note: phuslu calls os.Exit on FatalLevel from any dispatch path; this
// wrapper cannot suppress that, even with Config.DisableFatalExit.
func ExampleNew() {
	t := phuslu.New(phuslu.Config{Writer: io.Discard})
	log := loglayer.New(loglayer.Config{
		Transport:        t,
		DisableFatalExit: true,
	})
	log.Info("served")
}
