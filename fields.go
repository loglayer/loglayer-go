package loglayer

// WithFields returns a new logger with the given key/value pairs merged into
// the persistent fields bag. The receiver is unchanged.
//
// This matches the convention used by zerolog, zap, slog, and logrus: the
// derive operation produces a fresh logger so concurrent goroutines (HTTP
// handlers, workers) can each carry their own per-request fields without
// racing on shared state. Assign the result:
//
//	log = log.WithFields(loglayer.Fields{"requestId": "abc"})
//
// Discarding the return value is a no-op. The compiler does not catch this.
func (l *LogLayer) WithFields(f Fields) *LogLayer {
	// Plugin OnFieldsCalled hooks run before the fields are merged. A hook
	// returning nil drops the WithFields call entirely; the receiver's
	// existing fields are preserved either way (we still return a fresh
	// child so call sites that always reassign get the expected behavior).
	f = l.loadPlugins().runOnFieldsCalled(f)
	out := l.Child()
	for k, v := range f {
		out.fields[k] = v
	}
	return out
}

// ClearFields returns a new logger with the given keys removed from the
// persistent fields bag. With no arguments, all fields are cleared. The
// receiver is unchanged. Paired with WithFields; assign the result.
func (l *LogLayer) ClearFields(keys ...string) *LogLayer {
	out := l.Child()
	if len(keys) == 0 {
		out.fields = make(Fields)
	} else {
		for _, k := range keys {
			delete(out.fields, k)
		}
	}
	return out
}

// GetFields returns a shallow copy of the current persistent fields.
func (l *LogLayer) GetFields() Fields {
	out := make(Fields, len(l.fields))
	for k, v := range l.fields {
		out[k] = v
	}
	return out
}

// MuteFields disables persistent fields from appearing in log output.
//
// Safe to call concurrently with log emission: backed by atomic.Bool.
func (l *LogLayer) MuteFields() *LogLayer {
	l.muteFields.Store(true)
	return l
}

// UnmuteFields re-enables persistent fields in log output.
//
// Safe to call concurrently with log emission: backed by atomic.Bool.
func (l *LogLayer) UnmuteFields() *LogLayer {
	l.muteFields.Store(false)
	return l
}

// MuteMetadata disables metadata from appearing in log output.
//
// Safe to call concurrently with log emission: backed by atomic.Bool.
func (l *LogLayer) MuteMetadata() *LogLayer {
	l.muteMetadata.Store(true)
	return l
}

// UnmuteMetadata re-enables metadata in log output.
//
// Safe to call concurrently with log emission: backed by atomic.Bool.
func (l *LogLayer) UnmuteMetadata() *LogLayer {
	l.muteMetadata.Store(false)
	return l
}
