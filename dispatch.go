package loglayer

import (
	"context"
	"fmt"
	"os"
)

// osExit is the process-exit hook for fatal-level entries. Overridden in
// tests to verify the pre-exit flush without terminating the test runner.
var osExit = os.Exit

// formatLog applies the prefix to messages then hands the entry to processLog
// using the logger's persistent fields. Per-call goCtx overrides the
// logger's bound ctx (when one is provided), otherwise the bound ctx is
// passed through. source carries pre-captured call-site info from the
// emission entry point (nil if Config.AddSource is off and no adapter
// supplied one).
func (l *LogLayer) formatLog(level LogLevel, messages []any, goCtx context.Context, metadata any, err error, source *Source, plugins *pluginSet) {
	applyPrefix(l.prefix, messages)
	if goCtx == nil {
		goCtx = l.boundCtx
	}
	l.processLog(level, messages, l.fields, goCtx, metadata, err, source, l.assignedGroups, plugins)
}

// processLog assembles Data from fields + error, builds TransportParams, and
// dispatches to every enabled transport. After dispatch, calls os.Exit(1) for
// fatal-level entries unless Config.DisableFatalExit is set.
//
// goCtx is the optional per-call Go context.Context attached via WithCtx.
// entryGroups is the merged set of persistent + per-call group tags for
// routing decisions (nil when no groups apply). plugins is the snapshot
// used for all hooks on this entry; callers that already snapshotted
// (builder, MetadataOnly) pass theirs to keep one entry on one snapshot.
func (l *LogLayer) processLog(level LogLevel, messages []any, fields Fields, goCtx context.Context, metadata any, err error, source *Source, entryGroups []string, plugins *pluginSet) {
	cfg := &l.config
	includeFields := !l.muteFields.Load() && len(fields) > 0
	includeSource := source != nil

	var d Data
	if includeFields || err != nil || includeSource {
		size := 0
		if includeFields {
			if cfg.FieldsKey == "" {
				size = len(fields)
			} else {
				size = 1
			}
		}
		if err != nil {
			size++
		}
		if includeSource {
			size++
		}
		d = make(Data, size)
	}

	if includeFields {
		if cfg.FieldsKey != "" {
			nested := make(map[string]any, len(fields))
			for k, v := range fields {
				nested[k] = v
			}
			d[cfg.FieldsKey] = nested
		} else {
			for k, v := range fields {
				d[k] = v
			}
		}
	}

	if err != nil {
		if cfg.ErrorSerializer != nil {
			// A custom serializer returning nil drops the err key
			// entirely (matches plugin-hook nil-drop semantics).
			// Returning an empty map still adds the key with an
			// empty object.
			if m := cfg.ErrorSerializer(err); m != nil {
				d[cfg.ErrorFieldName] = m
			}
		} else {
			d[cfg.ErrorFieldName] = map[string]any{"message": err.Error()}
		}
	}

	if includeSource {
		d[cfg.SourceFieldName] = source
	}

	var rawMetadata any
	if !l.muteMetadata.Load() {
		rawMetadata = metadata
	}

	if plugins.anyDispatchHook {
		d = plugins.runOnBeforeDataOut(BeforeDataOutParams{
			LogLevel: level,
			Data:     d,
			Fields:   fields,
			Metadata: rawMetadata,
			Err:      err,
			Ctx:      goCtx,
		})
		messages = plugins.runOnBeforeMessageOut(BeforeMessageOutParams{
			LogLevel: level,
			Messages: messages,
			Ctx:      goCtx,
		})
		level = plugins.runTransformLogLevel(TransformLogLevelParams{
			LogLevel: level,
			Data:     d,
			Messages: messages,
			Fields:   fields,
			Metadata: rawMetadata,
			Err:      err,
			Ctx:      goCtx,
		})
	}

	params := TransportParams{
		LogLevel: level,
		Messages: messages,
		Data:     d,
		Metadata: rawMetadata,
		Err:      err,
		Fields:   fields,
		Ctx:      goCtx,
	}

	hasShouldSend := plugins.hasSendGate
	groupsConfig := l.loadGroups()
	needsRouting := groupsConfig.hasGroups
	for _, t := range l.loadTransports().list {
		if !t.IsEnabled() {
			continue
		}
		if needsRouting && !groupsConfig.shouldRoute(t.ID(), level, entryGroups) {
			continue
		}
		if hasShouldSend && !plugins.runShouldSend(ShouldSendParams{
			TransportID: t.ID(),
			LogLevel:    level,
			Messages:    messages,
			Data:        d,
			Fields:      fields,
			Metadata:    rawMetadata,
			Err:         err,
			Ctx:         goCtx,
		}) {
			continue
		}
		t.SendToLogger(params)
	}

	if level == LogLevelFatal && !cfg.DisableFatalExit {
		// Async transports (HTTP/Datadog) only enqueue in SendToLogger;
		// without flushing them here the Fatal entry would be lost when
		// os.Exit terminates the worker goroutines mid-batch. Bounded
		// by Config.TransportCloseTimeout so a wedged endpoint can't
		// hang the process indefinitely.
		flushTransports(l.loadTransports().list, cfg.TransportCloseTimeout)
		osExit(1)
	}

	if level == LogLevelPanic {
		// Panic is recoverable, so we don't pre-flush async transports
		// (closing them would break callers that recover and keep
		// emitting). The panic value is the joined message string,
		// matching zerolog / zap / logrus convention so a recover()
		// gets back something useful.
		panic(fmt.Sprint(messages...))
	}
}
