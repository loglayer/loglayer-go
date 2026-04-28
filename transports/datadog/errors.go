package datadog

import "errors"

// ErrAPIKeyRequired is returned by Build (and panicked by New) when
// Config.APIKey is empty.
var ErrAPIKeyRequired = errors.New("loglayer/transports/datadog: Config.APIKey is required")

// ErrInsecureURL is returned by Build (and panicked by New) when
// Config.URL has a non-https scheme. Datadog's API key would be sent in
// cleartext over http; refuse rather than ship credentials in the open.
var ErrInsecureURL = errors.New("loglayer/transports/datadog: Config.URL must use https")
