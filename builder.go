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
}

func newLogBuilder(l *LogLayer) *LogBuilder {
	return &LogBuilder{layer: l}
}

// WithMetadata attaches metadata to the log entry. Accepts any value: a struct,
// a map, or any other type. Serialization is handled by the transport.
// Calling this multiple times replaces the previous value.
func (b *LogBuilder) WithMetadata(v any) *LogBuilder {
	b.metadata = v
	return b
}

// WithError attaches an error to the log entry.
func (b *LogBuilder) WithError(err error) *LogBuilder {
	b.err = err
	return b
}

// WithCtx attaches a context.Context to the log entry. Transports receive it
// via TransportParams.Ctx and can extract trace IDs, span context, or anything
// else carried in the context.
//
// Unlike WithFields (which mutates the logger's persistent key/value bag),
// WithCtx is per-call only. Passing nil is a no-op.
func (b *LogBuilder) WithCtx(ctx context.Context) *LogBuilder {
	b.ctx = ctx
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
	b.layer.processLog(level, messages, b.layer.fields, b.ctx, b.metadata, b.err)
}
