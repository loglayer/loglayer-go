package loglayer

import (
	"fmt"
	"log/slog"
	"runtime"
)

// String returns a compact "function file:line" rendering, used by console
// and pretty transports that fall back to %v for unknown values. Empty
// fields are omitted; the result never has trailing whitespace.
func (s *Source) String() string {
	if s == nil {
		return ""
	}
	switch {
	case s.Function != "" && s.File != "" && s.Line > 0:
		return fmt.Sprintf("%s %s:%d", s.Function, s.File, s.Line)
	case s.File != "" && s.Line > 0:
		return fmt.Sprintf("%s:%d", s.File, s.Line)
	case s.Function != "":
		return s.Function
	case s.File != "":
		return s.File
	}
	return ""
}

// LogValue makes Source implement slog.LogValuer so a source attached to
// a slog logger (via the slog transport) renders as a nested
// {function, file, line} group rather than a stringified struct.
func (s *Source) LogValue() slog.Value {
	if s == nil {
		return slog.Value{}
	}
	attrs := make([]slog.Attr, 0, 3)
	if s.Function != "" {
		attrs = append(attrs, slog.String("function", s.Function))
	}
	if s.File != "" {
		attrs = append(attrs, slog.String("file", s.File))
	}
	if s.Line > 0 {
		attrs = append(attrs, slog.Int("line", s.Line))
	}
	return slog.GroupValue(attrs...)
}

// captureSource walks `skip` frames up the stack to find the caller's
// location. skip=1 means "the function that called captureSource"; skip=2
// is its caller, and so on. Used by emission entry points (Info, Warn,
// builder.Info, ...) to record the user's call site when Config.AddSource
// is enabled. Returns nil if the runtime cannot resolve the frame.
//
// Cost: roughly 100ns on amd64 for one frame (one runtime.Caller +
// FuncForPC). Paid only when AddSource is true; the dispatch path is
// untouched otherwise.
func captureSource(skip int) *Source {
	pc, file, line, ok := runtime.Caller(skip + 1)
	if !ok {
		return nil
	}
	s := &Source{File: file, Line: line}
	if fn := runtime.FuncForPC(pc); fn != nil {
		s.Function = fn.Name()
	}
	return s
}

// SourceFromPC builds a Source from a captured program counter.
// Adapters that already have a PC (slog.Record.PC, custom callers using
// runtime.Callers) can call this rather than re-walking the stack.
// Returns nil for a zero PC or an unresolvable frame.
func SourceFromPC(pc uintptr) *Source {
	if pc == 0 {
		return nil
	}
	frames := runtime.CallersFrames([]uintptr{pc})
	f, _ := frames.Next()
	if f.File == "" && f.Line == 0 && f.Function == "" {
		return nil
	}
	return &Source{
		Function: f.Function,
		File:     f.File,
		Line:     f.Line,
	}
}
