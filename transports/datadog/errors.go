package datadog

import "errors"

// ErrAPIKeyRequired is returned by Build (and panicked by New) when
// Config.APIKey is empty.
var ErrAPIKeyRequired = errors.New("loglayer/transports/datadog: Config.APIKey is required")

// ErrInsecureURL is returned by Build (and panicked by New) when
// Config.URL has a non-https scheme. Datadog's API key would be sent in
// cleartext over http; refuse rather than ship credentials in the open.
var ErrInsecureURL = errors.New("loglayer/transports/datadog: Config.URL must use https")

// ErrHTTPOverrideForbidden is returned by Build (and panicked by New)
// when Config.HTTP.URL or Config.HTTP.Encoder is non-zero. The Datadog
// transport sets these itself (URL from Config.URL or Config.Site,
// Encoder from the package's Datadog-format builder); a value supplied
// via the embedded HTTP config would be silently dropped, which used
// to surprise callers. Set Config.URL on the Datadog config instead;
// the Encoder cannot be customized.
var ErrHTTPOverrideForbidden = errors.New("loglayer/transports/datadog: Config.HTTP.URL and Config.HTTP.Encoder are managed by this package and must be left zero")
