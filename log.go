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

// WithContext returns a derived logger that automatically attaches the given
// context.Context to every emission. Transports receive it via
// TransportParams.Ctx; plugins receive it on dispatch-time hook params.
//
// Per-call (*LogBuilder).WithContext still overrides for one emission.
//
// The receiver is unchanged (returns a new logger; assign the result).
// Passing nil returns a clone with no bound context, which clears any
// context the receiver had previously bound.
func (l *LogLayer) WithContext(ctx context.Context) *LogLayer {
	child := l.Child()
	child.boundCtx = ctx
	return child
}

// Trace logs at the trace level. Trace sits below Debug; use it for
// extremely fine-grained diagnostic output that you'd typically want
// disabled in production.
func (l *LogLayer) Trace(messages ...any) {
	if !l.levels.isEnabled(LogLevelTrace) {
		return
	}
	var src *Source
	if l.config.Source.Enabled {
		src = captureSource(1)
	}
	l.formatLog(LogLevelTrace, messages, nil, nil, nil, src, l.loadPlugins())
}

// Info logs at the info level.
func (l *LogLayer) Info(messages ...any) {
	if !l.levels.isEnabled(LogLevelInfo) {
		return
	}
	var src *Source
	if l.config.Source.Enabled {
		src = captureSource(1)
	}
	l.formatLog(LogLevelInfo, messages, nil, nil, nil, src, l.loadPlugins())
}

// Warn logs at the warn level.
func (l *LogLayer) Warn(messages ...any) {
	if !l.levels.isEnabled(LogLevelWarn) {
		return
	}
	var src *Source
	if l.config.Source.Enabled {
		src = captureSource(1)
	}
	l.formatLog(LogLevelWarn, messages, nil, nil, nil, src, l.loadPlugins())
}

// Error logs at the error level.
func (l *LogLayer) Error(messages ...any) {
	if !l.levels.isEnabled(LogLevelError) {
		return
	}
	var src *Source
	if l.config.Source.Enabled {
		src = captureSource(1)
	}
	l.formatLog(LogLevelError, messages, nil, nil, nil, src, l.loadPlugins())
}

// Debug logs at the debug level.
func (l *LogLayer) Debug(messages ...any) {
	if !l.levels.isEnabled(LogLevelDebug) {
		return
	}
	var src *Source
	if l.config.Source.Enabled {
		src = captureSource(1)
	}
	l.formatLog(LogLevelDebug, messages, nil, nil, nil, src, l.loadPlugins())
}

// Fatal logs at the fatal level. Calls os.Exit(1) after dispatch unless
// Config.DisableFatalExit is set.
func (l *LogLayer) Fatal(messages ...any) {
	if !l.levels.isEnabled(LogLevelFatal) {
		return
	}
	var src *Source
	if l.config.Source.Enabled {
		src = captureSource(1)
	}
	l.formatLog(LogLevelFatal, messages, nil, nil, nil, src, l.loadPlugins())
}

// Panic logs at the panic level then panics with the joined message string.
// Unlike Fatal, the panic is recoverable, so async transports are not
// pre-flushed (closing them would break callers that recover and keep
// emitting). To suppress the panic in tests, recover in the calling
// goroutine.
func (l *LogLayer) Panic(messages ...any) {
	if !l.levels.isEnabled(LogLevelPanic) {
		return
	}
	var src *Source
	if l.config.Source.Enabled {
		src = captureSource(1)
	}
	l.formatLog(LogLevelPanic, messages, nil, nil, nil, src, l.loadPlugins())
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

	var src *Source
	if l.config.Source.Enabled {
		src = captureSource(1)
	}
	l.formatLog(level, messages, nil, nil, err, src, l.loadPlugins())
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
	plugins := l.loadPlugins()
	v = plugins.runOnMetadataCalled(v)
	if !l.levels.isEnabled(level) || l.config.MuteMetadata || v == nil {
		return
	}
	var src *Source
	if l.config.Source.Enabled {
		src = captureSource(1)
	}
	l.formatLog(level, nil, nil, v, nil, src, plugins)
}

// Raw dispatches a fully specified log entry, bypassing the builder API.
// All normal assembly and transport dispatch still applies.
//
// entry.Source takes precedence over runtime capture: if it's non-nil it's
// passed through as-is (the slog handler uses this to forward source from
// slog.Record.PC). Otherwise, when Config.Source.Enabled is true, source is
// captured at the Raw call site.
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
	src := entry.Source
	if src == nil && l.config.Source.Enabled {
		src = captureSource(1)
	}
	l.processLog(entry.LogLevel, entry.Messages, fields, ctx, entry.Metadata, entry.Err, src, groups, l.loadPlugins())
}
