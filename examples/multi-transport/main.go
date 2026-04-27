// Multi-transport fan-out: pretty in dev, structured to a file always.
// One log call writes to both. Also shows per-transport level filtering: the
// file transport only records warn+ to keep production noise down.
//
// Run:
//
//	go run ./examples/multi-transport
//
// Inspect the file output:
//
//	cat /tmp/loglayer-example.log
package main

import (
	"errors"
	"os"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
	"go.loglayer.dev/transports/pretty"
	"go.loglayer.dev/transports/structured"
)

func main() {
	logFile, err := os.Create("/tmp/loglayer-example.log")
	if err != nil {
		panic(err)
	}
	defer logFile.Close()

	log := loglayer.New(loglayer.Config{
		Transports: []loglayer.Transport{
			pretty.New(pretty.Config{
				BaseConfig: transport.BaseConfig{ID: "console"},
			}),
			structured.New(structured.Config{
				BaseConfig: transport.BaseConfig{
					ID:    "file",
					Level: loglayer.LogLevelWarn, // only warn+ goes to the file
				},
				Writer: logFile,
			}),
		},
	})

	log = log.WithFields(loglayer.Fields{"service": "demo"})

	log.Info("info goes to console only (below the file's threshold)")
	log.Warn("warn goes to BOTH transports")
	log.WithError(errors.New("simulated failure")).Error("error goes to BOTH transports")
}
