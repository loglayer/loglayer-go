// Package console provides a Transport that writes log entries to stdout/stderr.
package console

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
	"go.loglayer.dev/utils/sanitize"
)

// Config holds configuration options for Transport.
type Config struct {
	transport.BaseConfig

	// AppendObjectData appends data after messages (default: prepend).
	// Has no effect when MessageField is set.
	AppendObjectData bool

	// MessageField places the joined message text into this key, producing a
	// single structured object as the sole log argument.
	// When set, DateField and LevelField are also included in that object.
	MessageField string

	// DateField adds a timestamp entry to the structured output.
	// If MessageField is not set, date is added as an extra argument.
	DateField string

	// LevelField adds the log level to the structured output.
	// If MessageField is not set, level is added as an extra argument.
	LevelField string

	// DateFn overrides the default ISO-8601 timestamp when DateField is set.
	DateFn func() string

	// LevelFn overrides the default level string when LevelField is set.
	LevelFn func(loglayer.LogLevel) string

	// Stringify JSON-encodes the structured object instead of passing it raw.
	// Only applies when MessageField, DateField, or LevelField is set.
	Stringify bool

	// MessageFn, when set, formats the entire log output as a single string.
	// It receives the full TransportParams and its return value replaces messages.
	MessageFn func(params loglayer.TransportParams) string

	// Writer overrides the default stdout/stderr selection.
	// When set all log levels write to this writer.
	Writer io.Writer
}

// Transport writes log entries to stdout (info/debug/trace) or stderr (warn/error/fatal).
type Transport struct {
	transport.BaseTransport
	cfg Config
}

// New creates a Transport from the given Config.
func New(cfg Config) *Transport {
	return &Transport{
		BaseTransport: transport.NewBaseTransport(cfg.BaseConfig),
		cfg:           cfg,
	}
}

// GetLoggerInstance returns nil; console transport has no underlying logger library.
func (c *Transport) GetLoggerInstance() any { return nil }

// SendToLogger implements loglayer.Transport.
func (c *Transport) SendToLogger(params loglayer.TransportParams) {
	if !c.ShouldProcess(params.LogLevel) {
		return
	}
	messages := buildMessages(params, c.cfg)
	fmt.Fprintln(c.writer(params.LogLevel), messages...)
}

// writer picks the appropriate output stream for a log level.
func (c *Transport) writer(level loglayer.LogLevel) io.Writer {
	if c.cfg.Writer != nil {
		return c.cfg.Writer
	}
	switch level {
	case loglayer.LogLevelWarn, loglayer.LogLevelError, loglayer.LogLevelFatal:
		return os.Stderr
	default:
		return os.Stdout
	}
}

// buildMessages assembles the argument list to pass to Fprintln.
func buildMessages(params loglayer.TransportParams, cfg Config) []any {
	messages := make([]any, len(params.Messages))
	copy(messages, params.Messages)

	if cfg.MessageFn != nil {
		messages = []any{cfg.MessageFn(params)}
	}

	// Sanitize user-controlled message strings so a CRLF or ANSI ESC
	// can't forge log lines or smuggle terminal escapes through the
	// renderer. Non-string elements pass through; their %v rendering
	// doesn't introduce control characters.
	for i, m := range messages {
		if s, ok := m.(string); ok {
			messages[i] = sanitize.Message(s)
		}
	}

	combined := transport.MergeFieldsAndMetadata(params)
	hasCombined := len(combined) > 0

	if cfg.MessageField != "" {
		obj := make(map[string]any, len(combined)+3)
		for k, v := range combined {
			obj[k] = v
		}
		obj[cfg.MessageField] = transport.JoinMessages(messages)
		if cfg.DateField != "" {
			obj[cfg.DateField] = dateValue(cfg)
		}
		if cfg.LevelField != "" {
			obj[cfg.LevelField] = levelValue(cfg, params.LogLevel)
		}
		return []any{maybeStringify(obj, cfg.Stringify)}
	}

	if cfg.DateField != "" || cfg.LevelField != "" {
		obj := make(map[string]any, len(combined)+2)
		for k, v := range combined {
			obj[k] = v
		}
		if cfg.DateField != "" {
			obj[cfg.DateField] = dateValue(cfg)
		}
		if cfg.LevelField != "" {
			obj[cfg.LevelField] = levelValue(cfg, params.LogLevel)
		}
		messages = append(messages, maybeStringify(obj, cfg.Stringify))
		return messages
	}

	if hasCombined {
		if cfg.AppendObjectData {
			messages = append(messages, combined)
		} else {
			messages = append([]any{combined}, messages...)
		}
	}

	return messages
}

func dateValue(cfg Config) string {
	if cfg.DateFn != nil {
		return cfg.DateFn()
	}
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func levelValue(cfg Config, level loglayer.LogLevel) string {
	if cfg.LevelFn != nil {
		return cfg.LevelFn(level)
	}
	return level.String()
}

func maybeStringify(obj map[string]any, stringify bool) any {
	if !stringify {
		return obj
	}
	b, err := json.Marshal(obj)
	if err != nil {
		return obj
	}
	return string(b)
}
