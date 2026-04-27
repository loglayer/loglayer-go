package loglayer

import "errors"

// ErrNoTransport is returned by Build (and panicked by New) when a Config
// is constructed with neither Transport nor Transports set.
var ErrNoTransport = errors.New("loglayer: at least one transport must be provided")

// ErrTransportAndTransports is returned by Build (and panicked by New) when
// a Config sets both Transport and Transports. Use one or the other to
// avoid silently dropping entries.
var ErrTransportAndTransports = errors.New("loglayer: set Transport or Transports, not both")

// ErrPluginNoID is returned by Build (and panicked by New) when a Config
// includes a Plugin with an empty ID. Plugin IDs are required for
// AddPlugin / RemovePlugin / GetPlugin semantics.
var ErrPluginNoID = errors.New("loglayer: Plugin.ID is required")

// ErrUngroupedTransportsWithoutMode is returned by Build (and panicked by
// New) when Config.UngroupedRouting.Transports is non-empty but
// UngroupedRouting.Mode is left at its zero value (UngroupedToAll).
// Either set Mode to UngroupedToTransports to use the allowlist, or
// clear Transports.
var ErrUngroupedTransportsWithoutMode = errors.New("loglayer: UngroupedRouting.Transports set without UngroupedToTransports mode")
