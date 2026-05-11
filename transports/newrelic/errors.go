package newrelic

import "errors"

// ErrLicenseKeyRequired is returned by Build (and panicked by New) when
// Config.LicenseKey is empty.
var ErrLicenseKeyRequired = errors.New("loglayer/transports/newrelic: Config.LicenseKey is required")

// ErrInsecureURL is returned by Build (and panicked by New) when
// Config.URL has a non-https scheme. The license key travels in the
// Api-Key header on every request; refuse to ship it in cleartext.
var ErrInsecureURL = errors.New("loglayer/transports/newrelic: Config.URL must use https")

// ErrHTTPOverrideForbidden is returned when Config.HTTP.URL or
// Config.HTTP.Encoder is non-zero. These fields are managed by this
// package. Set Config.URL on the New Relic config instead; the encoder
// cannot be customized.
var ErrHTTPOverrideForbidden = errors.New("loglayer/transports/newrelic: Config.HTTP.URL and Config.HTTP.Encoder are managed by this package and must be left zero")
