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

// WithCtx returns a derived logger that automatically attaches the given
// context.Context to every emission. Transports receive it via
// TransportParams.Ctx; plugins receive it on dispatch-time hook params.
//
// Per-call (*LogBuilder).WithCtx still overrides for one emission.
//
// The receiver is unchanged (returns a new logger; assign the result).
// Passing nil returns a clone with no bound context, which clears any
// context the receiver had previously bound.
func (l *LogLayer) WithCtx(ctx context.Context) *LogLayer {
	child := l.Child()
	child.boundCtx = ctx
	return child
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
		switch o.CopyMsg {
		case CopyMsgEnabled:
			copyMsg = true
		case CopyMsgDisabled:
			copyMsg = false
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
// OnMetadataCalled plugin hooks run here, same as WithMetadata. If a
// plugin returns nil (the documented nil-drop signal), the entire entry
// is suppressed: there's no message and no metadata, so there's nothing
// to log. Plugin authors should be aware that returning nil from
// OnMetadataCalled silences MetadataOnly callers entirely. Same applies
// when MuteMetadata is set on the logger.
func (l *LogLayer) MetadataOnly(v any, opts ...MetadataOnlyOpts) {
	level := LogLevelInfo
	if len(opts) > 0 && opts[0].LogLevel != 0 {
		level = opts[0].LogLevel
	}
	v = l.loadPlugins().runOnMetadataCalled(v)
	if !l.levels.isEnabled(level) || l.config.MuteMetadata || v == nil {
		return
	}
	l.formatLog(level, nil, nil, v, nil)
}

// Raw dispatches a fully specified log entry, bypassing the builder API.
// All normal assembly and transport dispatch still applies.
func (l *LogLayer) Raw(entry RawLogEntry) {
	if !l.levels.isEnabled(entry.LogLevel) {
		return
	}
	applyPrefix(l.prefix, entry.Messages)
	fields := entry.Fields
	if fields == nil {
		fields = l.fields
	}
	groups := entry.Groups
	if groups == nil {
		groups = l.assignedGroups
	}
	ctx := entry.Ctx
	if ctx == nil {
		ctx = l.boundCtx
	}
	l.processLog(entry.LogLevel, entry.Messages, fields, ctx, entry.Metadata, entry.Err, groups, l.loadPlugins())
}
