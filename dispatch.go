package loglayer

import (
	"context"
	"os"
)

// formatLog applies the prefix to messages then hands the entry to processLog
// using the logger's persistent fields.
func (l *LogLayer) formatLog(level LogLevel, messages []any, goCtx context.Context, metadata any, err error) {
	applyPrefix(l.config.Prefix, messages)
	l.processLog(level, messages, l.fields, goCtx, metadata, err)
}

// processLog assembles Data from fields + error, builds TransportParams, and
// dispatches to every enabled transport. After dispatch, calls os.Exit(1) for
// fatal-level entries unless Config.DisableFatalExit is set.
//
// goCtx is the optional per-call Go context.Context attached via WithCtx.
func (l *LogLayer) processLog(level LogLevel, messages []any, fields Fields, goCtx context.Context, metadata any, err error) {
	cfg := &l.config
	includeFields := !l.muteFields.Load() && len(fields) > 0

	var d Data
	if includeFields || err != nil {
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
		var serialized any
		if cfg.ErrorSerializer != nil {
			serialized = cfg.ErrorSerializer(err)
		} else {
			serialized = map[string]any{"message": err.Error()}
		}
		d[cfg.ErrorFieldName] = serialized
	}

	var rawMetadata any
	if !l.muteMetadata.Load() {
		rawMetadata = metadata
	}

	params := TransportParams{
		LogLevel: level,
		Messages: messages,
		Data:     d,
		HasData:  d != nil,
		Metadata: rawMetadata,
		Err:      err,
		Fields:   fields,
		Ctx:      goCtx,
	}

	for _, t := range l.loadTransports().list {
		if t.IsEnabled() {
			t.SendToLogger(params)
		}
	}

	if level == LogLevelFatal && !cfg.DisableFatalExit {
		os.Exit(1)
	}
}
