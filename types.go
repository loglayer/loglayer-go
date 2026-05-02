package loglayer

import (
	"context"
	"time"
)

// Fields is persistent key/value data included with every log entry from a
// logger instance. Set via WithFields; surfaced to transports via
// TransportParams.Fields. Distinct from Metadata (per-call) and Data
// (assembled output) at the type level so the compiler catches misuse.
type Fields map[string]any

// F is a short alias for [Fields] for terser call sites:
// log.WithFields(loglayer.F{"reqId": "abc"}).Info("done").
type F = Fields

// Data is the assembled object sent to transports containing the persistent
// fields and the serialized error.
type Data map[string]any

// Metadata is the most common shape passed to WithMetadata: a string-keyed
// map of arbitrary values. WithMetadata accepts any value (struct, scalar,
// slice, anything), but when the data is an ad-hoc bag this named type
// keeps call sites short.
type Metadata map[string]any

// M is a short alias for [Metadata] for terser call sites:
// log.WithMetadata(loglayer.M{"duration": 150}).Info("served").
type M = Metadata

// ErrorSerializer converts an error into a structured map for the log output.
// If not set, the default serializer uses {"message": err.Error()}.
//
// Returning nil drops the error field entirely (the entry is emitted with
// no err key). Returning an empty map adds an empty err object.
type ErrorSerializer func(err error) map[string]any

// Source identifies the call site that produced a log entry. Surfaced under
// Config.Source.FieldName (default "source") when Config.Source.Enabled is true,
// or when an adapter (e.g. the slog handler) supplies it explicitly via
// RawLogEntry.Source. Field names match the slog convention so structured
// output is interchangeable.
type Source struct {
	Function string `json:"function,omitempty"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
}

// Config is the initialization configuration for a LogLayer instance.
type Config struct {
	// Transport is a convenience for the single-transport case. Mutually
	// exclusive with Transports; setting both panics with
	// ErrTransportAndTransports (or returns it from Build).
	Transport Transport
	// Transports is a slice for the multi-transport case. Mutually exclusive
	// with Transport.
	Transports []Transport

	// Plugins are added to the logger at construction time, in slice order.
	// Equivalent to calling AddPlugin for each entry after construction;
	// either form is fine.
	Plugins []Plugin

	// Prefix is exposed verbatim on TransportParams.Prefix and on
	// every dispatch-time plugin hook param struct. Transports
	// decide how to render it: most call
	// transport.JoinPrefixAndMessages to fold it into the first
	// message string; cli renders it in dim grey separate from
	// the level color. Equivalent to calling WithPrefix on the
	// freshly-constructed logger.
	Prefix string

	// Disabled suppresses all log output when true. Defaults to false
	// (logging on). Equivalent to calling DisableLogging() after construction.
	Disabled bool

	// ErrorSerializer customizes how errors are serialized into the log data.
	ErrorSerializer ErrorSerializer

	// ErrorFieldName is the key used for the serialized error in log data. Defaults to "err".
	ErrorFieldName string

	// CopyMsgOnOnlyError copies err.Error() as the log message when calling ErrorOnly.
	CopyMsgOnOnlyError bool

	// FieldsKey nests all persistent fields under this key. If empty, fields are merged at root.
	FieldsKey string

	// MetadataFieldName nests the per-call metadata value under this key in
	// the assembled output. If empty, transports use their default placement
	// policy (renderer transports flatten map metadata at root; wrapper
	// transports flatten map metadata to attributes and nest non-map metadata
	// under a transport-specific default key, typically "metadata").
	//
	// When non-empty, the entry's metadata (whether a map, struct, scalar,
	// or slice) is nested under this single key uniformly, and transports
	// honor that placement.
	MetadataFieldName string

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

	// TransportCloseTimeout caps how long the framework waits for an
	// io.Closer transport to drain on removal (RemoveTransport,
	// SetTransports, AddTransport-by-replace) or pre-Fatal flush, so a
	// wedged endpoint can't hang the process or mutator goroutine.
	// Defaults to 5 seconds when zero or negative.
	TransportCloseTimeout time.Duration

	// OnTransportPanic is called when a transport's SendToLogger panics.
	// The dispatch loop recovers the panic so a buggy transport can't
	// crash the host application from inside a log call, then continues
	// to the next transport.
	//
	// The argument is a [*RecoveredPanicError] with Kind = PanicKindTransport
	// and Hook = the panicking transport's ID. This matches the shape
	// passed to plugin [ErrorReporter.OnError] callbacks so a single
	// observability handler can absorb panics from either source.
	//
	// Default (nil): no recover wrap; a panicking transport propagates
	// up through the emission call (matching the convention used by
	// zerolog/zap/log/slog). Set this to plumb panics into your own
	// observability (a metrics counter, an error tracker, a separate
	// logger). A panic from inside this callback is itself recovered
	// (and dropped) so a buggy reporter can't take down the dispatch
	// loop.
	OnTransportPanic func(err *RecoveredPanicError)

	// Source configures call-site capture (file/line/function) per emission.
	// Off by default; see [SourceConfig] for the cost.
	Source SourceConfig

	// Routing configures group-based dispatch (named routing rules,
	// active-groups filter, behavior for ungrouped entries). The zero
	// value disables group routing entirely (every transport receives
	// every entry).
	Routing RoutingConfig
}

// SourceConfig configures Config.Source: capture and render the call
// site of every emission. Paired so the boolean and the output key live
// next to each other rather than two unrelated top-level fields.
type SourceConfig struct {
	// Enabled captures the call site (file/line/function) of every log
	// emission via runtime.Caller and includes it in the assembled Data
	// under FieldName. Off by default; opt in for production-debuggable
	// output. Costs about 620 ns and 5 extra allocations per emission
	// on amd64 (see Benchmarks). Paid only when this is true; the
	// dispatch path is untouched otherwise.
	//
	// Adapters that already have source information (notably the slog
	// handler, which extracts it from slog.Record.PC) can supply it via
	// RawLogEntry.Source without setting Enabled.
	Enabled bool

	// FieldName is the key under which the captured Source is rendered
	// in the assembled Data. Defaults to "source" to match the slog
	// convention.
	FieldName string
}

// RoutingConfig configures Config.Routing: named groups + active-groups
// filter + behavior for ungrouped entries. The zero value disables
// group routing entirely (every transport receives every entry).
type RoutingConfig struct {
	// Groups defines named routing rules. Each group lists the transport
	// IDs it routes to, an optional minimum level, and an optional
	// disabled flag. Tag entries with a group via WithGroup to limit
	// dispatch to that group's transports.
	Groups map[string]LogGroup

	// ActiveGroups, when non-empty, restricts routing to only these
	// groups. Logs tagged with groups not in this list are dropped (or
	// fall back to Ungrouped if none of the entry's groups are active).
	// Nil/empty means "no filter: all defined groups are active".
	ActiveGroups []string

	// Ungrouped controls what happens to entries with no group tag
	// when Groups is configured. Zero value (Mode: UngroupedToAll)
	// preserves the no-routing behavior of every transport receiving
	// every ungrouped entry.
	Ungrouped UngroupedRouting
}

// LogGroup is a named routing rule.
type LogGroup struct {
	// Transports lists the IDs of transports this group routes to.
	// Required for the group to do anything.
	Transports []string

	// Level is the minimum log level for this group. Entries below this
	// level are dropped for this group's transports. Zero value means "no
	// per-group filter: all levels pass" (the levelIndex check rejects 0
	// as an unknown level).
	Level LogLevel

	// Disabled suppresses this group's routing when true. Entries tagged
	// only with disabled groups are dropped. Entries tagged with both a
	// disabled and an enabled group still route through the enabled one.
	//
	// (Contrast with an undefined group name in the tag list: if every
	// tag refers to an undefined group, the entry falls back to
	// UngroupedRouting. Disabled is "explicitly off"; undefined is
	// "treated as no tag.")
	Disabled bool
}

// UngroupedMode is the routing strategy for entries that have no group tag.
type UngroupedMode uint8

const (
	// UngroupedToAll routes ungrouped entries to every transport. Default.
	UngroupedToAll UngroupedMode = iota
	// UngroupedToNone drops ungrouped entries entirely.
	UngroupedToNone
	// UngroupedToTransports routes ungrouped entries only to the transport
	// IDs listed in UngroupedRouting.Transports.
	UngroupedToTransports
)

// UngroupedRouting controls how entries with no group tag are dispatched
// when group routing is configured.
type UngroupedRouting struct {
	// Mode selects the routing strategy. Zero value is UngroupedToAll.
	Mode UngroupedMode
	// Transports is the allowlist used when Mode == UngroupedToTransports.
	// Ignored for the other modes.
	Transports []string
}

// CopyMsgPolicy controls per-call whether ErrorOnly copies err.Error()
// into the log message. The zero value (CopyMsgDefault) defers to
// Config.CopyMsgOnOnlyError.
type CopyMsgPolicy uint8

const (
	// CopyMsgDefault uses Config.CopyMsgOnOnlyError. Zero value.
	CopyMsgDefault CopyMsgPolicy = iota
	// CopyMsgEnabled forces err.Error() to be copied as the log message
	// for this call, regardless of Config.CopyMsgOnOnlyError.
	CopyMsgEnabled
	// CopyMsgDisabled forces no message copy for this call, regardless of
	// Config.CopyMsgOnOnlyError.
	CopyMsgDisabled
)

// ErrorOnlyOpts are optional settings for the ErrorOnly method.
type ErrorOnlyOpts struct {
	// LogLevel overrides the default error level. Defaults to LogLevelError.
	LogLevel LogLevel

	// CopyMsg overrides Config.CopyMsgOnOnlyError for this call. Zero
	// value (CopyMsgDefault) keeps the config default.
	CopyMsg CopyMsgPolicy
}

// MetadataOnlyOpts are optional settings for the MetadataOnly method.
type MetadataOnlyOpts struct {
	// LogLevel overrides the default info level. Defaults to LogLevelInfo.
	LogLevel LogLevel
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
	// Groups overrides the logger's assigned group tags for routing. Nil
	// uses the logger's groups (set via WithGroup).
	Groups []string
	// Source overrides the captured source info for this entry. Set this
	// from adapters that already have source info (e.g. the slog handler
	// extracts it from slog.Record.PC). Nil falls back to runtime capture
	// when Config.Source.Enabled is true; otherwise no source info is recorded.
	Source *Source
}
