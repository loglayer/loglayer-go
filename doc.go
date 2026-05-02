// Package loglayer is a transport-agnostic structured logger with a
// fluent builder API. The core defines the LogLayer type, the Transport
// and Plugin interfaces, and the dispatch pipeline. Concrete transports
// (zap, zerolog, slog, charmlog, OTel, etc.) ship as separately-versioned
// sub-modules under go.loglayer.dev/transports/<name>.
//
// Full docs: https://go.loglayer.dev
//
// # Quickstart
//
//	import (
//	    "go.loglayer.dev/v2"
//	    "go.loglayer.dev/transports/structured/v2"
//	)
//
//	log := loglayer.New(loglayer.Config{
//	    Transport: structured.New(structured.Config{}),
//	})
//	log.WithFields(loglayer.Fields{"requestId": "abc"}).
//	    WithMetadata(loglayer.Metadata{"durationMs": 42}).
//	    Info("served")
//
// # Three data shapes
//
// LogLayer separates persistent from per-call data on purpose. Pick the
// one whose lifetime matches the data:
//
//   - Fields (map[string]any): persistent on the logger. Set once via
//     WithFields and it appears on every subsequent log entry. Use for
//     request IDs, user IDs, and anything request-scoped.
//   - Metadata (any): single log call only. Use for per-event payloads
//     such as durations, counters, or structs. Maps merge at the entry
//     root; other values nest under Config.MetadataFieldName.
//   - Context (context.Context): single log call only. Transports that
//     understand context (OTel, slog) read trace IDs and deadlines from
//     it; others ignore it.
//
// # Choosing a constructor
//
//   - New panics on misconfiguration. Use at program start when failure
//     means the binary cannot run.
//   - Build returns (*LogLayer, error). Use when the config is loaded at
//     runtime (env vars, config file) and the caller wants to handle
//     ErrNoTransport, ErrTransportAndTransports, or
//     ErrUngroupedTransportsWithoutMode via errors.Is.
//   - NewMock returns a silent logger that accepts every call and emits
//     nothing. Use in tests that inject a *LogLayer.
//
// # Concurrency
//
// Every method on *LogLayer is safe to call from any goroutine,
// including concurrently with emission. Fluent chain methods
// (WithFields, WithoutFields, Child, WithPrefix, WithGroup, WithContext)
// return a new logger and never mutate the receiver. Level, transport,
// plugin, and group mutators are atomic and intended for live runtime
// toggling. See the per-method GoDoc for the exact class.
//
// # Authoring transports and plugins
//
// Transport and plugin implementations live in their own sub-modules.
// The transport/ package exports BaseTransport, BaseConfig, and shared
// helpers (JoinMessages, MetadataAsMap, MergeFieldsAndMetadata) for
// authors. See https://go.loglayer.dev for the authoring guides.
package loglayer
