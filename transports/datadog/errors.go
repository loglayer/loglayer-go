package datadog

import "errors"

// ErrAPIKeyRequired is returned by Build (and panicked by New) when
// Config.APIKey is empty.
var ErrAPIKeyRequired = errors.New("loglayer/transports/datadog: Config.APIKey is required")
