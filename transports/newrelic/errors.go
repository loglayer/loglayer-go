package newrelic

import "errors"

// ErrAPIKeyRequired is returned by Build (and panicked by New) when
// Config.APIKey is empty.
var ErrAPIKeyRequired = errors.New("loglayer/transports/newrelic: Config.APIKey is required")

// ErrHTTPOverrideForbidden is returned by Build (and panicked by New)
// when Config.HTTP.URL or Config.HTTP.Encoder is non-zero. The New Relic
// transport sets these itself (URL from the fixed Log API endpoint,
// Encoder from the package's NDJSON builder); a value supplied via the
// embedded HTTP config would be silently dropped, which used to surprise
// callers. The Encoder cannot be customized.
var ErrHTTPOverrideForbidden = errors.New("loglayer/transports/newrelic: Config.HTTP.URL and Config.HTTP.Encoder are managed by this package and must be left zero")

// ErrInsecureURL is returned by Build (and panicked by New) when
// Config.URL has a non-https scheme. The New Relic API key would be sent
// in cleartext over http; refuse rather than ship credentials in the open.
var ErrInsecureURL = errors.New("loglayer/transports/newrelic: Config.URL must use https")
