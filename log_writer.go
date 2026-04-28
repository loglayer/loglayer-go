package loglayer

import (
	"bytes"
	"io"
	"log"
)

// Writer returns an io.Writer that emits one log entry per Write call at
// the given level. Each Write becomes a single dispatch through the
// loglayer pipeline (plugins, fan-out, group routing, level state).
//
// Trailing newlines are stripped (callers like the stdlib log package
// always append "\n", and the rendered entry adds its own delimiter).
// Empty writes are suppressed so a flush of an already-empty buffer
// does not produce a blank line.
//
// Use it to plumb code that writes to an io.Writer (third-party
// libraries' loggers, raw byte streams, anything that calls Write)
// through your loglayer pipeline:
//
//	w := log.Writer(loglayer.LogLevelInfo)
//	fmt.Fprintln(w, "hello")
func (l *LogLayer) Writer(level LogLevel) io.Writer {
	return &logWriter{layer: l, level: level}
}

// NewLogLogger returns a *log.Logger that emits through this LogLayer at
// the given level. Use it for stdlib-log-shaped consumers that won't
// accept an arbitrary writer:
//
//	srv := &http.Server{
//	    ErrorLog: log.NewLogLogger(loglayer.LogLevelError),
//	}
//
// gorm, database/sql tracing, and most third-party libraries that take a
// *log.Logger plug in the same way. The returned logger has empty prefix
// and zero flags so the rendered entry doesn't get a duplicate timestamp
// or level prefix from the stdlib side; loglayer adds those itself.
//
// Mirrors slog.NewLogLogger.
func (l *LogLayer) NewLogLogger(level LogLevel) *log.Logger {
	return log.New(l.Writer(level), "", 0)
}

type logWriter struct {
	layer *LogLayer
	level LogLevel
}

func (w *logWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	trimmed := bytes.TrimRight(p, "\n")
	if len(trimmed) == 0 {
		return len(p), nil
	}
	w.layer.Raw(RawLogEntry{
		LogLevel: w.level,
		Messages: []any{string(trimmed)},
	})
	return len(p), nil
}
