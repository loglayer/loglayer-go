package loglayer

import (
	"context"
	"fmt"
	"os"

	"go.loglayer.dev/v2/utils/idgen"
)

// Plugin is the base contract every plugin satisfies. A plugin participates
// in zero or more lifecycle hooks by also implementing one or more of the
// hook interfaces below ([FieldsHook], [MetadataHook], [DataHook],
// [MessageHook], [LevelHook], [SendGate]).
//
// Plugins are added via *LogLayer.AddPlugin and identified by ID. Adding a
// plugin with an ID that's already registered replaces the previous one
// (matches the AddTransport convention). When ID returns the empty string,
// the framework assigns an auto-generated identifier at registration time;
// supply your own when you intend to call RemovePlugin / GetPlugin or
// replace the plugin later.
//
// Hook ordering: plugins run in the order they were added.
type Plugin interface {
	ID() string
}

// FieldsHook fires when *LogLayer.WithFields is called. It receives the
// fields about to be merged onto the derived logger and returns the fields
// to merge instead. Return nil to drop the WithFields call (the receiver's
// existing fields are preserved either way). Multiple plugins chain: each
// receives the previous plugin's output.
type FieldsHook interface {
	OnFieldsCalled(fields Fields) Fields
}

// MetadataHook fires from WithMetadata and MetadataOnly. It receives the
// metadata value the user passed (which may be a map, struct, scalar, or
// nil) and returns the metadata to use. Return nil to drop the metadata
// entirely for this entry.
type MetadataHook interface {
	OnMetadataCalled(metadata any) any
}

// DataHook fires per-emission, after the assembled data map (fields +
// serialized error) is built but before the entry reaches transports.
// Return the data to merge in; nil leaves the assembled data unchanged.
// The returned map is shallow-merged: keys present overwrite existing
// values; missing keys are left alone.
type DataHook interface {
	OnBeforeDataOut(BeforeDataOutParams) Data
}

// MessageHook fires per-emission, after [DataHook] and before [LevelHook].
// Return a replacement messages slice; nil leaves the messages unchanged.
type MessageHook interface {
	OnBeforeMessageOut(BeforeMessageOutParams) []any
}

// LevelHook fires per-emission, after [DataHook] and [MessageHook] but
// before per-transport dispatch. Return (level, true) to override the
// entry's level; (_, false) to leave it unchanged. If multiple plugins
// return ok=true, the last one wins.
type LevelHook interface {
	TransformLogLevel(TransformLogLevelParams) (LogLevel, bool)
}

// SendGate gates per-transport dispatch. Called once per (entry, transport)
// pair. Return false to skip dispatching that entry to that transport; the
// other transports are unaffected. If multiple plugins implement SendGate,
// the entry is sent only when every one returns true.
type SendGate interface {
	ShouldSend(ShouldSendParams) bool
}

// ErrorReporter is implemented by plugins that want to observe recovered
// panics in their own hooks. The framework recovers every hook panic so a
// buggy plugin can't tear down the calling goroutine; OnError lets the
// plugin observe the recovery (log it, increment a counter, etc.).
//
// If a plugin doesn't implement ErrorReporter, the framework writes a
// one-line description of the recovered panic to os.Stderr so it isn't
// silent.
//
// The error passed is a *RecoveredPanicError; the panicked hook is named
// in the error message and accessible via Hook / Value.
type ErrorReporter interface {
	OnError(err error)
}

// BeforeDataOutParams is the input to [DataHook.OnBeforeDataOut].
type BeforeDataOutParams struct {
	LogLevel LogLevel
	// Data is the assembled fields + error map. May be nil if the entry
	// has no fields and no error.
	Data Data
	// Fields is the logger's persistent fields, as the core sees them
	// (after OnFieldsCalled has already run at registration time).
	Fields Fields
	// Metadata is the value the user passed to WithMetadata, after any
	// OnMetadataCalled mutations.
	Metadata any
	// Err is the error attached via WithError, or nil.
	Err error
	// Ctx is the per-call context.Context attached via WithContext, or nil.
	Ctx context.Context
	// Groups mirrors [TransportParams.Groups].
	Groups []string
	// Schema mirrors [TransportParams.Schema].
	Schema Schema
	// Prefix mirrors [TransportParams.Prefix]: the value attached
	// via WithPrefix on the emitting logger (or set on Config.Prefix
	// at construction). Empty when no prefix was set. Read-only for
	// hooks; the framework propagates this value unchanged through
	// the dispatch path.
	Prefix string
}

// BeforeMessageOutParams is the input to [MessageHook.OnBeforeMessageOut].
type BeforeMessageOutParams struct {
	LogLevel LogLevel
	Messages []any
	// Ctx is the per-call context.Context attached via WithContext, or nil.
	Ctx context.Context
	// Groups mirrors [TransportParams.Groups].
	Groups []string
	// Schema mirrors [TransportParams.Schema].
	Schema Schema
	// Prefix mirrors [TransportParams.Prefix]: the value attached
	// via WithPrefix on the emitting logger (or set on Config.Prefix
	// at construction). Empty when no prefix was set. Read-only for
	// hooks; the framework propagates this value unchanged through
	// the dispatch path.
	Prefix string
}

// TransformLogLevelParams is the input to [LevelHook.TransformLogLevel].
type TransformLogLevelParams struct {
	LogLevel LogLevel
	Data     Data
	Messages []any
	Fields   Fields
	Metadata any
	Err      error
	// Ctx is the per-call context.Context attached via WithContext, or nil.
	Ctx context.Context
	// Groups mirrors [TransportParams.Groups].
	Groups []string
	// Schema mirrors [TransportParams.Schema].
	Schema Schema
	// Prefix mirrors [TransportParams.Prefix]: the value attached
	// via WithPrefix on the emitting logger (or set on Config.Prefix
	// at construction). Empty when no prefix was set. Read-only for
	// hooks; the framework propagates this value unchanged through
	// the dispatch path.
	Prefix string
}

// ShouldSendParams is the input to [SendGate.ShouldSend].
type ShouldSendParams struct {
	// TransportID is the ID of the transport this dispatch would target.
	// Use it to selectively gate per-transport (e.g. send debug to console
	// but not to the shipping transport).
	TransportID string
	LogLevel    LogLevel
	Messages    []any
	Data        Data
	Fields      Fields
	Metadata    any
	Err         error
	// Ctx is the per-call context.Context attached via WithContext, or nil.
	Ctx context.Context
	// Groups mirrors [TransportParams.Groups].
	Groups []string
	// Schema mirrors [TransportParams.Schema].
	Schema Schema
	// Prefix mirrors [TransportParams.Prefix]: the value attached
	// via WithPrefix on the emitting logger (or set on Config.Prefix
	// at construction). Empty when no prefix was set. Read-only for
	// hooks; the framework propagates this value unchanged through
	// the dispatch path.
	Prefix string
}

// inner is a private hook the framework uses to see through a wrapper
// (e.g. the one [WithErrorReporter] returns) to the underlying plugin.
// Hook interfaces are asserted against the inner plugin, so a wrapper
// only contributes the hooks the inner plugin implements; the wrapper's
// own added behavior (typically [ErrorReporter]) still resolves against
// the wrapper itself. Lowercase to keep this an internal extension
// point: only types defined in this package can satisfy it.
type pluginWithInner interface {
	inner() Plugin
}

// pluginEntry caches a registered plugin's resolved ID and the result of
// each hook-interface assertion, so the dispatch path doesn't re-assert.
// A nil cached field means the plugin does not implement that hook.
type pluginEntry struct {
	plugin     Plugin
	id         string
	onFields   FieldsHook
	onMetadata MetadataHook
	onData     DataHook
	onMessage  MessageHook
	onLevel    LevelHook
	sendGate   SendGate
	reporter   ErrorReporter
}

// pluginSet is an immutable snapshot of the registered plugins.
type pluginSet struct {
	entries []pluginEntry
	byID    map[string]int

	hasData, hasMessage, hasLevel, hasSendGate bool
	hasFields, hasMetadata                     bool
	// anyDispatchHook lets the dispatch hot path skip building
	// hook-param structs when no plugin would consume them.
	anyDispatchHook bool
}

func newPluginSet(plugins []Plugin) *pluginSet {
	s := &pluginSet{
		entries: make([]pluginEntry, len(plugins)),
		byID:    make(map[string]int, len(plugins)),
	}
	for i, p := range plugins {
		id := p.ID()
		if id == "" {
			id = idgen.Random(idgen.PluginPrefix)
		}
		entry := pluginEntry{plugin: p, id: id}
		// Hook assertions resolve against the inner plugin when present so
		// a wrapper only contributes hooks the inner actually implements.
		// ErrorReporter still resolves against the outer wrapper.
		hookTarget := p
		if w, ok := p.(pluginWithInner); ok {
			hookTarget = w.inner()
		}
		if h, ok := hookTarget.(FieldsHook); ok {
			entry.onFields = h
			s.hasFields = true
		}
		if h, ok := hookTarget.(MetadataHook); ok {
			entry.onMetadata = h
			s.hasMetadata = true
		}
		if h, ok := hookTarget.(DataHook); ok {
			entry.onData = h
			s.hasData = true
		}
		if h, ok := hookTarget.(MessageHook); ok {
			entry.onMessage = h
			s.hasMessage = true
		}
		if h, ok := hookTarget.(LevelHook); ok {
			entry.onLevel = h
			s.hasLevel = true
		}
		if g, ok := hookTarget.(SendGate); ok {
			entry.sendGate = g
			s.hasSendGate = true
		}
		if r, ok := p.(ErrorReporter); ok {
			entry.reporter = r
		}
		s.entries[i] = entry
		s.byID[id] = i
	}
	s.anyDispatchHook = s.hasData || s.hasMessage || s.hasLevel || s.hasSendGate
	return s
}

// Hook names used in [RecoveredPanicError.Hook] and the framework's
// stderr fallback message. Use the constants instead of raw strings when
// adding new dispatch sites.
const (
	hookFieldsCalled   = "OnFieldsCalled"
	hookMetadataCalled = "OnMetadataCalled"
	hookBeforeDataOut  = "OnBeforeDataOut"
	hookBeforeMsgOut   = "OnBeforeMessageOut"
	hookTransformLevel = "TransformLogLevel"
	hookShouldSend     = "ShouldSend"
)

// PanicKind values for [RecoveredPanicError.Kind].
const (
	// PanicKindPlugin marks a panic recovered from a plugin hook.
	// ID is the plugin's ID; Plugin carries the hook method name.
	PanicKindPlugin = "plugin"
	// PanicKindTransport marks a panic recovered from a transport's
	// SendToLogger. ID is the transport ID; Plugin is nil.
	PanicKindTransport = "transport"
)

// RecoveredPanicError is the error type produced by the framework's
// centralized panic recovery (plugin hooks via [ErrorReporter.OnError],
// and transport SendToLogger via [Config.OnTransportPanic]).
//
// Kind identifies the category ([PanicKindPlugin] or [PanicKindTransport]).
// ID is the panicking component's identifier — the plugin ID for plugins,
// the transport ID for transports. Plugin carries plugin-specific details
// (the hook method name) and is non-nil iff Kind == PanicKindPlugin; for
// transport panics it is nil so the absence of a hook-method dimension
// is a typed condition rather than an empty-string convention.
//
// Value is the value originally passed to panic(). When Value satisfies
// the error interface, errors.Unwrap reaches it (and errors.Is /
// errors.As work transparently); when it doesn't, read Value directly to
// inspect the concrete type.
type RecoveredPanicError struct {
	Kind    string
	ID      string
	Plugin  *PluginPanicDetails
	Value   any
	wrapped error // set when Value implements error
}

// PluginPanicDetails carries plugin-specific information attached to a
// [RecoveredPanicError]. Non-nil iff Kind == PanicKindPlugin.
type PluginPanicDetails struct {
	// Hook is the hook method that panicked, e.g. "OnBeforeDataOut".
	Hook string
}

func (e *RecoveredPanicError) Error() string {
	if e.Plugin != nil {
		return fmt.Sprintf("loglayer: %s %q hook %s panicked: %v", e.Kind, e.ID, e.Plugin.Hook, e.Value)
	}
	return fmt.Sprintf("loglayer: %s %q panicked: %v", e.Kind, e.ID, e.Value)
}

func (e *RecoveredPanicError) Unwrap() error { return e.wrapped }

func panicError(r any, kind, id string, plugin *PluginPanicDetails) *RecoveredPanicError {
	pe := &RecoveredPanicError{Kind: kind, ID: id, Plugin: plugin, Value: r}
	if e, ok := r.(error); ok {
		pe.wrapped = e
	}
	return pe
}

// recoverHook is the canonical deferred recovery for plugin hook calls.
// Reports recovered panics via the plugin's [ErrorReporter] when one is
// implemented, else writes a one-line description to os.Stderr so the
// failure isn't silent.
//
// Defer it directly: `defer recoverHook(entry.reporter, entry.id, "HookName")`.
func recoverHook(reporter ErrorReporter, pluginID, hook string) {
	r := recover()
	if r == nil {
		return
	}
	err := panicError(r, PanicKindPlugin, pluginID, &PluginPanicDetails{Hook: hook})
	if reporter != nil {
		reporter.OnError(err)
		return
	}
	fmt.Fprintln(os.Stderr, err)
}

func callBeforeDataOut(e *pluginEntry, p BeforeDataOutParams) (out Data) {
	defer recoverHook(e.reporter, e.id, hookBeforeDataOut)
	return e.onData.OnBeforeDataOut(p)
}

func callBeforeMessageOut(e *pluginEntry, p BeforeMessageOutParams) (out []any) {
	defer recoverHook(e.reporter, e.id, hookBeforeMsgOut)
	return e.onMessage.OnBeforeMessageOut(p)
}

func callTransformLogLevel(e *pluginEntry, p TransformLogLevelParams) (level LogLevel, ok bool) {
	defer recoverHook(e.reporter, e.id, hookTransformLevel)
	return e.onLevel.TransformLogLevel(p)
}

// callShouldSend fails open: a panicking ShouldSend returns true so the
// entry still dispatches. Silent dropping would mask plugin bugs as data
// loss; ErrorReporter surfaces the panic for operators to fix.
func callShouldSend(e *pluginEntry, p ShouldSendParams) (ok bool) {
	ok = true
	defer recoverHook(e.reporter, e.id, hookShouldSend)
	return e.sendGate.ShouldSend(p)
}

func callOnMetadataCalled(e *pluginEntry, metadata any) (out any) {
	defer recoverHook(e.reporter, e.id, hookMetadataCalled)
	return e.onMetadata.OnMetadataCalled(metadata)
}

func callOnFieldsCalled(e *pluginEntry, fields Fields) (out Fields) {
	defer recoverHook(e.reporter, e.id, hookFieldsCalled)
	return e.onFields.OnFieldsCalled(fields)
}

func (s *pluginSet) runOnBeforeDataOut(p BeforeDataOutParams) Data {
	if !s.hasData {
		return p.Data
	}
	// p is by-value; mutate p.Data so each plugin sees the running merge.
	for i := range s.entries {
		e := &s.entries[i]
		if e.onData == nil {
			continue
		}
		patch := callBeforeDataOut(e, p)
		if len(patch) == 0 {
			continue
		}
		if p.Data == nil {
			p.Data = make(Data, len(patch))
		}
		for k, v := range patch {
			p.Data[k] = v
		}
	}
	return p.Data
}

func (s *pluginSet) runOnBeforeMessageOut(p BeforeMessageOutParams) []any {
	if !s.hasMessage {
		return p.Messages
	}
	for i := range s.entries {
		e := &s.entries[i]
		if e.onMessage == nil {
			continue
		}
		if next := callBeforeMessageOut(e, p); next != nil {
			p.Messages = next
		}
	}
	return p.Messages
}

func (s *pluginSet) runTransformLogLevel(p TransformLogLevelParams) LogLevel {
	if !s.hasLevel {
		return p.LogLevel
	}
	level := p.LogLevel
	for i := range s.entries {
		e := &s.entries[i]
		if e.onLevel == nil {
			continue
		}
		if next, ok := callTransformLogLevel(e, p); ok {
			level = next
		}
	}
	return level
}

func (s *pluginSet) runShouldSend(p ShouldSendParams) bool {
	if !s.hasSendGate {
		return true
	}
	for i := range s.entries {
		e := &s.entries[i]
		if e.sendGate == nil {
			continue
		}
		if !callShouldSend(e, p) {
			return false
		}
	}
	return true
}

func (s *pluginSet) runOnMetadataCalled(metadata any) any {
	if !s.hasMetadata {
		return metadata
	}
	out := metadata
	for i := range s.entries {
		e := &s.entries[i]
		if e.onMetadata == nil {
			continue
		}
		out = callOnMetadataCalled(e, out)
		if out == nil {
			return nil
		}
	}
	return out
}

func (s *pluginSet) runOnFieldsCalled(fields Fields) Fields {
	if !s.hasFields {
		return fields
	}
	out := fields
	for i := range s.entries {
		e := &s.entries[i]
		if e.onFields == nil {
			continue
		}
		out = callOnFieldsCalled(e, out)
		if out == nil {
			return nil
		}
	}
	return out
}

// Adapter constructors for inline single-hook plugins. Use these when you
// don't want to declare a type for a one-off plugin. For multi-hook plugins,
// declare your own type implementing [Plugin] plus the relevant hook
// interfaces.
//
// Each constructor returns an unexported type that implements [Plugin]
// plus the named hook interface. ID auto-generates when empty.

// NewPlugin returns a Plugin that implements no hook interfaces. Useful
// for tests that exercise registration/replacement/removal semantics
// without needing actual hook behavior.
func NewPlugin(id string) Plugin { return &noopPlugin{id: id} }

// WithErrorReporter wraps p with an [ErrorReporter] backed by onError.
// Hook dispatch goes to p exactly as if it were registered directly; the
// framework recognises the wrapper internally and resolves hook
// interfaces against p, not the wrapper. The wrapper only contributes
// the [ErrorReporter] behavior, so panics in p's hooks reach onError
// instead of the default stderr path.
//
// Returns p unchanged when onError is nil.
func WithErrorReporter(p Plugin, onError func(error)) Plugin {
	if onError == nil {
		return p
	}
	return &reporterWrapper{p: p, onError: onError}
}

type reporterWrapper struct {
	p       Plugin
	onError func(error)
}

func (r *reporterWrapper) ID() string        { return r.p.ID() }
func (r *reporterWrapper) inner() Plugin     { return r.p }
func (r *reporterWrapper) OnError(err error) { r.onError(err) }

type noopPlugin struct{ id string }

func (n *noopPlugin) ID() string { return n.id }

// NewFieldsHook returns a Plugin that implements [FieldsHook] only.
func NewFieldsHook(id string, fn func(Fields) Fields) Plugin {
	return &fieldsHookFn{id: id, fn: fn}
}

// NewMetadataHook returns a Plugin that implements [MetadataHook] only.
func NewMetadataHook(id string, fn func(any) any) Plugin {
	return &metadataHookFn{id: id, fn: fn}
}

// NewDataHook returns a Plugin that implements [DataHook] only.
func NewDataHook(id string, fn func(BeforeDataOutParams) Data) Plugin {
	return &dataHookFn{id: id, fn: fn}
}

// NewMessageHook returns a Plugin that implements [MessageHook] only.
func NewMessageHook(id string, fn func(BeforeMessageOutParams) []any) Plugin {
	return &messageHookFn{id: id, fn: fn}
}

// NewLevelHook returns a Plugin that implements [LevelHook] only.
func NewLevelHook(id string, fn func(TransformLogLevelParams) (LogLevel, bool)) Plugin {
	return &levelHookFn{id: id, fn: fn}
}

// NewSendGate returns a Plugin that implements [SendGate] only.
func NewSendGate(id string, fn func(ShouldSendParams) bool) Plugin {
	return &sendGateFn{id: id, fn: fn}
}

type fieldsHookFn struct {
	id string
	fn func(Fields) Fields
}

func (f *fieldsHookFn) ID() string                      { return f.id }
func (f *fieldsHookFn) OnFieldsCalled(in Fields) Fields { return f.fn(in) }

type metadataHookFn struct {
	id string
	fn func(any) any
}

func (m *metadataHookFn) ID() string                  { return m.id }
func (m *metadataHookFn) OnMetadataCalled(in any) any { return m.fn(in) }

type dataHookFn struct {
	id string
	fn func(BeforeDataOutParams) Data
}

func (d *dataHookFn) ID() string                                 { return d.id }
func (d *dataHookFn) OnBeforeDataOut(p BeforeDataOutParams) Data { return d.fn(p) }

type messageHookFn struct {
	id string
	fn func(BeforeMessageOutParams) []any
}

func (m *messageHookFn) ID() string                                        { return m.id }
func (m *messageHookFn) OnBeforeMessageOut(p BeforeMessageOutParams) []any { return m.fn(p) }

type levelHookFn struct {
	id string
	fn func(TransformLogLevelParams) (LogLevel, bool)
}

func (l *levelHookFn) ID() string { return l.id }
func (l *levelHookFn) TransformLogLevel(p TransformLogLevelParams) (LogLevel, bool) {
	return l.fn(p)
}

type sendGateFn struct {
	id string
	fn func(ShouldSendParams) bool
}

func (s *sendGateFn) ID() string                         { return s.id }
func (s *sendGateFn) ShouldSend(p ShouldSendParams) bool { return s.fn(p) }
