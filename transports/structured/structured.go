// Package structured provides a StructuredTransport that always outputs a single
// JSON object per log entry with msg, level, and time fields by default.
package structured

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
)

// Config holds configuration options for StructuredTransport.
type Config struct {
	transport.BaseConfig

	// MessageField is the key for the joined message text. Defaults to "msg".
	MessageField string

	// DateField is the key for the timestamp. Defaults to "time".
	DateField string

	// LevelField is the key for the log level. Defaults to "level".
	LevelField string

	// DateFn overrides the default ISO-8601 timestamp.
	DateFn func() string

	// LevelFn overrides the default level string.
	LevelFn func(loglayer.LogLevel) string

	// MessageFn formats the message portion. Its return value is used as the message text.
	MessageFn func(params loglayer.TransportParams) string

	// Writer overrides the default output (os.Stdout).
	Writer io.Writer
}

// StructuredTransport always outputs one JSON object per log entry.
// All messages are joined with a space and placed under MessageField.
type StructuredTransport struct {
	transport.BaseTransport
	cfg Config
}

// New creates a StructuredTransport from the given Config.
func New(cfg Config) *StructuredTransport {
	if cfg.MessageField == "" {
		cfg.MessageField = "msg"
	}
	if cfg.DateField == "" {
		cfg.DateField = "time"
	}
	if cfg.LevelField == "" {
		cfg.LevelField = "level"
	}
	return &StructuredTransport{
		BaseTransport: transport.NewBaseTransport(cfg.BaseConfig),
		cfg:           cfg,
	}
}

// GetLoggerInstance returns nil; structured transport has no underlying logger library.
func (s *StructuredTransport) GetLoggerInstance() any { return nil }

// SendToLogger implements loglayer.Transport.
func (s *StructuredTransport) SendToLogger(params loglayer.TransportParams) {
	if !s.ShouldProcess(params.LogLevel) {
		return
	}
	messages := params.Messages
	if s.cfg.MessageFn != nil {
		messages = []any{s.cfg.MessageFn(params)}
	}

	obj := make(map[string]any)

	obj[s.cfg.LevelField] = s.levelValue(params.LogLevel)
	obj[s.cfg.DateField] = s.dateValue()
	obj[s.cfg.MessageField] = transport.JoinMessages(messages)

	if params.HasData {
		for k, v := range params.Data {
			obj[k] = v
		}
	}

	if params.Metadata != nil {
		for k, v := range transport.MetadataAsMap(params.Metadata) {
			obj[k] = v
		}
	}

	b, err := json.Marshal(obj)
	if err != nil {
		fmt.Fprintf(s.writer(), `{"level":"error","msg":"loglayer: failed to marshal log entry","error":%q}`+"\n", err.Error())
		return
	}
	fmt.Fprintln(s.writer(), string(b))
}

func (s *StructuredTransport) writer() io.Writer {
	if s.cfg.Writer != nil {
		return s.cfg.Writer
	}
	return os.Stdout
}

func (s *StructuredTransport) dateValue() string {
	if s.cfg.DateFn != nil {
		return s.cfg.DateFn()
	}
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func (s *StructuredTransport) levelValue(level loglayer.LogLevel) string {
	if s.cfg.LevelFn != nil {
		return s.cfg.LevelFn(level)
	}
	return level.String()
}
