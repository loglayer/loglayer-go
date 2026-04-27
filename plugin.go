package loglayer

import (
	"context"
	"fmt"
)

// Plugin is a unit of logic that runs at lifecycle points during emission.
// Populate the function fields that match the hooks you want to participate
// in; nil fields are skipped.
//
// Plugins are added via *LogLayer.AddPlugin and identified by ID. Adding a
// plugin with an ID that's already registered replaces the previous one
// (matches the AddTransport convention).
//
// Hook ordering: plugins run in the order they were added.
//
// The Plugin struct is consumed by-value at registration time. Mutating
// the function fields after AddPlugin has no effect on registered behavior;
// to update a plugin, construct a new Plugin and AddPlugin it again.
type Plugin struct {
	// ID uniquely identifies the plugin. Required for RemovePlugin and
	// for replacement-on-readd semantics.
	ID string

	// OnBeforeDataOut runs after the assembled data map is built (fields +
	// error). Return the data map to use; nil leaves the assembled data
	// unchanged. The returned map is shallow-merged into the entry's data:
	// keys present in the returned map overwrite existing values; keys not
	// present are left alone.
	OnBeforeDataOut func(BeforeDataOutParams) Data

	// OnBeforeMessageOut runs after OnBeforeDataOut and before
	// TransformLogLevel. Return a replacement messages slice; nil leaves
	// the messages unchanged.
	OnBeforeMessageOut func(BeforeMessageOutParams) []any

	// TransformLogLevel runs after OnBeforeDataOut and OnBeforeMessageOut
	// but before per-transport dispatch. Return (level, true) to override
	// the entry's level; (_, false) to leave it unchanged.
	//
	// If multiple plugins return ok=true, the last one wins.
	TransformLogLevel func(TransformLogLevelParams) (LogLevel, bool)

	// ShouldSend gates per-transport dispatch. Called once per (entry,
	// transport) pair. Return false to skip dispatching that entry to that
	// transport; the other transports are unaffected.
	//
	// If multiple plugins define ShouldSend, the entry is sent only when
	// every plugin returns true.
	ShouldSend func(ShouldSendParams) bool

	// OnMetadataCalled fires from WithMetadata and MetadataOnly. Return the
	// metadata to use; nil drops metadata entirely for this entry.
	//
	// Multiple plugins chain: each receives the previous plugin's output.
	OnMetadataCalled func(metadata any) any

	// OnFieldsCalled fires from WithFields. Receives the fields about to be
	// merged. Return the fields to merge; nil drops the WithFields call
	// entirely (the receiver's existing fields are preserved either way).
	//
	// Multiple plugins chain: each receives the previous plugin's output.
	OnFieldsCalled func(fields Fields) Fields

	// OnError is invoked when one of this plugin's hook functions panics
	// during emission. The framework always recovers hook panics so a
	// buggy plugin can't tear down the caller's goroutine; OnError lets
	// the plugin observe the recovered panic (log it, increment a
	// counter, etc.). nil means "swallow silently" (the default).
	//
	// The error passed to OnError is either the recovered value as-is
	// (when it implements error) or a fmt-wrapped form of it. The hook
	// that panicked is identified in the error message.
	OnError func(err error)
}

// BeforeDataOutParams is the input to OnBeforeDataOut.
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
	// Ctx is the per-call context.Context attached via WithCtx, or nil.
	Ctx context.Context
}

// BeforeMessageOutParams is the input to OnBeforeMessageOut.
type BeforeMessageOutParams struct {
	LogLevel LogLevel
	Messages []any
	// Ctx is the per-call context.Context attached via WithCtx, or nil.
	Ctx context.Context
}

// TransformLogLevelParams is the input to TransformLogLevel.
type TransformLogLevelParams struct {
	LogLevel LogLevel
	Data     Data
	Messages []any
	Fields   Fields
	Metadata any
	Err      error
	// Ctx is the per-call context.Context attached via WithCtx, or nil.
	Ctx context.Context
}

// ShouldSendParams is the input to ShouldSend.
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
	// Ctx is the per-call context.Context attached via WithCtx, or nil.
	Ctx context.Context
}

// pluginSet is an immutable snapshot of the plugins. Hook lists are
// pre-indexed at construction time so the dispatch path doesn't pay for
// nil-checks; it just iterates the per-hook slice.
type pluginSet struct {
	all               []Plugin
	byID              map[string]int // index into all
	beforeDataOut     []int
	beforeMessageOut  []int
	transformLogLevel []int
	shouldSend        []int
	onMetadataCalled  []int
	onFieldsCalled    []int
	// anyDispatchHook is true when at least one plugin defines a hook that
	// fires from processLog (OnBeforeDataOut, OnBeforeMessageOut,
	// TransformLogLevel, or ShouldSend). The dispatch hot path uses this
	// to skip building hook-param structs when no plugin would consume them.
	anyDispatchHook bool
}

func newPluginSet(plugins []Plugin) *pluginSet {
	s := &pluginSet{
		all:  plugins,
		byID: make(map[string]int, len(plugins)),
	}
	for i, p := range plugins {
		s.byID[p.ID] = i
		if p.OnBeforeDataOut != nil {
			s.beforeDataOut = append(s.beforeDataOut, i)
		}
		if p.OnBeforeMessageOut != nil {
			s.beforeMessageOut = append(s.beforeMessageOut, i)
		}
		if p.TransformLogLevel != nil {
			s.transformLogLevel = append(s.transformLogLevel, i)
		}
		if p.ShouldSend != nil {
			s.shouldSend = append(s.shouldSend, i)
		}
		if p.OnMetadataCalled != nil {
			s.onMetadataCalled = append(s.onMetadataCalled, i)
		}
		if p.OnFieldsCalled != nil {
			s.onFieldsCalled = append(s.onFieldsCalled, i)
		}
	}
	s.anyDispatchHook = len(s.beforeDataOut)+len(s.beforeMessageOut)+len(s.transformLogLevel)+len(s.shouldSend) > 0
	return s
}

// panicError wraps a recovered panic value in an error tagged with the
// hook name. Used by recoverHook to give plugin authors context when
// their OnError fires.
func panicError(r any, hook string) error {
	if e, ok := r.(error); ok {
		return fmt.Errorf("loglayer: plugin %s panicked: %w", hook, e)
	}
	return fmt.Errorf("loglayer: plugin %s panicked: %v", hook, r)
}

// recoverHook is the canonical deferred recovery for plugin hook calls.
// Defer it directly: `defer recoverHook(plugin.OnError, "HookName")`.
func recoverHook(onErr func(error), hook string) {
	if r := recover(); r != nil && onErr != nil {
		onErr(panicError(r, hook))
	}
}

func (s *pluginSet) callBeforeDataOut(i int, p BeforeDataOutParams) (out Data) {
	defer recoverHook(s.all[i].OnError, "OnBeforeDataOut")
	return s.all[i].OnBeforeDataOut(p)
}

func (s *pluginSet) callBeforeMessageOut(i int, p BeforeMessageOutParams) (out []any) {
	defer recoverHook(s.all[i].OnError, "OnBeforeMessageOut")
	return s.all[i].OnBeforeMessageOut(p)
}

func (s *pluginSet) callTransformLogLevel(i int, p TransformLogLevelParams) (level LogLevel, ok bool) {
	defer recoverHook(s.all[i].OnError, "TransformLogLevel")
	return s.all[i].TransformLogLevel(p)
}

// callShouldSend fails open: a panicking ShouldSend returns true so the
// entry still dispatches. Silent dropping would mask plugin bugs as data
// loss; OnError surfaces the panic for operators to fix.
func (s *pluginSet) callShouldSend(i int, p ShouldSendParams) (ok bool) {
	ok = true
	defer recoverHook(s.all[i].OnError, "ShouldSend")
	return s.all[i].ShouldSend(p)
}

func (s *pluginSet) callOnMetadataCalled(i int, metadata any) (out any) {
	defer recoverHook(s.all[i].OnError, "OnMetadataCalled")
	return s.all[i].OnMetadataCalled(metadata)
}

func (s *pluginSet) callOnFieldsCalled(i int, fields Fields) (out Fields) {
	defer recoverHook(s.all[i].OnError, "OnFieldsCalled")
	return s.all[i].OnFieldsCalled(fields)
}

func (s *pluginSet) runOnBeforeDataOut(p BeforeDataOutParams) Data {
	if len(s.beforeDataOut) == 0 {
		return p.Data
	}
	out := p.Data
	for _, i := range s.beforeDataOut {
		patch := s.callBeforeDataOut(i, BeforeDataOutParams{
			LogLevel: p.LogLevel,
			Data:     out,
			Fields:   p.Fields,
			Metadata: p.Metadata,
			Err:      p.Err,
			Ctx:      p.Ctx,
		})
		if patch == nil {
			continue
		}
		if out == nil {
			out = make(Data, len(patch))
		}
		for k, v := range patch {
			out[k] = v
		}
	}
	return out
}

func (s *pluginSet) runOnBeforeMessageOut(p BeforeMessageOutParams) []any {
	if len(s.beforeMessageOut) == 0 {
		return p.Messages
	}
	msgs := p.Messages
	for _, i := range s.beforeMessageOut {
		next := s.callBeforeMessageOut(i, BeforeMessageOutParams{
			LogLevel: p.LogLevel,
			Messages: msgs,
			Ctx:      p.Ctx,
		})
		if next != nil {
			msgs = next
		}
	}
	return msgs
}

func (s *pluginSet) runTransformLogLevel(p TransformLogLevelParams) LogLevel {
	if len(s.transformLogLevel) == 0 {
		return p.LogLevel
	}
	level := p.LogLevel
	for _, i := range s.transformLogLevel {
		if next, ok := s.callTransformLogLevel(i, p); ok {
			level = next
		}
	}
	return level
}

func (s *pluginSet) runShouldSend(p ShouldSendParams) bool {
	for _, i := range s.shouldSend {
		if !s.callShouldSend(i, p) {
			return false
		}
	}
	return true
}

func (s *pluginSet) runOnMetadataCalled(metadata any) any {
	if len(s.onMetadataCalled) == 0 {
		return metadata
	}
	out := metadata
	for _, i := range s.onMetadataCalled {
		out = s.callOnMetadataCalled(i, out)
		if out == nil {
			return nil
		}
	}
	return out
}

func (s *pluginSet) runOnFieldsCalled(fields Fields) Fields {
	if len(s.onFieldsCalled) == 0 {
		return fields
	}
	out := fields
	for _, i := range s.onFieldsCalled {
		out = s.callOnFieldsCalled(i, out)
		if out == nil {
			return nil
		}
	}
	return out
}

// MetadataPlugin returns a Plugin with only OnMetadataCalled set. Sugar
// for the common single-hook case; equivalent to:
//
//	loglayer.Plugin{ID: id, OnMetadataCalled: fn}
func MetadataPlugin(id string, fn func(metadata any) any) Plugin {
	return Plugin{ID: id, OnMetadataCalled: fn}
}

// FieldsPlugin returns a Plugin with only OnFieldsCalled set. Sugar for
// the common single-hook case; equivalent to:
//
//	loglayer.Plugin{ID: id, OnFieldsCalled: fn}
func FieldsPlugin(id string, fn func(fields Fields) Fields) Plugin {
	return Plugin{ID: id, OnFieldsCalled: fn}
}

// LevelPlugin returns a Plugin with only TransformLogLevel set. Sugar
// for the common single-hook case (e.g., "promote entries that carry an
// error key to LogLevelError"); equivalent to:
//
//	loglayer.Plugin{ID: id, TransformLogLevel: fn}
func LevelPlugin(id string, fn func(TransformLogLevelParams) (LogLevel, bool)) Plugin {
	return Plugin{ID: id, TransformLogLevel: fn}
}
