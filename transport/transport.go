// Package transport defines the Transport interface and BaseTransport helper
// used by all LogLayer transport implementations.
package transport

import "go.loglayer.dev/loglayer"

// BaseTransport provides common fields and level-filtering logic for transports.
// Concrete transports should embed *BaseTransport and implement ShipToLogger.
type BaseTransport struct {
	id      string
	enabled bool
	level   loglayer.LogLevel
}

// BaseConfig holds the common configuration fields shared by all transports.
type BaseConfig struct {
	// ID uniquely identifies this transport. Required for transport management.
	ID string

	// Enabled controls whether the transport accepts log entries. Defaults to true.
	Enabled *bool

	// Level sets the minimum log level this transport will process. Defaults to LogLevelTrace.
	Level loglayer.LogLevel
}

// NewBaseTransport creates a BaseTransport from a BaseConfig.
func NewBaseTransport(cfg BaseConfig) BaseTransport {
	enabled := true
	if cfg.Enabled != nil {
		enabled = *cfg.Enabled
	}
	level := loglayer.LogLevelTrace
	if cfg.Level != 0 {
		level = cfg.Level
	}
	return BaseTransport{
		id:      cfg.ID,
		enabled: enabled,
		level:   level,
	}
}

// ID returns the transport's unique identifier.
func (b *BaseTransport) ID() string { return b.id }

// IsEnabled returns whether the transport is currently enabled.
func (b *BaseTransport) IsEnabled() bool { return b.enabled }

// SetEnabled enables or disables the transport.
func (b *BaseTransport) SetEnabled(v bool) { b.enabled = v }

// MinLevel returns the minimum log level this transport will process.
func (b *BaseTransport) MinLevel() loglayer.LogLevel { return b.level }

// ShouldProcess returns true if the transport is enabled and the log level
// meets the minimum level threshold.
func (b *BaseTransport) ShouldProcess(level loglayer.LogLevel) bool {
	return b.enabled && level >= b.level
}
