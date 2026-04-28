// Package fmtlog provides format-string convenience helpers for the
// LogLayer emission API. It exists as an opt-in sub-package so the
// core stays structured-first (matching log/slog and zap's fast core)
// while users who want printf-style logging can import what they need.
//
// The helpers are free functions that take a [*loglayer.LogLayer] and
// forward to the corresponding level method with [fmt.Sprintf]:
//
//	import "go.loglayer.dev/fmtlog"
//
//	fmtlog.Infof(log, "user %d signed in", userID)
//	fmtlog.Errorf(log, "request %s failed: %v", requestID, err)
//
// For chained metadata, error, or context, the canonical pattern is
// to format inline:
//
//	log.WithMetadata(loglayer.Metadata{"userId": id}).
//	    Info(fmt.Sprintf("session %s ended", sessionID))
//
// fmtlog deliberately avoids wrapping [*loglayer.LogBuilder]. The
// builder chain is for structured fields and per-call metadata; format
// strings on top of structured data work against log search and
// aggregation, so the package doesn't make that path easier.
package fmtlog

import (
	"fmt"

	"go.loglayer.dev"
)

// Debugf formats according to a format specifier and dispatches at
// debug level. Equivalent to log.Debug(fmt.Sprintf(format, args...)).
func Debugf(log *loglayer.LogLayer, format string, args ...any) {
	log.Debug(fmt.Sprintf(format, args...))
}

// Infof formats according to a format specifier and dispatches at info level.
func Infof(log *loglayer.LogLayer, format string, args ...any) {
	log.Info(fmt.Sprintf(format, args...))
}

// Warnf formats according to a format specifier and dispatches at warn level.
func Warnf(log *loglayer.LogLayer, format string, args ...any) {
	log.Warn(fmt.Sprintf(format, args...))
}

// Errorf formats according to a format specifier and dispatches at error level.
func Errorf(log *loglayer.LogLayer, format string, args ...any) {
	log.Error(fmt.Sprintf(format, args...))
}

// Fatalf formats according to a format specifier and dispatches at
// fatal level. The framework's Fatal contract applies: dispatched
// before os.Exit(1), unless Config.DisableFatalExit is set.
func Fatalf(log *loglayer.LogLayer, format string, args ...any) {
	log.Fatal(fmt.Sprintf(format, args...))
}
