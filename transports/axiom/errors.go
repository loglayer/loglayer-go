package axiom

import "errors"

// ErrClientRequired is returned by Build (and panicked by New) when
// Config.Client is nil. The user supplies the Axiom client; the transport
// never constructs one itself, so a nil client can't be defaulted.
var ErrClientRequired = errors.New("loglayer/transports/axiom: Config.Client is required")

// ErrDatasetNameRequired is returned by Build (and panicked by New) when
// Config.DatasetName is empty. The dataset ID/name is required to know
// where to ingest logs.
var ErrDatasetNameRequired = errors.New("loglayer/transports/axiom: Config.DatasetName is required")
