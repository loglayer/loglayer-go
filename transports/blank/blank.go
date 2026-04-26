// Package blank provides a Transport that delegates SendToLogger to a
// user-supplied function. It is useful for prototyping a new transport
// inline without creating a full package, for one-off integrations
// (forward to a metrics system, a queue, etc.), and for tests that need
// to inspect raw TransportParams without writing a TestTransport setup.
//
// If you find yourself reusing the same blank.Config in multiple places,
// promote it to its own transport package; see
// /transports/creating-transports.md for the full template.
package blank

import (
	"go.loglayer.dev/loglayer"
	"go.loglayer.dev/loglayer/transport"
)

// Config holds configuration for the blank transport.
type Config struct {
	transport.BaseConfig

	// ShipToLogger is invoked for every entry that passes the transport's
	// level filter. If nil, entries are silently discarded (which makes a
	// nil-ShipToLogger Transport a useful no-op for tests).
	ShipToLogger func(params loglayer.TransportParams)
}

// Transport delegates dispatch to the configured ShipToLogger function.
type Transport struct {
	transport.BaseTransport
	fn func(loglayer.TransportParams)
}

// New creates a blank Transport from the given Config.
func New(cfg Config) *Transport {
	return &Transport{
		BaseTransport: transport.NewBaseTransport(cfg.BaseConfig),
		fn:            cfg.ShipToLogger,
	}
}

// GetLoggerInstance returns nil; blank has no underlying library.
func (t *Transport) GetLoggerInstance() any { return nil }

// SendToLogger implements loglayer.Transport. Calls ShipToLogger if set;
// otherwise the entry is dropped after the level filter.
func (t *Transport) SendToLogger(params loglayer.TransportParams) {
	if !t.ShouldProcess(params.LogLevel) {
		return
	}
	if t.fn != nil {
		t.fn(params)
	}
}
