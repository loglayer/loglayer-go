// Package zap provides a LogLayer transport backed by go.uber.org/zap.
//
// See https://go.loglayer.dev for usage guides and the full API reference.
package zap

import (
	"io"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.loglayer.dev/v2"
	"go.loglayer.dev/v2/transport"
)

// Config holds configuration options for the zap transport.
type Config struct {
	transport.BaseConfig

	// Logger is the underlying *zap.Logger. When nil a default logger writing
	// to Writer with a JSON encoder is constructed. Provide your own logger
	// to share encoders, hooks, samplers, or fields already configured.
	//
	// Whichever logger is used, this transport always wraps it with
	// zap.WithFatalHook(noopFatalHook{}) so loglayer's contract that
	// Fatal does NOT terminate the process is honored. (zap silently
	// reverts zapcore.WriteThenNoop back to WriteThenFatal, so a custom
	// no-op hook is required.)
	Logger *zap.Logger

	// Writer is used only when Logger is nil. Defaults to os.Stderr.
	Writer io.Writer
}

// Transport sends log entries to a *zap.Logger.
type Transport struct {
	transport.BaseTransport
	cfg    Config
	logger *zap.Logger
}

// New creates a zap Transport from the given Config.
func New(cfg Config) *Transport {
	var base *zap.Logger
	if cfg.Logger != nil {
		base = cfg.Logger
	} else {
		enc := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
		core := zapcore.NewCore(enc, zapcore.AddSync(transport.WriterOrStderr(cfg.Writer)), zapcore.DebugLevel)
		base = zap.New(core)
	}
	logger := base.WithOptions(zap.WithFatalHook(noopFatalHook{}))
	return &Transport{
		BaseTransport: transport.NewBaseTransport(cfg.BaseConfig),
		cfg:           cfg,
		logger:        logger,
	}
}

// GetLoggerInstance returns the underlying *zap.Logger.
func (t *Transport) GetLoggerInstance() any { return t.logger }

// SendToLogger implements loglayer.Transport.
func (t *Transport) SendToLogger(params loglayer.TransportParams) {
	if !t.ShouldProcess(params.LogLevel) {
		return
	}
	// Fold the prefix into Messages[0] for the rendered output;
	// transports own this rendering choice.
	params.Messages = transport.JoinPrefixAndMessages(params.Prefix, params.Messages)
	fields := make([]zap.Field, 0, transport.FieldEstimate(params))

	if len(params.Data) > 0 {
		for k, v := range params.Data {
			fields = append(fields, zap.Any(k, v))
		}
	}

	if params.Metadata != nil {
		if key := params.Schema.MetadataFieldName; key != "" {
			fields = append(fields, zap.Any(key, params.Metadata))
		} else if m, ok := transport.MetadataAsRootMap(params.Metadata); ok {
			for k, v := range m {
				fields = append(fields, zap.Any(k, v))
			}
		} else {
			fields = append(fields, zap.Any("metadata", params.Metadata))
		}
	}

	t.logger.Log(toZapLevel(params.LogLevel), transport.JoinMessages(params.Messages), fields...)
}

// noopFatalHook keeps Fatal-level entries from terminating the process.
type noopFatalHook struct{}

func (noopFatalHook) OnWrite(*zapcore.CheckedEntry, []zapcore.Field) {}

// toZapLevel maps loglayer levels to zapcore.Level.
func toZapLevel(l loglayer.LogLevel) zapcore.Level {
	switch l {
	case loglayer.LogLevelTrace:
		// zap has no Trace level; map to its lowest.
		return zapcore.DebugLevel
	case loglayer.LogLevelDebug:
		return zapcore.DebugLevel
	case loglayer.LogLevelInfo:
		return zapcore.InfoLevel
	case loglayer.LogLevelWarn:
		return zapcore.WarnLevel
	case loglayer.LogLevelError:
		return zapcore.ErrorLevel
	case loglayer.LogLevelFatal:
		return zapcore.FatalLevel
	case loglayer.LogLevelPanic:
		return zapcore.PanicLevel
	default:
		return zapcore.InfoLevel
	}
}
