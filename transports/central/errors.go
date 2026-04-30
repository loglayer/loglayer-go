package central

import "errors"

// ErrServiceRequired is returned by Build (and panicked by New) when
// Config.Service is empty. The Central server uses this field to identify
// the source application; without it the entry can't be routed.
var ErrServiceRequired = errors.New("loglayer/transports/central: Config.Service is required")

// ErrHTTPOverrideForbidden is returned by Build (and panicked by New) when
// Config.HTTP.URL or Config.HTTP.Encoder is non-zero. The Central transport
// sets these itself (URL from Config.BaseURL, Encoder from this package's
// Central-format builder); a value supplied via the embedded HTTP config
// would be silently dropped.
var ErrHTTPOverrideForbidden = errors.New("loglayer/transports/central: Config.HTTP.URL and Config.HTTP.Encoder are managed by this package and must be left zero")
