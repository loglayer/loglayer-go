// Package transport defines the Transport interface and BaseTransport helper
// used by all LogLayer transport implementations.
package transport

import (
	"sync/atomic"

	"go.loglayer.dev"
	"go.loglayer.dev/utils/idgen"
)

// BaseTransport provides common fields and level-filtering logic for transports.
// Concrete transports should embed *BaseTransport and implement ShipToLogger.
//
// enabled is a pointer to an atomic so the struct itself stays copyable
// (NewBaseTransport returns by value into the embedding transport's
// constructor); the pointer is allocated once and shared.
type BaseTransport struct {
	id      string
	level   loglayer.LogLevel
	enabled *atomic.Bool
}

// BaseConfig holds the common configuration fields shared by all transports.
type BaseConfig struct {
	// ID uniquely identifies this transport. Optional: when empty,
	// NewBaseTransport assigns an auto-generated identifier. Supply your
	// own when you intend to call RemoveTransport / GetLoggerInstance by
	// ID later.
	ID string

	// Disabled suppresses this transport from accepting log entries when true.
	// Defaults to false (transport active). Equivalent to calling
	// SetEnabled(false) after construction.
	Disabled bool

	// Level sets the minimum log level this transport will process. Defaults to LogLevelDebug.
	Level loglayer.LogLevel
}

// NewBaseTransport creates a BaseTransport from a BaseConfig. An empty
// cfg.ID is replaced with an auto-generated identifier.
func NewBaseTransport(cfg BaseConfig) BaseTransport {
	level := loglayer.LogLevelDebug
	if cfg.Level != 0 {
		level = cfg.Level
	}
	id := cfg.ID
	if id == "" {
		id = idgen.Random(idgen.TransportPrefix)
	}
	enabled := &atomic.Bool{}
	enabled.Store(!cfg.Disabled)
	return BaseTransport{
		id:      id,
		level:   level,
		enabled: enabled,
	}
}

// ID returns the transport's unique identifier.
func (b *BaseTransport) ID() string { return b.id }

// IsEnabled returns whether the transport is currently enabled. Safe to call
// concurrently with SetEnabled.
func (b *BaseTransport) IsEnabled() bool { return b.enabled.Load() }

// SetEnabled enables or disables the transport. Safe to call concurrently
// with emission and IsEnabled.
func (b *BaseTransport) SetEnabled(v bool) { b.enabled.Store(v) }

// MinLevel returns the minimum log level this transport will process.
func (b *BaseTransport) MinLevel() loglayer.LogLevel { return b.level }

// ShouldProcess returns true if the transport is enabled and the log level
// meets the minimum level threshold.
func (b *BaseTransport) ShouldProcess(level loglayer.LogLevel) bool {
	return b.enabled.Load() && level >= b.level
}
