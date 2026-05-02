package slog_test

import (
	"io"

	llslog "go.loglayer.dev/transports/slog/v2"
	"go.loglayer.dev/v2"
)

// New wraps a *slog.Logger. When Logger is nil a default logger is
// built using slog.NewJSONHandler over Writer (stderr by default).
func ExampleNew() {
	t := llslog.New(llslog.Config{Writer: io.Discard})
	log := loglayer.New(loglayer.Config{
		Transport:        t,
		DisableFatalExit: true,
	})
	log.Info("served")
}
