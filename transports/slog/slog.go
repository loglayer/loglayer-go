// Package slog provides a LogLayer transport backed by the standard library's
// log/slog package.
//
// Use this transport when your codebase or its dependencies are wired against
// *slog.Logger and you want LogLayer's API on top, or when you want to plug
// LogLayer into a slog-based handler stack (JSON, text, OpenTelemetry, etc).
package slog

import (
	"context"
	"io"
	"log/slog"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
)

// Config holds configuration options for the slog transport.
type Config struct {
	transport.BaseConfig

	// Logger is the underlying *slog.Logger to wrap. When nil a default logger
	// using slog.NewJSONHandler writing to Writer is constructed.
	Logger *slog.Logger

	// Writer is used only when Logger is nil. Defaults to os.Stderr.
	Writer io.Writer

	// MetadataFieldName is the key under which non-map metadata values are
	// emitted (structs, scalars, slices, etc.). Map metadata is always merged
	// at the root via individual slog attributes. Defaults to "metadata".
	MetadataFieldName string
}

// Transport sends log entries to a *slog.Logger.
type Transport struct {
	transport.BaseTransport
	cfg    Config
	logger *slog.Logger
}

// New creates a slog Transport from the given Config.
func New(cfg Config) *Transport {
	if cfg.MetadataFieldName == "" {
		cfg.MetadataFieldName = "metadata"
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(transport.WriterOrStderr(cfg.Writer), nil))
	}
	return &Transport{
		BaseTransport: transport.NewBaseTransport(cfg.BaseConfig),
		cfg:           cfg,
		logger:        logger,
	}
}

// GetLoggerInstance returns the underlying *slog.Logger.
func (t *Transport) GetLoggerInstance() any { return t.logger }

// SendToLogger implements loglayer.Transport.
//
// Dispatches via slog.Logger.LogAttrs which does not call os.Exit on any
// level: the core's Config.DisableFatalExit setting controls termination.
func (t *Transport) SendToLogger(params loglayer.TransportParams) {
	if !t.ShouldProcess(params.LogLevel) {
		return
	}

	attrs := make([]slog.Attr, 0, transport.FieldEstimate(params))

	if len(params.Data) > 0 {
		for k, v := range params.Data {
			attrs = append(attrs, slog.Any(k, v))
		}
	}

	if params.Metadata != nil {
		if m, ok := transport.MetadataAsRootMap(params.Metadata); ok {
			for k, v := range m {
				attrs = append(attrs, slog.Any(k, v))
			}
		} else {
			attrs = append(attrs, slog.Any(t.cfg.MetadataFieldName, params.Metadata))
		}
	}

	ctx := params.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	t.logger.LogAttrs(ctx, toSlogLevel(params.LogLevel), transport.JoinMessages(params.Messages), attrs...)
}

// toSlogLevel maps loglayer levels to slog levels.
//
// slog has no Trace, Fatal, or Panic levels of its own, but slog.Level
// is just an int so we synthesize them by offsetting from the named
// neighbours. Trace = LevelDebug-4 (renders as "DEBUG-4"). Fatal =
// LevelError+4 ("ERROR+4"). Panic = LevelError+8 ("ERROR+8"). The actual
// os.Exit / panic decisions are made by the core; slog itself never
// exits or panics through this transport.
func toSlogLevel(l loglayer.LogLevel) slog.Level {
	switch l {
	case loglayer.LogLevelTrace:
		return slog.LevelDebug - 4
	case loglayer.LogLevelDebug:
		return slog.LevelDebug
	case loglayer.LogLevelInfo:
		return slog.LevelInfo
	case loglayer.LogLevelWarn:
		return slog.LevelWarn
	case loglayer.LogLevelError:
		return slog.LevelError
	case loglayer.LogLevelFatal:
		return slog.LevelError + 4
	case loglayer.LogLevelPanic:
		return slog.LevelError + 8
	default:
		return slog.LevelInfo
	}
}
