// Implementing a custom Transport (attribute-forwarding / wrapper policy).
//
// This example demonstrates the policy used by the zerolog / zap / slog /
// logrus / charmlog / phuslu / OpenTelemetry wrappers: forward each piece
// of structured data to a backend's native attribute API. Map metadata
// flattens at the root; non-map metadata nests under a single
// MetadataFieldName key so the backend's own marshaler renders it.
//
// Use transport.MetadataAsRootMap (no allocation, no JSON roundtrip) to
// branch on whether the metadata is map-shaped before forwarding.
//
// Reach for this pattern when your backend has an attribute API like
// zerolog's Event.Interface, zap's zap.Any, or OTel's KeyValue.
//
// Run:
//
//	go run ./examples/custom-transport-attribute
package main

import (
	"fmt"
	"os"

	"go.loglayer.dev/v2"
	"go.loglayer.dev/v2/transport"
)

// fakeBackend is a stand-in for a real attribute-aware logger. It accepts
// key/value pairs via Add and renders them as `key=value` on emit. In a
// real wrapper transport, this is where the third-party library lives.
type fakeBackend struct {
	w *os.File
}

func (b *fakeBackend) Emit(level, msg string, attrs []attr) {
	fmt.Fprintf(b.w, "[%s] %s", level, msg)
	for _, a := range attrs {
		fmt.Fprintf(b.w, " %s=%v", a.key, a.value)
	}
	fmt.Fprintln(b.w)
}

type attr struct {
	key   string
	value any
}

// attrTransport wraps fakeBackend behind LogLayer.
type attrTransport struct {
	transport.BaseTransport
	cfg attrConfig
}

type attrConfig struct {
	transport.BaseConfig
	Backend *fakeBackend

	// MetadataFieldName is the single attribute key that non-map metadata
	// (structs, slices, scalars) lands under. Defaults to "metadata".
	// Conventionally configurable so callers can rename it per service.
	MetadataFieldName string
}

func newAttr(cfg attrConfig) *attrTransport {
	if cfg.MetadataFieldName == "" {
		cfg.MetadataFieldName = "metadata"
	}
	return &attrTransport{
		BaseTransport: transport.NewBaseTransport(cfg.BaseConfig),
		cfg:           cfg,
	}
}

func (t *attrTransport) GetLoggerInstance() any { return t.cfg.Backend }

func (t *attrTransport) SendToLogger(params loglayer.TransportParams) {
	if !t.ShouldProcess(params.LogLevel) {
		return
	}

	// Pre-size the slice; FieldEstimate counts the eventual root-level
	// fields after metadata flattening or nesting.
	attrs := make([]attr, 0, transport.FieldEstimate(params))

	// Persistent fields and the serialized error are already in params.Data.
	for k, v := range params.Data {
		attrs = append(attrs, attr{k, v})
	}

	// Branch on metadata shape. Map metadata flattens at the root; non-map
	// metadata (struct, slice, scalar) is forwarded to the backend under a
	// single MetadataFieldName key, letting the backend's own marshaler
	// handle the value. This is the same dispatch the wrapper transports
	// use internally.
	if params.Metadata != nil {
		if m, ok := transport.MetadataAsRootMap(params.Metadata); ok {
			for k, v := range m {
				attrs = append(attrs, attr{k, v})
			}
		} else {
			attrs = append(attrs, attr{t.cfg.MetadataFieldName, params.Metadata})
		}
	}

	t.cfg.Backend.Emit(params.LogLevel.String(), transport.JoinMessages(params.Messages), attrs)
}

func main() {
	tr := newAttr(attrConfig{
		BaseConfig: transport.BaseConfig{ID: "attr"},
		Backend:    &fakeBackend{w: os.Stdout},
	})

	log := loglayer.New(loglayer.Config{Transport: tr})
	log = log.WithFields(loglayer.Fields{"service": "demo"})

	log.Info("hello world")
	log.WithMetadata(loglayer.Metadata{"action": "ship"}).Warn("processing")

	type usage struct {
		BytesIn  int `json:"bytes_in"`
		BytesOut int `json:"bytes_out"`
	}
	// Struct metadata lands as a single attr under the configured field
	// name; the backend's own renderer handles the nested fields.
	log.WithMetadata(usage{BytesIn: 1024, BytesOut: 4096}).Info("rpc done")
}
