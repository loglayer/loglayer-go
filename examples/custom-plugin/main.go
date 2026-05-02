// Demonstrates writing a LogLayer plugin from scratch. The plugin in
// this example tags every log entry with the host's name and process
// PID under configurable keys, and counts emissions in an atomic
// counter that you could surface to a metrics endpoint.
//
// It exercises three different plugin hooks so you see each one in
// context:
//
//   - OnBeforeDataOut: per-emission, runs after fields and the
//     serialized error are assembled. Use it to attach data like trace
//     IDs, host info, etc.
//   - OnMetadataCalled: builder-time, runs from WithMetadata. Use it
//     to redact or reshape metadata before the entry is emitted.
//   - ShouldSend: per-(entry, transport) gate. Use it to drop entries
//     from specific transports based on content.
//
// Run from the repo root:
//
//	go run ./examples/custom-plugin
package main

import (
	"fmt"
	"os"
	"strings"
	"sync/atomic"

	"go.loglayer.dev/transports/pretty/v2"
	"go.loglayer.dev/v2"
	"go.loglayer.dev/v2/transport"
)

// hostInfoPlugin tags every entry with the host name and PID, and
// counts emissions in an exported atomic counter. The plugin is
// stateless apart from the counter; the per-entry hook reads no
// per-instance fields, so it's safe to add to as many loggers as you
// want without coordination.
func hostInfoPlugin(hostKey, pidKey string, counter *atomic.Uint64) loglayer.Plugin {
	host, err := os.Hostname()
	if err != nil {
		host = "unknown"
	}
	pid := os.Getpid()
	return &hostInfo{hostKey: hostKey, pidKey: pidKey, host: host, pid: pid, counter: counter}
}

// hostInfo implements DataHook + ErrorReporter.
type hostInfo struct {
	hostKey, pidKey string
	host            string
	pid             int
	counter         *atomic.Uint64
}

func (h *hostInfo) ID() string { return "host-info" }

func (h *hostInfo) OnBeforeDataOut(loglayer.BeforeDataOutParams) loglayer.Data {
	h.counter.Add(1)
	return loglayer.Data{
		h.hostKey: h.host,
		h.pidKey:  h.pid,
	}
}

func (h *hostInfo) OnError(err error) {
	fmt.Fprintf(os.Stderr, "host-info plugin error: %v\n", err)
}

// scrubPasswordPlugin strips any "password" key from metadata maps
// before they reach a transport. This runs at WithMetadata time, so by
// the time the transport sees the entry the field is already gone.
//
// In production, prefer the built-in plugins/redact plugin: it handles
// nested maps, structs, regex patterns, and preserves runtime types.
// This implementation exists only to demonstrate the OnMetadataCalled
// hook on a flat top-level map.
func scrubPasswordPlugin() loglayer.Plugin {
	return loglayer.NewMetadataHook("scrub-password", func(metadata any) any {
		m, ok := transport.MetadataAsRootMap(metadata)
		if !ok {
			return metadata
		}
		cleaned := make(loglayer.Metadata, len(m))
		for k, v := range m {
			if strings.EqualFold(k, "password") {
				cleaned[k] = "[REDACTED]"
				continue
			}
			cleaned[k] = v
		}
		return cleaned
	})
}

// dropDebugFromTransport returns a plugin that vetoes Debug-level
// entries on the named transport ID. The same entry still reaches
// other transports. ShouldSend is a per-(entry, transport) gate, not
// a global filter. Useful when you want one transport to be quieter
// than the rest.
func dropDebugFromTransport(transportID string) loglayer.Plugin {
	return loglayer.NewSendGate("drop-debug-on-"+transportID, func(p loglayer.ShouldSendParams) bool {
		if p.TransportID == transportID && p.LogLevel == loglayer.LogLevelDebug {
			return false
		}
		return true
	})
}

func main() {
	var counter atomic.Uint64

	tr := pretty.New(pretty.Config{
		BaseConfig: transport.BaseConfig{ID: "pretty"},
	})

	log := loglayer.New(loglayer.Config{
		Transport: tr,
		Plugins: []loglayer.Plugin{
			hostInfoPlugin("host", "pid", &counter),
			scrubPasswordPlugin(),
			dropDebugFromTransport("pretty"),
		},
	})

	log.WithFields(loglayer.Fields{"requestId": "abc-123"}).
		Info("server started")

	log.WithMetadata(loglayer.Metadata{
		"username": "alice",
		"password": "shouldnotappear",
	}).Info("login")

	// Debug entry: vetoed by dropDebugFromTransport for the pretty transport.
	log.Debug("vetoed; not emitted")

	// Note the count is 3, not 2: OnBeforeDataOut runs *before* ShouldSend
	// gates the dispatch, so the vetoed Debug entry still ran through the
	// host-info plugin's hook. If you need a "what actually shipped" count,
	// move the counter into a transport's SendToLogger instead.
	fmt.Printf("\nEntries that ran through OnBeforeDataOut: %d\n", counter.Load())
}
