// Package charmlog provides a LogLayer transport backed by github.com/charmbracelet/log.
//
// See https://go.loglayer.dev for usage guides and the full API reference.
package charmlog

import (
	"io"

	clog "github.com/charmbracelet/log"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
)

// Config holds configuration options for the charmbracelet/log transport.
type Config struct {
	transport.BaseConfig

	// Logger is the underlying *charmbracelet/log.Logger. When nil a default
	// logger writing to Writer is constructed.
	//
	// Note: The charmbracelet Logger.Fatal() shortcut calls os.Exit. This
	// transport always dispatches via Logger.Log(level, msg, keyvals...) which
	// does not exit, so loglayer's "Fatal does not exit" contract holds.
	Logger *clog.Logger

	// Writer is used only when Logger is nil. Defaults to os.Stderr.
	Writer io.Writer
}

// Transport sends log entries to a *charmbracelet/log.Logger.
type Transport struct {
	transport.BaseTransport
	cfg    Config
	logger *clog.Logger
}

// New creates a charmlog Transport from the given Config.
func New(cfg Config) *Transport {
	logger := cfg.Logger
	if logger == nil {
		logger = clog.NewWithOptions(transport.WriterOrStderr(cfg.Writer), clog.Options{Level: clog.DebugLevel})
	}
	return &Transport{
		BaseTransport: transport.NewBaseTransport(cfg.BaseConfig),
		cfg:           cfg,
		logger:        logger,
	}
}

// GetLoggerInstance returns the underlying *charmbracelet/log.Logger.
func (t *Transport) GetLoggerInstance() any { return t.logger }

// SendToLogger implements loglayer.Transport. Dispatches via Log so fatal
// entries do not call os.Exit (charmbracelet's Log method, unlike Fatal,
// does not exit; the core decides via Config.DisableFatalExit).
func (t *Transport) SendToLogger(params loglayer.TransportParams) {
	if !t.ShouldProcess(params.LogLevel) {
		return
	}
	keyvals := make([]any, 0, transport.FieldEstimate(params)*2)

	if len(params.Data) > 0 {
		for k, v := range params.Data {
			keyvals = append(keyvals, k, v)
		}
	}

	if params.Metadata != nil {
		if key := params.Schema.MetadataFieldName; key != "" {
			keyvals = append(keyvals, key, params.Metadata)
		} else if m, ok := transport.MetadataAsRootMap(params.Metadata); ok {
			for k, v := range m {
				keyvals = append(keyvals, k, v)
			}
		} else {
			keyvals = append(keyvals, "metadata", params.Metadata)
		}
	}

	t.logger.Log(toCharmLevel(params.LogLevel), transport.JoinMessages(params.Messages), keyvals...)
}

// toCharmLevel maps loglayer levels to charmbracelet/log levels.
func toCharmLevel(l loglayer.LogLevel) clog.Level {
	switch l {
	case loglayer.LogLevelTrace:
		// charmbracelet/log has no Trace; map to its lowest level.
		return clog.DebugLevel
	case loglayer.LogLevelDebug:
		return clog.DebugLevel
	case loglayer.LogLevelInfo:
		return clog.InfoLevel
	case loglayer.LogLevelWarn:
		return clog.WarnLevel
	case loglayer.LogLevelError:
		return clog.ErrorLevel
	case loglayer.LogLevelFatal:
		return clog.FatalLevel
	case loglayer.LogLevelPanic:
		// No Panic level in charmbracelet/log; surface as Fatal so the
		// rendering at least signals "highest severity". The actual
		// panic() is triggered by loglayer's dispatch path regardless.
		return clog.FatalLevel
	default:
		return clog.InfoLevel
	}
}
