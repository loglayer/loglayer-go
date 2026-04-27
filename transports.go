package loglayer

// AddTransport appends one or more transports. If a transport with the same ID
// already exists it is replaced.
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
	for _, t := range current {
		if !newIDs[t.ID()] {
			filtered = append(filtered, t)
		}
	}
	l.publishTransports(append(filtered, transports...))
	return l
}

// RemoveTransport removes the transport with the given ID.
// Returns true if found and removed, false otherwise.
//
// Safe to call concurrently with log emission.
func (l *LogLayer) RemoveTransport(id string) bool {
	l.txMu.Lock()
	defer l.txMu.Unlock()

	current := l.loadTransports()
	if _, ok := current.byID[id]; !ok {
		return false
	}
	remaining := make([]Transport, 0, len(current.list)-1)
	for _, t := range current.list {
		if t.ID() != id {
			remaining = append(remaining, t)
		}
	}
	l.publishTransports(remaining)
	return true
}

// SetTransports replaces all existing transports.
//
// Safe to call concurrently with log emission.
func (l *LogLayer) SetTransports(transports ...Transport) *LogLayer {
	l.txMu.Lock()
	defer l.txMu.Unlock()

	l.publishTransports(transports)
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
