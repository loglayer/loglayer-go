package gcplogging

import "errors"

// ErrLoggerRequired is returned by Build (and panicked by New) when
// Config.Logger is nil. The user supplies the GCP logger (typically
// via logging.NewClient(ctx, projectID).Logger(logID)); the transport
// never constructs one itself, so a nil logger can't be defaulted.
var ErrLoggerRequired = errors.New("loglayer/transports/gcplogging: Config.Logger is required")
