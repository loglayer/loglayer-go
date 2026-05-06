// Package axiom provides a LogLayer transport backed by a
// caller-supplied [axiom.Client] from github.com/axiomhq/axiom-go.
//
// The user owns the client lifecycle (authentication, base URL); this
// transport assembles an NDJSON entry from the params and dispatches it
// via Client.Ingest().
//
// # Payload shape
//
// Each log entry is sent as a JSON object with:
//
//   - the joined message text under Config.MessageField (default "msg");
//   - all persistent fields and the serialized error from params.Data
//     merged at root;
//   - map metadata flattened at root, or any other metadata nested
//     under params.Schema.MetadataFieldName (or "metadata" by default).
//
// See https://go.loglayer.dev for usage guides and the full API reference.
package axiom

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/axiomhq/axiom-go/axiom"

	"go.loglayer.dev/v2"
	"go.loglayer.dev/v2/transport"
)

// Config holds configuration options for the Axiom transport.
type Config struct {
	transport.BaseConfig

	// Client is the underlying *axiom.Client from github.com/axiomhq/axiom-go.
	// Required. Construct via axiom.NewClient() or with options.
	Client *axiom.Client

	// DatasetName is the ID or name of the Axiom dataset to ingest logs into.
	// Required.
	DatasetName string

	// MessageField is the key under which the joined message text is
	// placed in the JSON object. Defaults to "msg".
	MessageField string

	// OnError is called when Client.Ingest returns an error.
	// The default writes a one-line message to os.Stderr.
	OnError func(error)
}

// Transport ships log entries to an Axiom dataset via Client.Ingest().
type Transport struct {
	transport.BaseTransport
	cfg Config
}

// New constructs an Axiom Transport. Panics if cfg.Client is nil or
// cfg.DatasetName is empty. Use Build for an error-returning variant.
func New(cfg Config) *Transport {
	t, err := Build(cfg)
	if err != nil {
		panic(err)
	}
	return t
}

// Build constructs a Transport like New but returns errors instead of panicking.
func Build(cfg Config) (*Transport, error) {
	if cfg.Client == nil {
		return nil, ErrClientRequired
	}
	if cfg.DatasetName == "" {
		return nil, ErrDatasetNameRequired
	}
	if cfg.MessageField == "" {
		cfg.MessageField = "msg"
	}
	return &Transport{
		BaseTransport: transport.NewBaseTransport(cfg.BaseConfig),
		cfg:           cfg,
	}, nil
}

// GetLoggerInstance returns the underlying *axiom.Client.
func (t *Transport) GetLoggerInstance() any { return t.cfg.Client }

// SendToLogger implements loglayer.Transport.
func (t *Transport) SendToLogger(params loglayer.TransportParams) {
	if !t.ShouldProcess(params.LogLevel) {
		return
	}

	params.Messages = transport.JoinPrefixAndMessages(params.Prefix, params.Messages)
	entry := t.buildEntry(params)
	buf, err := json.Marshal(entry)
	if err != nil {
		t.reportError(fmt.Errorf("marshal entry: %w", err))
		return
	}

	ctx := params.Ctx
	if ctx == nil {
		ctx = context.Background()
	}

	reader := io.NopCloser(bytes.NewReader(buf))
	if _, err := t.cfg.Client.Ingest(ctx, t.cfg.DatasetName, reader, axiom.NDJSON, axiom.Identity); err != nil {
		t.reportError(err)
	}
}

// buildEntry assembles a JSON object from TransportParams + Config.
func (t *Transport) buildEntry(params loglayer.TransportParams) map[string]any {
	payload := make(map[string]any, transport.FieldEstimate(params)+1)
	payload[t.cfg.MessageField] = transport.JoinMessages(params.Messages)

	// Merge params.Data (persistent fields + error) into payload
	for k, v := range params.Data {
		payload[k] = v
	}

	transport.MergeIntoMap(payload, nil, params.Metadata, params.Schema.MetadataFieldName)
	return payload
}

func (t *Transport) reportError(err error) {
	if t.cfg.OnError != nil {
		t.cfg.OnError(err)
		return
	}
	fmt.Fprintf(os.Stderr, "loglayer/transports/axiom: %v\n", err)
}

// Close implements io.Closer for use with AddTransport / RemoveTransport
// lifecycle management. The Axiom client handles its own buffering and
// flushes on each Ingest call, so this is a no-op.
func (t *Transport) Close() error { return nil }
