// Package zerolog provides a LogLayer transport backed by github.com/rs/zerolog.
package zerolog

import (
	"io"

	zlog "github.com/rs/zerolog"

	"go.loglayer.dev/loglayer"
	"go.loglayer.dev/loglayer/transport"
)

// Config holds configuration options for the zerolog transport.
type Config struct {
	transport.BaseConfig

	// Logger is the underlying zerolog.Logger. When nil a default logger writing
	// to Writer is constructed. Use this to share a logger that already has
	// hooks, samplers, or formatters configured.
	Logger *zlog.Logger

	// Writer is used only when Logger is nil. Defaults to os.Stderr.
	Writer io.Writer

	// MetadataFieldName is the key under which non-map metadata values are
	// emitted (structs, scalars, slices, etc.). Map metadata is always merged
	// at the root. Defaults to "metadata".
	MetadataFieldName string

	// DisableFatalExit overrides the underlying zerolog behavior when handling
	// fatal-level events. loglayer's contract is that Fatal does NOT terminate
	// the process — this transport always honors that contract regardless of
	// this setting (kept here only to flag the intentional divergence).
	DisableFatalExit bool
}

// Transport sends log entries to a zerolog.Logger.
type Transport struct {
	transport.BaseTransport
	cfg    Config
	logger zlog.Logger
}

// New creates a zerolog Transport from the given Config.
func New(cfg Config) *Transport {
	if cfg.MetadataFieldName == "" {
		cfg.MetadataFieldName = "metadata"
	}
	var logger zlog.Logger
	if cfg.Logger != nil {
		logger = *cfg.Logger
	} else {
		logger = zlog.New(transport.WriterOrStderr(cfg.Writer)).With().Timestamp().Logger()
	}
	return &Transport{
		BaseTransport: transport.NewBaseTransport(cfg.BaseConfig),
		cfg:           cfg,
		logger:        logger,
	}
}

// GetLoggerInstance returns a pointer to the underlying zerolog.Logger so callers
// can use it directly when they need zerolog-specific features.
func (t *Transport) GetLoggerInstance() any { return &t.logger }

// SendToLogger implements loglayer.Transport.
func (t *Transport) SendToLogger(params loglayer.TransportParams) {
	if !t.ShouldProcess(params.LogLevel) {
		return
	}
	event := t.logger.WithLevel(toZerologLevel(params.LogLevel))
	if event == nil {
		return
	}

	if params.HasData {
		event = event.Fields(map[string]any(params.Data))
	}

	if params.Metadata != nil {
		if m, ok := params.Metadata.(map[string]any); ok {
			event = event.Fields(m)
		} else {
			event = event.Interface(t.cfg.MetadataFieldName, params.Metadata)
		}
	}

	event.Msg(transport.JoinMessages(params.Messages))
}

// toZerologLevel maps loglayer levels to zerolog levels.
//
// loglayer's contract is that Fatal does NOT call os.Exit; we deliberately use
// WithLevel(FatalLevel) above (instead of .Fatal()) which writes the entry but
// never terminates the process.
func toZerologLevel(l loglayer.LogLevel) zlog.Level {
	switch l {
	case loglayer.LogLevelTrace:
		return zlog.TraceLevel
	case loglayer.LogLevelDebug:
		return zlog.DebugLevel
	case loglayer.LogLevelInfo:
		return zlog.InfoLevel
	case loglayer.LogLevelWarn:
		return zlog.WarnLevel
	case loglayer.LogLevelError:
		return zlog.ErrorLevel
	case loglayer.LogLevelFatal:
		return zlog.FatalLevel
	default:
		return zlog.InfoLevel
	}
}
