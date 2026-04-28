package loglayer

import "sync/atomic"

// LogLevel represents the severity of a log entry.
// Higher numeric values indicate higher severity.
type LogLevel int

const (
	LogLevelDebug LogLevel = 10
	LogLevelInfo  LogLevel = 20
	LogLevelWarn  LogLevel = 30
	LogLevelError LogLevel = 40
	LogLevelFatal LogLevel = 50
)

const (
	// numLevels is the count of distinct levels (Debug through Fatal).
	numLevels = 5
	// levelStep is the numeric spacing between adjacent levels (Debug=10,
	// Info=20, ..., Fatal=50). levelIndex relies on this contract.
	levelStep = 10
	// allLevelsBits is bits 0..numLevels-1 set: every level enabled.
	allLevelsBits uint32 = 1<<numLevels - 1
	// masterEnabledBit lives just above the per-level bits and represents the
	// global on/off toggle.
	masterEnabledBit uint32 = 1 << numLevels
	// initialState is the default: all levels enabled, master on.
	initialState = allLevelsBits | masterEnabledBit
)

// levelIndex maps a LogLevel to its slot in the levelState bitmap.
// Returns -1 for unknown levels.
func levelIndex(l LogLevel) int {
	v := int(l)
	if v < levelStep || v > numLevels*levelStep || v%levelStep != 0 {
		return -1
	}
	return v/levelStep - 1
}

// String returns the lowercase string name of a log level.
func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "debug"
	case LogLevelInfo:
		return "info"
	case LogLevelWarn:
		return "warn"
	case LogLevelError:
		return "error"
	case LogLevelFatal:
		return "fatal"
	default:
		return "unknown"
	}
}

// ParseLogLevel converts a string level name to a LogLevel.
// Returns LogLevelInfo and false if the name is not recognized.
func ParseLogLevel(s string) (LogLevel, bool) {
	switch s {
	case "debug":
		return LogLevelDebug, true
	case "info":
		return LogLevelInfo, true
	case "warn":
		return LogLevelWarn, true
	case "error":
		return LogLevelError, true
	case "fatal":
		return LogLevelFatal, true
	default:
		return LogLevelInfo, false
	}
}

// levelState tracks which levels are enabled plus the master logging switch.
//
// Stored as a single atomic.Uint32 bitmap (bits 0..4 = per-level enabled, bit
// 5 = master) so emission and runtime reconfiguration (e.g. SIGUSR1-driven
// level toggles, admin endpoints flipping debug logging) compose without
// locks. Mirrors zap.AtomicLevel.
type levelState struct {
	bits atomic.Uint32
}

func newLevelState() *levelState {
	s := &levelState{}
	s.bits.Store(initialState)
	return s
}

// clone returns an independent copy of s holding a snapshot of the current bits.
func (s *levelState) clone() *levelState {
	c := &levelState{}
	c.bits.Store(s.bits.Load())
	return c
}

func (s *levelState) isEnabled(l LogLevel) bool {
	cur := s.bits.Load()
	if cur&masterEnabledBit == 0 {
		return false
	}
	idx := levelIndex(l)
	if idx < 0 {
		return false
	}
	return cur&(1<<idx) != 0
}

// setLevel enables all levels >= l and disables all levels below l.
// No-op for unknown levels. Preserves the master enabled bit.
func (s *levelState) setLevel(l LogLevel) {
	target := levelIndex(l)
	if target < 0 {
		return
	}
	var levelBits uint32
	for i := 0; i < numLevels; i++ {
		if i >= target {
			levelBits |= 1 << i
		}
	}
	s.update(func(old uint32) uint32 {
		return (old & masterEnabledBit) | levelBits
	})
}

// setEnabled toggles a single level. No-op for unknown levels.
func (s *levelState) setEnabled(l LogLevel, on bool) {
	idx := levelIndex(l)
	if idx < 0 {
		return
	}
	bit := uint32(1 << idx)
	s.update(func(old uint32) uint32 {
		if on {
			return old | bit
		}
		return old &^ bit
	})
}

// setMaster toggles the master logging switch.
func (s *levelState) setMaster(on bool) {
	s.update(func(old uint32) uint32 {
		if on {
			return old | masterEnabledBit
		}
		return old &^ masterEnabledBit
	})
}

// update applies fn to the current bits and CAS-stores the result, retrying
// on contention. Lock-free; safe to call concurrently with anything that
// reads s.bits.
func (s *levelState) update(fn func(uint32) uint32) {
	for {
		old := s.bits.Load()
		next := fn(old)
		if next == old || s.bits.CompareAndSwap(old, next) {
			return
		}
	}
}
