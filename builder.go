package loglayer

import "context"

// LogBuilder accumulates per-log metadata, error, and context.Context before
// dispatching to a log level method. Obtain one via LogLayer.WithMetadata,
// LogLayer.WithError, or LogLayer.WithCtx.
//
// LogBuilders are intended to be single-use and stack-allocated. Build, chain,
// and terminate inline:
//
//	log.WithCtx(ctx).WithMetadata(meta).WithError(err).Error("failed")
//
// Holding a *LogBuilder past its terminal call works but discards the
// stack-allocation benefit.
type LogBuilder struct {
	layer    *LogLayer
	metadata any
	err      error
	ctx      context.Context
	groups   []string // per-call group tags from WithGroup
}

func newLogBuilder(l *LogLayer) *LogBuilder {
	return &LogBuilder{layer: l}
}

// WithMetadata attaches metadata to the log entry. Accepts any value: a struct,
// a map, or any other type. Serialization is handled by the transport.
// Calling this multiple times replaces the previous value.
//
// OnMetadataCalled plugin hooks run here. A hook returning nil drops the
// metadata entirely for this entry.
func (b *LogBuilder) WithMetadata(v any) *LogBuilder {
	b.metadata = b.layer.loadPlugins().runOnMetadataCalled(v)
	return b
}

// WithError attaches an error to the log entry.
func (b *LogBuilder) WithError(err error) *LogBuilder {
	b.err = err
	return b
}

// WithCtx attaches a context.Context to this single log entry, overriding
// any context bound to the parent logger via (*LogLayer).WithCtx for this
// emission only.
//
// For the persistent variant (bind once, every subsequent emission carries
// the ctx), use (*LogLayer).WithCtx instead.
//
// Passing nil clears any per-call ctx previously set on this builder.
// On a fresh builder it has no observable effect (the layer's bound ctx,
// if any, still applies on dispatch).
func (b *LogBuilder) WithCtx(ctx context.Context) *LogBuilder {
	b.ctx = ctx
	return b
}

// WithGroup tags this single log entry with one or more group names.
// Routing rules in Config.Groups decide which transports receive the
// entry. Tags are merged with any persistent groups assigned via
// (*LogLayer).WithGroup.
//
// Calling this multiple times accumulates groups (deduplicated).
func (b *LogBuilder) WithGroup(groups ...string) *LogBuilder {
	if len(groups) == 0 {
		return b
	}
	merged := mergeGroups(b.groups, groups)
	// Detach from the caller's variadic backing on the first WithGroup
	// call so a mutation between WithGroup and the terminal level call
	// can't leak through.
	if len(b.groups) == 0 {
		merged = append([]string(nil), merged...)
	}
	b.groups = merged
	return b
}

// Info dispatches the accumulated entry at the info level.
func (b *LogBuilder) Info(messages ...any) {
	if !b.layer.levels.isEnabled(LogLevelInfo) {
		return
	}
	b.dispatch(LogLevelInfo, messages)
}

// Warn dispatches the accumulated entry at the warn level.
func (b *LogBuilder) Warn(messages ...any) {
	if !b.layer.levels.isEnabled(LogLevelWarn) {
		return
	}
	b.dispatch(LogLevelWarn, messages)
}

// Error dispatches the accumulated entry at the error level.
func (b *LogBuilder) Error(messages ...any) {
	if !b.layer.levels.isEnabled(LogLevelError) {
		return
	}
	b.dispatch(LogLevelError, messages)
}

// Debug dispatches the accumulated entry at the debug level.
func (b *LogBuilder) Debug(messages ...any) {
	if !b.layer.levels.isEnabled(LogLevelDebug) {
		return
	}
	b.dispatch(LogLevelDebug, messages)
}

// Trace dispatches the accumulated entry at the trace level.
func (b *LogBuilder) Trace(messages ...any) {
	if !b.layer.levels.isEnabled(LogLevelTrace) {
		return
	}
	b.dispatch(LogLevelTrace, messages)
}

// Fatal dispatches the accumulated entry at the fatal level.
// Calls os.Exit(1) after dispatch unless Config.DisableFatalExit is set.
func (b *LogBuilder) Fatal(messages ...any) {
	if !b.layer.levels.isEnabled(LogLevelFatal) {
		return
	}
	b.dispatch(LogLevelFatal, messages)
}

func (b *LogBuilder) dispatch(level LogLevel, messages []any) {
	applyPrefix(b.layer.config.Prefix, messages)
	// Hot path: builder has no per-call groups → pass the layer's
	// assigned groups straight through. mergeGroups is out-of-line and
	// would be a measurable hit per emission for the dominant case.
	groups := b.layer.assignedGroups
	if len(b.groups) > 0 {
		groups = mergeGroups(groups, b.groups)
	}
	// Per-call WithCtx on the builder overrides the layer's bound ctx.
	ctx := b.ctx
	if ctx == nil {
		ctx = b.layer.boundCtx
	}
	b.layer.processLog(level, messages, b.layer.fields, ctx, b.metadata, b.err, groups)
}
