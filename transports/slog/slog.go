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

	if params.HasData {
		for k, v := range params.Data {
			attrs = append(attrs, slog.Any(k, v))
		}
	}

	if params.Metadata != nil {
		if m, ok := params.Metadata.(map[string]any); ok {
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
// Trace and Fatal don't have direct slog equivalents:
//   - Trace collapses to Debug (slog has no level below Debug).
//   - Fatal maps to slog.LevelError + 4 to keep it above Error in any
//     downstream handler. The actual os.Exit decision is made by the core's
//     Config.DisableFatalExit; slog itself never exits.
func toSlogLevel(l loglayer.LogLevel) slog.Level {
	switch l {
	case loglayer.LogLevelTrace, loglayer.LogLevelDebug:
		return slog.LevelDebug
	case loglayer.LogLevelInfo:
		return slog.LevelInfo
	case loglayer.LogLevelWarn:
		return slog.LevelWarn
	case loglayer.LogLevelError:
		return slog.LevelError
	case loglayer.LogLevelFatal:
		return slog.LevelError + 4
	default:
		return slog.LevelInfo
	}
}
