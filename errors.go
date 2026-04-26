package loglayer

import "errors"

// ErrNoTransport is returned by Build (and panicked by New) when a Config
// is constructed with neither Transport nor Transports set.
var ErrNoTransport = errors.New("loglayer: at least one transport must be provided")
