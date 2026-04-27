package loglayer

import "context"

// WithMetadata returns a LogBuilder with the given metadata attached.
// Accepts any value: a struct, a map, or any other type. Serialization is
// handled by the transport.
func (l *LogLayer) WithMetadata(v any) *LogBuilder {
	return newLogBuilder(l).WithMetadata(v)
}

// WithError returns a LogBuilder with the given error attached.
func (l *LogLayer) WithError(err error) *LogBuilder {
	return newLogBuilder(l).WithError(err)
}

// WithCtx returns a LogBuilder with the given context.Context attached.
// Transports receive it via TransportParams.Ctx. The context is per-call;
// it does not persist on the logger.
func (l *LogLayer) WithCtx(ctx context.Context) *LogBuilder {
	return newLogBuilder(l).WithCtx(ctx)
}

// Info logs at the info level.
func (l *LogLayer) Info(messages ...any) {
	if !l.levels.isEnabled(LogLevelInfo) {
		return
	}
	l.formatLog(LogLevelInfo, messages, nil, nil, nil)
}

// Warn logs at the warn level.
func (l *LogLayer) Warn(messages ...any) {
	if !l.levels.isEnabled(LogLevelWarn) {
		return
	}
	l.formatLog(LogLevelWarn, messages, nil, nil, nil)
}

// Error logs at the error level.
func (l *LogLayer) Error(messages ...any) {
	if !l.levels.isEnabled(LogLevelError) {
		return
	}
	l.formatLog(LogLevelError, messages, nil, nil, nil)
}

// Debug logs at the debug level.
func (l *LogLayer) Debug(messages ...any) {
	if !l.levels.isEnabled(LogLevelDebug) {
		return
	}
	l.formatLog(LogLevelDebug, messages, nil, nil, nil)
}

// Trace logs at the trace level.
func (l *LogLayer) Trace(messages ...any) {
	if !l.levels.isEnabled(LogLevelTrace) {
		return
	}
	l.formatLog(LogLevelTrace, messages, nil, nil, nil)
}

// Fatal logs at the fatal level. Calls os.Exit(1) after dispatch unless
// Config.DisableFatalExit is set.
func (l *LogLayer) Fatal(messages ...any) {
	if !l.levels.isEnabled(LogLevelFatal) {
		return
	}
	l.formatLog(LogLevelFatal, messages, nil, nil, nil)
}

// ErrorOnly logs an error without a message. The log level defaults to error.
func (l *LogLayer) ErrorOnly(err error, opts ...ErrorOnlyOpts) {
	level := LogLevelError
	copyMsg := l.config.CopyMsgOnOnlyError

	if len(opts) > 0 {
		o := opts[0]
		if o.LogLevel != 0 {
			level = o.LogLevel
		}
		if o.CopyMsg != nil {
			copyMsg = *o.CopyMsg
		}
	}

	if !l.levels.isEnabled(level) {
		return
	}

	var messages []any
	if copyMsg && err != nil {
		messages = []any{err.Error()}
	}

	l.formatLog(level, messages, nil, nil, err)
}

// MetadataOnly logs metadata without a message. The log level defaults to info.
// Accepts any value: a struct, a map, or any other type.
//
// OnMetadataCalled plugin hooks run here, same as WithMetadata.
func (l *LogLayer) MetadataOnly(v any, level ...LogLevel) {
	lvl := LogLevelInfo
	if len(level) > 0 && level[0] != 0 {
		lvl = level[0]
	}
	v = l.loadPlugins().runOnMetadataCalled(v)
	if !l.levels.isEnabled(lvl) || l.config.MuteMetadata || v == nil {
		return
	}
	l.formatLog(lvl, nil, nil, v, nil)
}

// Raw dispatches a fully specified log entry, bypassing the builder API.
// All normal assembly and transport dispatch still applies.
func (l *LogLayer) Raw(entry RawLogEntry) {
	if !l.levels.isEnabled(entry.LogLevel) {
		return
	}
	applyPrefix(l.config.Prefix, entry.Messages)
	fields := entry.Fields
	if fields == nil {
		fields = l.fields
	}
	l.processLog(entry.LogLevel, entry.Messages, fields, entry.Ctx, entry.Metadata, entry.Err)
}
