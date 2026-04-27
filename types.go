package loglayer

import "context"

// Fields is persistent key/value data included with every log entry from a
// logger instance. Set via WithFields; surfaced to transports via
// TransportParams.Fields.
type Fields = map[string]any

// Data is the assembled object sent to transports containing the persistent
// fields and the serialized error.
type Data = map[string]any

// Metadata is a convenience alias for the most common metadata shape: a string-keyed
// map of arbitrary values. WithMetadata accepts any value, but when the data is an
// ad-hoc bag of fields this alias keeps call sites short.
type Metadata = map[string]any

// ErrorSerializer converts an error into a structured map for the log output.
// If not set, the default serializer uses {"message": err.Error()}.
type ErrorSerializer func(err error) map[string]any

// Config is the initialization configuration for a LogLayer instance.
type Config struct {
	// Transport is a single transport. Set either Transport or Transports, not both.
	Transport Transport
	// Transports is a slice of transports. Set either Transport or Transports, not both.
	Transports []Transport

	// Plugins are added to the logger at construction time, in slice order.
	// Equivalent to calling AddPlugin for each entry after construction;
	// either form is fine.
	Plugins []Plugin

	// Prefix is prepended to the first string message of every log call.
	Prefix string

	// Enabled controls whether logging is active. Defaults to true.
	Enabled *bool

	// ErrorSerializer customizes how errors are serialized into the log data.
	ErrorSerializer ErrorSerializer

	// ErrorFieldName is the key used for the serialized error in log data. Defaults to "err".
	ErrorFieldName string

	// CopyMsgOnOnlyError copies err.Error() as the log message when calling ErrorOnly.
	CopyMsgOnOnlyError bool

	// FieldsKey nests all persistent fields under this key. If empty, fields are merged at root.
	FieldsKey string

	// MuteFields disables inclusion of persistent fields in log output.
	MuteFields bool

	// MuteMetadata disables inclusion of metadata in log output.
	MuteMetadata bool

	// DisableFatalExit prevents the library from calling os.Exit(1) after a
	// Fatal-level log is dispatched. Defaults to false (Fatal exits, matching
	// the Go convention used by log.Fatal, zerolog, zap, logrus, and others).
	//
	// Set to true in tests, library code that shouldn't terminate the host
	// process, or any context where os.Exit would be inappropriate.
	//
	// Note: a few logger-wrapper transports (notably phuslu) may still trigger
	// their underlying library's exit before this option takes effect. See
	// each wrapper's docs for details.
	DisableFatalExit bool
}

// ErrorOnlyOpts are optional settings for the ErrorOnly method.
type ErrorOnlyOpts struct {
	// LogLevel overrides the default error level. Defaults to LogLevelError.
	LogLevel LogLevel

	// CopyMsg overrides Config.CopyMsgOnOnlyError for this call.
	// nil means "use the config default".
	CopyMsg *bool
}

// RawLogEntry is a fully specified log entry used with the Raw method.
type RawLogEntry struct {
	LogLevel LogLevel
	Messages []any
	// Metadata is per-entry data. Accepts any value: structs, maps, or any other type.
	// Serialization is handled by the transport.
	Metadata any
	Err      error
	// Fields overrides the logger's persistent fields for this entry. If nil
	// the logger's current fields are used.
	Fields Fields
	// Ctx is an optional Go context.Context for the entry; surfaced to transports
	// via TransportParams.Ctx. Use it to carry trace IDs, deadlines, or anything
	// else a transport may extract.
	Ctx context.Context
}
