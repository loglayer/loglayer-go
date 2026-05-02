// Implementing a custom Transport (renderer / "flatten" policy).
//
// This example demonstrates the policy used by the structured / pretty /
// console transports: every shape of metadata (map, struct, scalar) is
// flattened to root-level fields via transport.MergeFieldsAndMetadata.
//
// Reach for this pattern when your backend writes flat key=value lines or
// a flat JSON object. For an attribute-style backend (zerolog, zap, OTel)
// see examples/custom-transport-attribute. For an encoder over a protocol
// (HTTP, Datadog) see transports/datadog/datadog.go.
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

	"go.loglayer.dev/v2"
	"go.loglayer.dev/v2/transport"
)

// pipeTransport renders entries as `LEVEL | msg | k=v k=v ...`.
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

	// MergeFieldsAndMetadata returns a single flat map: persistent fields,
	// the serialized error, and metadata all merged at the root. Non-map
	// metadata (struct, slice, scalar) is JSON-roundtripped into a map.
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

	type usage struct {
		BytesIn  int `json:"bytes_in"`
		BytesOut int `json:"bytes_out"`
	}
	log.WithMetadata(usage{BytesIn: 1024, BytesOut: 4096}).Info("rpc done")
}
