package loglayer

import "io"

// closeIfCloser closes t if it implements io.Closer. Drains async-transport
// workers (HTTP/Datadog) when a transport is removed or replaced so they
// aren't orphaned.
func closeIfCloser(t Transport) {
	if c, ok := t.(io.Closer); ok {
		_ = c.Close()
	}
}

// AddTransport appends one or more transports. If a transport with the same ID
// already exists it is closed (if it implements io.Closer) and replaced.
//
// Safe to call concurrently with log emission: the new transport set is
// published atomically. Concurrent mutators on the same logger serialize via
// an internal mutex.
func (l *LogLayer) AddTransport(transports ...Transport) *LogLayer {
	if len(transports) == 0 {
		return l
	}
	l.txMu.Lock()
	defer l.txMu.Unlock()

	current := l.loadTransports().list
	newIDs := make(map[string]bool, len(transports))
	for _, t := range transports {
		newIDs[t.ID()] = true
	}
	filtered := make([]Transport, 0, len(current)+len(transports))
	var displaced []Transport
	for _, t := range current {
		if newIDs[t.ID()] {
			displaced = append(displaced, t)
			continue
		}
		filtered = append(filtered, t)
	}
	l.publishTransports(append(filtered, transports...))
	for _, t := range displaced {
		closeIfCloser(t)
	}
	return l
}

// RemoveTransport removes the transport with the given ID. The removed
// transport is closed if it implements io.Closer (HTTP/Datadog drain
// pending entries before returning).
// Returns true if found and removed, false otherwise.
//
// Safe to call concurrently with log emission.
func (l *LogLayer) RemoveTransport(id string) bool {
	l.txMu.Lock()
	defer l.txMu.Unlock()

	current := l.loadTransports()
	removed, ok := current.byID[id]
	if !ok {
		return false
	}
	remaining := make([]Transport, 0, len(current.list)-1)
	for _, t := range current.list {
		if t.ID() != id {
			remaining = append(remaining, t)
		}
	}
	l.publishTransports(remaining)
	closeIfCloser(removed)
	return true
}

// SetTransports replaces all existing transports. Any previous transport
// not present in the new set (matched by ID) is closed if it implements
// io.Closer.
//
// Safe to call concurrently with log emission.
func (l *LogLayer) SetTransports(transports ...Transport) *LogLayer {
	l.txMu.Lock()
	defer l.txMu.Unlock()

	current := l.loadTransports().list
	keep := make(map[string]bool, len(transports))
	for _, t := range transports {
		keep[t.ID()] = true
	}
	l.publishTransports(transports)
	for _, t := range current {
		if !keep[t.ID()] {
			closeIfCloser(t)
		}
	}
	return l
}

// GetLoggerInstance returns the underlying logger instance for the transport
// with the given ID, or nil if not found.
func (l *LogLayer) GetLoggerInstance(id string) any {
	if t, ok := l.loadTransports().byID[id]; ok {
		return t.GetLoggerInstance()
	}
	return nil
}
