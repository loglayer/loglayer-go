// Package lltest provides a TestTransport and TestLoggingLibrary for use
// in the main module's own tests. It's a private copy of transports/testing
// because the public package lives in its own Go module, and main importing
// it would create a require cycle.
//
// Keep this file byte-for-byte equivalent to transports/testing/testing.go
// (modulo the package declaration). Any change here must be mirrored there
// and vice versa.
package lltest

import (
	"context"
	"sync"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
)

// LogLine is a single captured log entry. Fields are exposed directly so tests
// can assert on each piece independently rather than parsing a flattened arg list.
type LogLine struct {
	Level    loglayer.LogLevel
	Messages []any
	// Data holds the assembled fields + error map. Nil when neither were set.
	Data loglayer.Data
	// Metadata is the raw value passed to WithMetadata. Nil if not set.
	Metadata any
	// Ctx is the per-call context.Context attached via WithContext. Nil if not set.
	Ctx context.Context
}

// TestLoggingLibrary captures log lines for assertion in tests.
// All methods are safe for concurrent use.
type TestLoggingLibrary struct {
	mu    sync.Mutex
	lines []LogLine
}

// Log records a log line.
func (lib *TestLoggingLibrary) Log(line LogLine) {
	lib.mu.Lock()
	defer lib.mu.Unlock()
	lib.lines = append(lib.lines, line)
}

// Lines returns a copy of all captured lines.
func (lib *TestLoggingLibrary) Lines() []LogLine {
	lib.mu.Lock()
	defer lib.mu.Unlock()
	out := make([]LogLine, len(lib.lines))
	copy(out, lib.lines)
	return out
}

// GetLastLine returns the most recently captured line without removing it,
// or nil if no lines have been captured.
func (lib *TestLoggingLibrary) GetLastLine() *LogLine {
	lib.mu.Lock()
	defer lib.mu.Unlock()
	if len(lib.lines) == 0 {
		return nil
	}
	l := lib.lines[len(lib.lines)-1]
	return &l
}

// PopLine removes and returns the most recently captured line,
// or nil if no lines have been captured.
func (lib *TestLoggingLibrary) PopLine() *LogLine {
	lib.mu.Lock()
	defer lib.mu.Unlock()
	if len(lib.lines) == 0 {
		return nil
	}
	last := len(lib.lines) - 1
	l := lib.lines[last]
	// Zero the popped slot before reslicing so the slice header doesn't
	// retain references to the popped LogLine's Data/Metadata maps.
	lib.lines[last] = LogLine{}
	lib.lines = lib.lines[:last]
	return &l
}

// ClearLines removes all captured lines.
func (lib *TestLoggingLibrary) ClearLines() {
	lib.mu.Lock()
	defer lib.mu.Unlock()
	clear(lib.lines)
	lib.lines = lib.lines[:0]
}

// Len returns the number of captured lines.
func (lib *TestLoggingLibrary) Len() int {
	lib.mu.Lock()
	defer lib.mu.Unlock()
	return len(lib.lines)
}

// TestTransport is a transport that records all log entries into a TestLoggingLibrary.
type TestTransport struct {
	transport.BaseTransport
	Library *TestLoggingLibrary
}

// Config holds configuration for TestTransport.
type Config struct {
	transport.BaseConfig
	// Library is the TestLoggingLibrary to write into.
	// If nil, a new one is created automatically.
	Library *TestLoggingLibrary
}

// New creates a TestTransport. If cfg.Library is nil a fresh one is allocated.
func New(cfg Config) *TestTransport {
	lib := cfg.Library
	if lib == nil {
		lib = &TestLoggingLibrary{}
	}
	return &TestTransport{
		BaseTransport: transport.NewBaseTransport(cfg.BaseConfig),
		Library:       lib,
	}
}

// GetLoggerInstance returns the underlying TestLoggingLibrary.
func (t *TestTransport) GetLoggerInstance() any { return t.Library }

// SendToLogger implements loglayer.Transport.
func (t *TestTransport) SendToLogger(params loglayer.TransportParams) {
	if !t.ShouldProcess(params.LogLevel) {
		return
	}
	messages := make([]any, len(params.Messages))
	copy(messages, params.Messages)
	t.Library.Log(LogLine{
		Level:    params.LogLevel,
		Messages: messages,
		Data:     params.Data,
		Metadata: params.Metadata,
		Ctx:      params.Ctx,
	})
}
