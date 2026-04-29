package lumberjack

import "errors"

// ErrFilenameRequired is returned by Build (and panicked by New) when
// Config.Filename is empty.
var ErrFilenameRequired = errors.New("loglayer/transports/lumberjack: Config.Filename is required")
