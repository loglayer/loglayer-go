// Package logrus provides a LogLayer transport backed by github.com/sirupsen/logrus.
package logrus

import (
	"io"
	"os"

	"github.com/sirupsen/logrus"

	"go.loglayer.dev/loglayer"
	"go.loglayer.dev/loglayer/transport"
)

// Config holds configuration options for the logrus transport.
type Config struct {
	transport.BaseConfig

	// Logger is the underlying *logrus.Logger to wrap. When nil a default
	// logrus.Logger is constructed writing to Writer with the default
	// TextFormatter.
	//
	// Whichever logger is supplied, this transport always builds a fresh
	// *logrus.Logger that copies the supplied logger's settings (Out, Hooks,
	// Formatter, ReportCaller, Level, BufferPool) but with ExitFunc set to a
	// no-op. That way logrus.FatalLevel writes the entry without terminating
	// the process — honoring loglayer's "Fatal does not exit" contract — and
	// the user's original *logrus.Logger is never mutated.
	Logger *logrus.Logger

	// Writer is used only when Logger is nil. Defaults to os.Stderr.
	Writer io.Writer

	// MetadataFieldName is the key under which non-map metadata values are
	// emitted (structs, scalars, slices, etc.). Map metadata is always merged
	// at the root. Defaults to "metadata".
	MetadataFieldName string
}

// Transport sends log entries to a *logrus.Logger.
type Transport struct {
	transport.BaseTransport
	cfg    Config
	logger *logrus.Logger
}

// New creates a logrus Transport from the given Config.
func New(cfg Config) *Transport {
	if cfg.MetadataFieldName == "" {
		cfg.MetadataFieldName = "metadata"
	}
	logger := wrap(cfg.Logger, cfg.Writer)
	return &Transport{
		BaseTransport: transport.NewBaseTransport(cfg.BaseConfig),
		cfg:           cfg,
		logger:        logger,
	}
}

// wrap clones the supplied logger's settings into a fresh *logrus.Logger with
// ExitFunc neutralized so Fatal entries never call os.Exit.
func wrap(src *logrus.Logger, fallbackWriter io.Writer) *logrus.Logger {
	out := logrus.New()
	out.ExitFunc = func(int) {} // never terminate the process

	if src != nil {
		out.Out = src.Out
		out.Hooks = src.Hooks
		out.Formatter = src.Formatter
		out.ReportCaller = src.ReportCaller
		out.Level = src.Level
		out.BufferPool = src.BufferPool
	} else {
		w := fallbackWriter
		if w == nil {
			w = os.Stderr
		}
		out.Out = w
		out.Level = logrus.TraceLevel
	}
	return out
}

// GetLoggerInstance returns the wrapped *logrus.Logger (the one with the
// neutralized ExitFunc, not the original).
func (t *Transport) GetLoggerInstance() any { return t.logger }

// SendToLogger implements loglayer.Transport.
func (t *Transport) SendToLogger(params loglayer.TransportParams) {
	if !t.ShouldProcess(params.LogLevel) {
		return
	}
	fields := logrus.Fields{}

	if params.HasData {
		for k, v := range params.Data {
			fields[k] = v
		}
	}

	if params.Metadata != nil {
		if m, ok := params.Metadata.(map[string]any); ok {
			for k, v := range m {
				fields[k] = v
			}
		} else {
			fields[t.cfg.MetadataFieldName] = params.Metadata
		}
	}

	entry := t.logger.WithFields(fields)
	entry.Log(toLogrusLevel(params.LogLevel), transport.JoinMessages(params.Messages))
}

// toLogrusLevel maps loglayer levels to logrus levels.
func toLogrusLevel(l loglayer.LogLevel) logrus.Level {
	switch l {
	case loglayer.LogLevelTrace:
		return logrus.TraceLevel
	case loglayer.LogLevelDebug:
		return logrus.DebugLevel
	case loglayer.LogLevelInfo:
		return logrus.InfoLevel
	case loglayer.LogLevelWarn:
		return logrus.WarnLevel
	case loglayer.LogLevelError:
		return logrus.ErrorLevel
	case loglayer.LogLevelFatal:
		return logrus.FatalLevel
	default:
		return logrus.InfoLevel
	}
}
