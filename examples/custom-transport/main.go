// Implementing a custom Transport. The four methods are: ID, IsEnabled,
// SendToLogger, GetLoggerInstance. Embedding transport.BaseTransport gives
// you ID/IsEnabled/ShouldProcess for free.
//
// Run:
//
//	go run ./examples/custom-transport
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
)

// pipeTransport renders entries as `LEVEL | msg | k=v k=v ...` and writes
// to its configured writer. Built solely to demonstrate the Transport
// interface; for serious use prefer the structured or pretty transport.
type pipeTransport struct {
	transport.BaseTransport
	w *os.File
}

type pipeConfig struct {
	transport.BaseConfig
	Writer *os.File
}

func newPipe(cfg pipeConfig) *pipeTransport {
	return &pipeTransport{
		BaseTransport: transport.NewBaseTransport(cfg.BaseConfig),
		w:             cfg.Writer,
	}
}

func (p *pipeTransport) GetLoggerInstance() any { return nil }

func (p *pipeTransport) SendToLogger(params loglayer.TransportParams) {
	if !p.ShouldProcess(params.LogLevel) {
		return
	}

	pairs := transport.MergeFieldsAndMetadata(params)
	parts := make([]string, 0, len(pairs))
	for k, v := range pairs {
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}

	fmt.Fprintf(p.w,
		"%s | %s | %s | %s\n",
		strings.ToUpper(params.LogLevel.String()),
		time.Now().Format("15:04:05"),
		transport.JoinMessages(params.Messages),
		strings.Join(parts, " "),
	)
}

func main() {
	tr := newPipe(pipeConfig{
		BaseConfig: transport.BaseConfig{ID: "pipe"},
		Writer:     os.Stdout,
	})

	log := loglayer.New(loglayer.Config{Transport: tr})
	log = log.WithFields(loglayer.Fields{"service": "demo"})

	log.Info("hello world")
	log.WithMetadata(loglayer.Metadata{"action": "ship"}).Warn("processing")
}
