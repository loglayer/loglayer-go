package sentrytransport

import "errors"

// ErrLoggerRequired is returned by Build (and panicked by New) when
// Config.Logger is nil. The user supplies the Sentry logger (typically
// from sentry.NewLogger(ctx)); the transport never constructs one
// itself, so a nil logger can't be defaulted.
var ErrLoggerRequired = errors.New("loglayer/transports/sentry: Config.Logger is required")
