package loglayer

// SetLevel enables all levels at or above level and disables those below it.
//
// Safe to call concurrently with log emission: mutates an atomic bitmap.
func (l *LogLayer) SetLevel(level LogLevel) *LogLayer {
	l.levels.setLevel(level)
	return l
}

// EnableLevel turns on a specific log level without affecting others.
// Unknown levels are silently ignored.
//
// Safe to call concurrently with log emission: mutates an atomic bitmap.
func (l *LogLayer) EnableLevel(level LogLevel) *LogLayer {
	l.levels.setEnabled(level, true)
	return l
}

// DisableLevel turns off a specific log level without affecting others.
// Unknown levels are silently ignored.
//
// Safe to call concurrently with log emission: mutates an atomic bitmap.
func (l *LogLayer) DisableLevel(level LogLevel) *LogLayer {
	l.levels.setEnabled(level, false)
	return l
}

// EnableLogging re-enables all logging after DisableLogging.
//
// Safe to call concurrently with log emission: mutates an atomic bitmap.
func (l *LogLayer) EnableLogging() *LogLayer {
	l.levels.setMaster(true)
	return l
}

// DisableLogging suppresses all log output regardless of individual level state.
//
// Safe to call concurrently with log emission: mutates an atomic bitmap.
func (l *LogLayer) DisableLogging() *LogLayer {
	l.levels.setMaster(false)
	return l
}

// IsLevelEnabled reports whether the given level will produce output.
func (l *LogLayer) IsLevelEnabled(level LogLevel) bool {
	return l.levels.isEnabled(level)
}
