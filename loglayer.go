// Package loglayer provides a transport-agnostic logging abstraction that routes
// structured log entries to one or more backend transports.
package loglayer

import (
	"context"
	"sync"
	"sync/atomic"
)

// Transport is the interface that all LogLayer transports must implement.
type Transport interface {
	// ID returns the unique identifier for this transport.
	ID() string

	// IsEnabled returns whether the transport is currently active.
	IsEnabled() bool

	// SendToLogger receives a fully assembled log entry and dispatches it.
	// Implementations should perform their own level filtering if needed.
	SendToLogger(params TransportParams)

	// GetLoggerInstance returns the underlying logger instance, if any.
	// Returns nil for transports that have no underlying library.
	GetLoggerInstance() any
}

// TransportParams is the fully assembled log entry passed to each transport.
//
// Data contains the assembled fields and error.
// Metadata carries the raw value passed to WithMetadata; the transport is
// responsible for serializing it in whatever way suits the underlying library.
type TransportParams struct {
	LogLevel LogLevel
	Messages []any
	// Data holds the assembled persistent fields and error. Nil when no
	// fields are set and no error is attached. Use len(Data) > 0 to test.
	Data Data
	// Metadata is the raw value passed to WithMetadata. May be a struct, map,
	// or any other type. Nil if WithMetadata was not called or metadata is muted.
	Metadata any
	Err      error
	// Fields is the logger's persistent key/value bag, raw (not yet folded into Data).
	Fields Fields
	// Ctx is the per-call context.Context attached via WithCtx, if any.
	// Transports can use it to extract trace IDs, span context, deadlines, etc.
	// Nil when no Go context was attached.
	Ctx context.Context
}

// transportSet is an immutable snapshot of the transport list and the
// id-keyed lookup. New snapshots are published atomically by the mutators;
// the dispatch path loads the current snapshot and iterates without locking.
type transportSet struct {
	list []Transport
	byID map[string]Transport
}

// LogLayer is the central logger. It assembles log entries from fields, metadata,
// and error data, then dispatches them to one or more transports.
type LogLayer struct {
	config       Config
	fields       Fields
	levels       *levelState
	transports   atomic.Pointer[transportSet]
	plugins      atomic.Pointer[pluginSet]
	muteFields   atomic.Bool
	muteMetadata atomic.Bool
	// txMu serializes transport mutators (AddTransport / RemoveTransport /
	// SetTransports) so two concurrent admin operations on the same
	// logger don't lose updates. The dispatch path doesn't take this lock;
	// it just Loads the current snapshot.
	txMu sync.Mutex
	// pluginMu serializes plugin mutators (AddPlugin / RemovePlugin); same
	// pattern as txMu.
	pluginMu sync.Mutex
}

// New creates a new LogLayer from the given Config.
//
// Panics if no transport is provided. For applications that prefer explicit
// error handling on misconfiguration, use Build instead.
func New(config Config) *LogLayer {
	l, err := build(config)
	if err != nil {
		panic(err)
	}
	return l
}

// Build creates a new LogLayer from the given Config, returning an error
// instead of panicking if the configuration is invalid (e.g. no transport).
//
// Use New for the more concise idiom when misconfiguration is a programmer
// error (the typical case for application setup).
func Build(config Config) (*LogLayer, error) {
	return build(config)
}

func build(config Config) (*LogLayer, error) {
	if config.Transport != nil && len(config.Transports) > 0 {
		return nil, ErrTransportAndTransports
	}
	all := config.Transports
	if config.Transport != nil {
		all = []Transport{config.Transport}
	}
	if len(all) == 0 {
		return nil, ErrNoTransport
	}

	l := &LogLayer{
		config: config,
		fields: make(Fields),
		levels: newLevelState(),
	}

	if config.ErrorFieldName == "" {
		l.config.ErrorFieldName = "err"
	}

	if config.Disabled {
		l.levels.setMaster(false)
	}

	l.muteFields.Store(config.MuteFields)
	l.muteMetadata.Store(config.MuteMetadata)
	l.transports.Store(newTransportSet(all))

	for _, p := range config.Plugins {
		if p.ID == "" {
			return nil, ErrPluginNoID
		}
	}
	l.plugins.Store(newPluginSet(append([]Plugin(nil), config.Plugins...)))

	return l, nil
}

// newTransportSet builds an immutable transportSet snapshot. Caller must
// ensure all is non-empty.
func newTransportSet(all []Transport) *transportSet {
	set := &transportSet{
		list: all,
		byID: make(map[string]Transport, len(all)),
	}
	for _, t := range all {
		set.byID[t.ID()] = t
	}
	return set
}

// publishTransports validates and atomically swaps in a new transport set.
// Used by every mutator after building the new slice. Panics if all is empty.
func (l *LogLayer) publishTransports(all []Transport) {
	if len(all) == 0 {
		panic(ErrNoTransport)
	}
	l.transports.Store(newTransportSet(all))
}

// loadTransports returns the current transport snapshot. Hot path: called on
// every emission.
func (l *LogLayer) loadTransports() *transportSet {
	return l.transports.Load()
}

// Child creates a new LogLayer that inherits the current config, fields (shallow copy),
// level state, transports, and plugins. Changes to the child do not affect the parent.
func (l *LogLayer) Child() *LogLayer {
	parentSet := l.loadTransports()
	transports := make([]Transport, len(parentSet.list))
	copy(transports, parentSet.list)
	child := &LogLayer{
		config: l.config,
		fields: copyFields(l.fields),
		levels: l.levels.clone(),
	}
	child.muteFields.Store(l.muteFields.Load())
	child.muteMetadata.Store(l.muteMetadata.Load())
	child.publishTransports(transports)
	// pluginSet is immutable; mutators publish a new set via copy-on-write,
	// so child can share the parent's snapshot until either side mutates.
	child.plugins.Store(l.loadPlugins())
	return child
}

// WithPrefix creates a child logger with the given prefix prepended to every message.
func (l *LogLayer) WithPrefix(prefix string) *LogLayer {
	child := l.Child()
	child.config.Prefix = prefix
	return child
}

func applyPrefix(prefix string, messages []any) {
	if prefix == "" || len(messages) == 0 {
		return
	}
	if s, ok := messages[0].(string); ok {
		messages[0] = prefix + " " + s
	}
}

func copyFields(src Fields) Fields {
	dst := make(Fields, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
