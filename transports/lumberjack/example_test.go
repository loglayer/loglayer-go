package lumberjack_test

import (
	"go.loglayer.dev"
	"go.loglayer.dev/transports/lumberjack"
)

// New writes one JSON object per entry to a rotating file. Filename is
// required; MaxSize/MaxBackups/MaxAge/Compress tune lumberjack's
// rotation policy. Call Close on shutdown to release the file handle.
func ExampleNew() {
	t := lumberjack.New(lumberjack.Config{
		Filename:   "/var/log/app.log",
		MaxSize:    100, // megabytes per file before rotating
		MaxBackups: 5,
		Compress:   true,
	})
	defer t.Close()

	log := loglayer.New(loglayer.Config{
		Transport:        t,
		DisableFatalExit: true,
	})
	log.Info("served")
}
