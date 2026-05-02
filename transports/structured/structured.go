// Package structured provides a Transport that always outputs a single
// JSON object per log entry with msg, level, and time fields by default.
//
// See https://go.loglayer.dev for usage guides and the full API reference.
package structured

import (
	"bytes"
	"io"
	"sync"
	"time"

	"github.com/goccy/go-json"

	"go.loglayer.dev/v2"
	"go.loglayer.dev/v2/transport"
)

// Config holds configuration options for Transport.
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

// maxPooledBufCap caps the buffer capacity returned to bufPool so a single
// oversized log entry can't pin its capacity in the pool indefinitely.
const maxPooledBufCap = 64 << 10

// rfc3339NanoMaxLen bounds time.RFC3339Nano output (30 bytes + headroom).
const rfc3339NanoMaxLen = 32

var bufPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

func putBuffer(buf *bytes.Buffer) {
	if buf.Cap() > maxPooledBufCap {
		return
	}
	bufPool.Put(buf)
}

// Pre-quoted level names. writeLevel writes one of these directly when
// LevelFn is nil, keeping json.Marshal off the simple-message hot path.
var (
	jsonDebug = []byte(`"debug"`)
	jsonInfo  = []byte(`"info"`)
	jsonWarn  = []byte(`"warn"`)
	jsonError = []byte(`"error"`)
	jsonFatal = []byte(`"fatal"`)
)

// Transport always outputs one JSON object per log entry.
// All messages are joined with a space and placed under MessageField.
type Transport struct {
	transport.BaseTransport
	cfg    Config
	writer io.Writer

	// Pre-quoted JSON header fragments. Each carries its own delimiter
	// (open-brace or comma) and the field's key + colon, so the hot path
	// is a sequence of `buf.Write(prefix); writeValue(...)`.
	levelOpen []byte // {"level":
	dateOpen  []byte // ,"time":
	msgOpen   []byte // ,"msg":
}

// New creates a Transport from the given Config.
func New(cfg Config) *Transport {
	if cfg.MessageField == "" {
		cfg.MessageField = "msg"
	}
	if cfg.DateField == "" {
		cfg.DateField = "time"
	}
	if cfg.LevelField == "" {
		cfg.LevelField = "level"
	}
	return &Transport{
		BaseTransport: transport.NewBaseTransport(cfg.BaseConfig),
		cfg:           cfg,
		writer:        transport.WriterOrStdout(cfg.Writer),
		levelOpen:     append(append([]byte{'{'}, jsonKey(cfg.LevelField)...), ':'),
		dateOpen:      append(append([]byte{','}, jsonKey(cfg.DateField)...), ':'),
		msgOpen:       append(append([]byte{','}, jsonKey(cfg.MessageField)...), ':'),
	}
}

// jsonKey returns the JSON-encoded form of s. json.Marshal on a string never
// fails; error dropped.
func jsonKey(s string) []byte {
	b, _ := json.Marshal(s)
	return b
}

// GetLoggerInstance returns nil; structured transport has no underlying logger library.
func (s *Transport) GetLoggerInstance() any { return nil }

// SendToLogger writes one JSON object per entry: the configured level, date,
// and message fields first (in that order), followed by Data and Metadata
// entries in iteration order. Per-value marshaling uses goccy/go-json so
// structs, slices, and json.Marshaler types render exactly as encoding/json
// would.
func (s *Transport) SendToLogger(params loglayer.TransportParams) {
	if !s.ShouldProcess(params.LogLevel) {
		return
	}
	// Fold the prefix into Messages[0] for the rendered output;
	// transports own this rendering choice.
	params.Messages = transport.JoinPrefixAndMessages(params.Prefix, params.Messages)
	messages := params.Messages
	if s.cfg.MessageFn != nil {
		messages = []any{s.cfg.MessageFn(params)}
	}

	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer putBuffer(buf)

	buf.Write(s.levelOpen)
	s.writeLevel(buf, params.LogLevel)
	buf.Write(s.dateOpen)
	s.writeDate(buf)
	buf.Write(s.msgOpen)
	writeJSONString(buf, transport.JoinMessages(messages))

	for k, v := range params.Data {
		buf.WriteByte(',')
		if err := writeKeyValue(buf, k, v); err != nil {
			s.writeMarshalError(err)
			return
		}
	}
	if params.Metadata != nil {
		if key := params.Schema.MetadataFieldName; key != "" {
			// Whole metadata value nests under the configured key.
			buf.WriteByte(',')
			if err := writeKeyValue(buf, key, params.Metadata); err != nil {
				s.writeMarshalError(err)
				return
			}
		} else {
			for k, v := range transport.MetadataAsMap(params.Metadata) {
				buf.WriteByte(',')
				if err := writeKeyValue(buf, k, v); err != nil {
					s.writeMarshalError(err)
					return
				}
			}
		}
	}
	buf.WriteString("}\n")
	_, _ = s.writer.Write(buf.Bytes())
}

func (s *Transport) writeLevel(buf *bytes.Buffer, l loglayer.LogLevel) {
	if s.cfg.LevelFn != nil {
		writeJSONString(buf, s.cfg.LevelFn(l))
		return
	}
	switch l {
	case loglayer.LogLevelDebug:
		buf.Write(jsonDebug)
	case loglayer.LogLevelInfo:
		buf.Write(jsonInfo)
	case loglayer.LogLevelWarn:
		buf.Write(jsonWarn)
	case loglayer.LogLevelError:
		buf.Write(jsonError)
	case loglayer.LogLevelFatal:
		buf.Write(jsonFatal)
	default:
		writeJSONString(buf, l.String())
	}
}

// writeDate appends the timestamp as a JSON string into buf.
func (s *Transport) writeDate(buf *bytes.Buffer) {
	if s.cfg.DateFn != nil {
		writeJSONString(buf, s.cfg.DateFn())
		return
	}
	var stamp [rfc3339NanoMaxLen]byte
	formatted := time.Now().UTC().AppendFormat(stamp[:0], time.RFC3339Nano)
	buf.WriteByte('"')
	buf.Write(formatted)
	buf.WriteByte('"')
}

// writeJSONString writes s as a JSON string into buf.
// json.Marshal on a string cannot fail; error dropped.
func writeJSONString(buf *bytes.Buffer, s string) {
	b, _ := json.Marshal(s)
	buf.Write(b)
}

func writeKeyValue(buf *bytes.Buffer, key string, value any) error {
	writeJSONString(buf, key)
	buf.WriteByte(':')
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	buf.Write(b)
	return nil
}

func (s *Transport) writeMarshalError(err error) {
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer putBuffer(buf)

	buf.Write(s.levelOpen)
	buf.Write(jsonError)
	buf.Write(s.msgOpen)
	writeJSONString(buf, "loglayer: failed to marshal log entry")
	buf.WriteString(`,"error":`)
	writeJSONString(buf, err.Error())
	buf.WriteString("}\n")
	_, _ = s.writer.Write(buf.Bytes())
}
