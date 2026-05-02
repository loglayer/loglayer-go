// Package gcplogging provides a LogLayer transport backed by a
// caller-supplied [logging.Logger] from cloud.google.com/go/logging.
//
// The user owns logging.Client and logging.Logger lifecycle (project
// resolution, credentials, log name selection); this transport assembles
// a logging.Entry from the params and dispatches it via Logger.Log
// (async) or Logger.LogSync (blocking).
//
// # Severity mapping
//
// LogLayer levels map to GCP [logging.Severity] as follows:
//
//   - LogLevelTrace, LogLevelDebug -> Debug
//   - LogLevelInfo                 -> Info
//   - LogLevelWarn                 -> Warning
//   - LogLevelError                -> Error
//   - LogLevelFatal                -> Critical
//   - LogLevelPanic                -> Alert
//
// # Payload shape
//
// Entry.Payload is a map[string]any containing:
//
//   - the joined message text under Config.MessageField (default
//     "message");
//   - all persistent fields and the serialized error from params.Data
//     merged at root;
//   - map metadata flattened at root, or any other metadata nested
//     under params.Schema.MetadataFieldName (or "metadata" by default).
//
// Other Entry fields (Resource, Labels, HTTPRequest, Trace,
// SourceLocation, ...) come from Config.RootEntry, with per-entry
// overrides via Config.EntryFn.
//
// See https://go.loglayer.dev for usage guides and the full API reference.
package gcplogging

import (
	"context"
	"fmt"
	"os"
	"time"

	"cloud.google.com/go/logging"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
)

// Config holds configuration options for the GCP Cloud Logging transport.
type Config struct {
	transport.BaseConfig

	// Logger is the underlying *logging.Logger from
	// cloud.google.com/go/logging. Required. Construct via
	// logging.NewClient(ctx, projectID).Logger(logID).
	Logger *logging.Logger

	// RootEntry is the LogEntry skeleton merged into every entry.
	// Severity, Timestamp, and Payload are populated by the transport
	// and override any values set here. Common fields to set: Resource,
	// Labels, HTTPRequest, Operation, Trace, SourceLocation.
	RootEntry logging.Entry

	// EntryFn is an optional per-entry hook that mutates the resolved
	// Entry just before dispatch. Use it to lift values from
	// params.Metadata onto typed Entry fields (Labels, HTTPRequest,
	// Trace, SpanID, SourceLocation, ...). Severity, Timestamp, and
	// Payload are already set when EntryFn runs; overwriting them is
	// allowed but rarely useful.
	EntryFn func(params loglayer.TransportParams, entry *logging.Entry)

	// MessageField is the key under which the joined message text is
	// placed inside Payload. Defaults to "message".
	MessageField string

	// Sync routes entries through Logger.LogSync (blocking) instead of
	// Logger.Log (async, batched). Use Sync for short-lived processes
	// (Cloud Functions, CI tasks) where queued entries might not flush
	// before the process exits.
	Sync bool

	// OnError is called when LogSync returns an error or Close's Flush
	// fails. The default writes a one-line message to os.Stderr.
	// Errors from async Logger.Log calls go through
	// logging.Client.OnError, not this hook.
	OnError func(error)
}

// Transport ships log entries to a *logging.Logger.
type Transport struct {
	transport.BaseTransport
	cfg Config
}

// New constructs a GCP Cloud Logging Transport. Panics if cfg.Logger is
// nil. Use Build for an error-returning variant.
func New(cfg Config) *Transport {
	t, err := Build(cfg)
	if err != nil {
		panic(err)
	}
	return t
}

// Build constructs a Transport like New but returns ErrLoggerRequired
// instead of panicking when cfg.Logger is nil.
func Build(cfg Config) (*Transport, error) {
	if cfg.Logger == nil {
		return nil, ErrLoggerRequired
	}
	if cfg.MessageField == "" {
		cfg.MessageField = "message"
	}
	return &Transport{
		BaseTransport: transport.NewBaseTransport(cfg.BaseConfig),
		cfg:           cfg,
	}, nil
}

// GetLoggerInstance returns the underlying *logging.Logger.
func (t *Transport) GetLoggerInstance() any { return t.cfg.Logger }

// SendToLogger implements loglayer.Transport.
func (t *Transport) SendToLogger(params loglayer.TransportParams) {
	if !t.ShouldProcess(params.LogLevel) {
		return
	}
	// Preserve the v1 "prefix folded into Messages[0]" rendering;
	// the core no longer mutates messages, transports own it now.
	params.Messages = transport.JoinPrefixAndMessages(params.Prefix, params.Messages)
	entry := t.buildEntry(params)
	if t.cfg.Sync {
		ctx := params.Ctx
		if ctx == nil {
			ctx = context.Background()
		}
		if err := t.cfg.Logger.LogSync(ctx, entry); err != nil {
			t.reportError(err)
		}
		return
	}
	t.cfg.Logger.Log(entry)
}

// Close flushes any buffered async entries via Logger.Flush. Implements
// io.Closer so AddTransport / RemoveTransport can drain the transport
// when it's swapped out.
func (t *Transport) Close() error {
	if err := t.cfg.Logger.Flush(); err != nil {
		t.reportError(err)
		return err
	}
	return nil
}

// buildEntry assembles a logging.Entry from TransportParams + Config.
// Split out from SendToLogger so the assembly logic is unit-testable
// without a live *logging.Logger.
func (t *Transport) buildEntry(params loglayer.TransportParams) logging.Entry {
	payload := make(map[string]any, transport.FieldEstimate(params)+1)
	payload[t.cfg.MessageField] = transport.JoinMessages(params.Messages)
	transport.MergeIntoMap(payload, params.Data, params.Metadata, params.Schema.MetadataFieldName)

	entry := t.cfg.RootEntry
	entry.Severity = severityFor(params.LogLevel)
	entry.Timestamp = time.Now()
	entry.Payload = payload

	if t.cfg.EntryFn != nil {
		t.cfg.EntryFn(params, &entry)
	}
	return entry
}

func (t *Transport) reportError(err error) {
	if t.cfg.OnError != nil {
		t.cfg.OnError(err)
		return
	}
	fmt.Fprintf(os.Stderr, "loglayer/transports/gcplogging: %v\n", err)
}

// severityFor maps a loglayer LogLevel to the corresponding GCP
// logging.Severity. Trace collapses into Debug since GCP has no
// dedicated trace severity. Panic maps to Alert (one above Critical)
// because loglayer's core triggers the runtime panic itself; the
// transport only forwards severity.
func severityFor(level loglayer.LogLevel) logging.Severity {
	switch level {
	case loglayer.LogLevelTrace, loglayer.LogLevelDebug:
		return logging.Debug
	case loglayer.LogLevelInfo:
		return logging.Info
	case loglayer.LogLevelWarn:
		return logging.Warning
	case loglayer.LogLevelError:
		return logging.Error
	case loglayer.LogLevelFatal:
		return logging.Critical
	case loglayer.LogLevelPanic:
		return logging.Alert
	default:
		return logging.Default
	}
}
