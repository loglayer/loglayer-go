// Package phuslu provides a LogLayer transport backed by github.com/phuslu/log.
package phuslu

import (
	"io"

	plog "github.com/phuslu/log"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
)

// Config holds configuration options for the phuslu transport.
type Config struct {
	transport.BaseConfig

	// Logger is the underlying *phuslu/log.Logger. When nil a default logger
	// writing to Writer at TraceLevel is constructed.
	//
	// Caveat: phuslu calls os.Exit on FatalLevel from any dispatch path,
	// including Logger.WithLevel(FatalLevel).Msg(). This wrapper cannot
	// suppress that behavior. If a fatal-level log is emitted via this
	// transport, the process WILL exit — even when loglayer.Config.DisableFatalExit
	// is set to true. Use a different transport for tests or library code that
	// must avoid os.Exit on fatal.
	Logger *plog.Logger

	// Writer is used only when Logger is nil. Defaults to os.Stderr.
	Writer io.Writer

	// MetadataFieldName is the key under which non-map metadata values are
	// emitted (structs, scalars, slices, etc.). Map metadata is always merged
	// at the root. Defaults to "metadata".
	MetadataFieldName string
}

// Transport sends log entries to a *phuslu/log.Logger.
type Transport struct {
	transport.BaseTransport
	cfg    Config
	logger *plog.Logger
}

// New creates a phuslu Transport from the given Config.
func New(cfg Config) *Transport {
	if cfg.MetadataFieldName == "" {
		cfg.MetadataFieldName = "metadata"
	}
	logger := cfg.Logger
	if logger == nil {
		logger = &plog.Logger{
			Level:  plog.TraceLevel,
			Writer: &plog.IOWriter{Writer: transport.WriterOrStderr(cfg.Writer)},
		}
	}
	return &Transport{
		BaseTransport: transport.NewBaseTransport(cfg.BaseConfig),
		cfg:           cfg,
		logger:        logger,
	}
}

// GetLoggerInstance returns the underlying *phuslu/log.Logger.
func (t *Transport) GetLoggerInstance() any { return t.logger }

// SendToLogger implements loglayer.Transport.
//
// Note: phuslu calls os.Exit on FatalLevel through any dispatch path; this
// wrapper cannot suppress that behavior. See the package doc.
func (t *Transport) SendToLogger(params loglayer.TransportParams) {
	if !t.ShouldProcess(params.LogLevel) {
		return
	}
	entry := t.logger.WithLevel(toPhusluLevel(params.LogLevel))
	if entry == nil {
		return
	}

	if len(params.Data) > 0 {
		for k, v := range params.Data {
			entry = entry.Any(k, v)
		}
	}

	if params.Metadata != nil {
		if m, ok := transport.MetadataAsRootMap(params.Metadata); ok {
			for k, v := range m {
				entry = entry.Any(k, v)
			}
		} else {
			entry = entry.Any(t.cfg.MetadataFieldName, params.Metadata)
		}
	}

	entry.Msg(transport.JoinMessages(params.Messages))
}

// toPhusluLevel maps loglayer levels to phuslu levels.
func toPhusluLevel(l loglayer.LogLevel) plog.Level {
	switch l {
	case loglayer.LogLevelTrace:
		return plog.TraceLevel
	case loglayer.LogLevelDebug:
		return plog.DebugLevel
	case loglayer.LogLevelInfo:
		return plog.InfoLevel
	case loglayer.LogLevelWarn:
		return plog.WarnLevel
	case loglayer.LogLevelError:
		return plog.ErrorLevel
	case loglayer.LogLevelFatal:
		return plog.FatalLevel
	case loglayer.LogLevelPanic:
		return plog.PanicLevel
	default:
		return plog.InfoLevel
	}
}
