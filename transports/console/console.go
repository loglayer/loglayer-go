// Package console provides a Transport that writes log entries to stdout/stderr
// using logfmt-style output (key=value pairs after the message).
//
// See https://go.loglayer.dev for usage guides and the full API reference.
package console

import (
	"fmt"
	"io"
	"maps"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/goccy/go-json"

	"go.loglayer.dev/v2"
	"go.loglayer.dev/v2/transport"
	"go.loglayer.dev/v2/utils/sanitize"
)

// Config holds configuration options for Transport.
type Config struct {
	transport.BaseConfig

	// MessageField places the joined message text into this key, producing a
	// single structured object as the sole log argument.
	// When set, DateField and LevelField are also included in that object.
	MessageField string

	// DateField, when set, emits a timestamp as a logfmt key alongside fields.
	// When MessageField is set, the timestamp is included in the structured object instead.
	DateField string

	// LevelField, when set, emits the log level as a logfmt key alongside fields.
	// When MessageField is set, the level is included in the structured object instead.
	LevelField string

	// DateFn overrides the default ISO-8601 timestamp when DateField is set.
	DateFn func() string

	// LevelFn overrides the default level string when LevelField is set.
	LevelFn func(loglayer.LogLevel) string

	// Stringify JSON-encodes the structured object instead of emitting logfmt.
	// Only applies when MessageField, DateField, or LevelField is set.
	Stringify bool

	// MessageFn, when set, formats the entire log output as a single string.
	// It receives the full TransportParams and its return value replaces messages.
	MessageFn func(params loglayer.TransportParams) string

	// Writer overrides the default stdout/stderr selection.
	// When set all log levels write to this writer.
	Writer io.Writer
}

// Transport writes log entries to stdout (info/debug) or stderr (warn/error/fatal).
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
	// Fold the prefix into Messages[0] for the rendered output;
	// transports own this rendering choice.
	params.Messages = transport.JoinPrefixAndMessages(params.Prefix, params.Messages)
	messages := buildMessages(params, c.cfg)
	fmt.Fprintln(c.writer(params.LogLevel), messages...)
}

// writer picks the appropriate output stream for a log level.
func (c *Transport) writer(level loglayer.LogLevel) io.Writer {
	if c.cfg.Writer != nil {
		return c.cfg.Writer
	}
	switch level {
	case loglayer.LogLevelWarn, loglayer.LogLevelError, loglayer.LogLevelFatal, loglayer.LogLevelPanic:
		return os.Stderr
	default:
		return os.Stdout
	}
}

// buildMessages assembles the argument list to pass to Fprintln. In the
// default mode, fields and metadata render as logfmt (key=value, key=value)
// after the message. MessageField and Stringify switch to single-object output
// for callers that want a structured-but-non-pipeline shape.
func buildMessages(params loglayer.TransportParams, cfg Config) []any {
	// Apply MessageFn override before Multiline-aware assembly so the
	// override produces the canonical input (a single string).
	rawMessages := params.Messages
	if cfg.MessageFn != nil {
		rawMessages = []any{cfg.MessageFn(params)}
	}

	// Per-line, Multiline-aware sanitize-and-join for the headline.
	headline := transport.AssembleMessage(rawMessages, sanitize.Message)

	combined := transport.MergeFieldsAndMetadata(params)

	// MessageField: single structured object as the sole arg.
	if cfg.MessageField != "" {
		obj := make(map[string]any, len(combined)+3)
		maps.Copy(obj, combined)
		obj[cfg.MessageField] = headline
		if cfg.DateField != "" {
			obj[cfg.DateField] = dateValue(cfg)
		}
		if cfg.LevelField != "" {
			obj[cfg.LevelField] = levelValue(cfg, params.LogLevel)
		}
		return []any{maybeStringify(obj, cfg.Stringify)}
	}

	// Bake in date/level as additional logfmt keys when configured.
	if cfg.DateField != "" || cfg.LevelField != "" {
		if combined == nil {
			combined = make(map[string]any, 2)
		}
		if cfg.DateField != "" {
			combined[cfg.DateField] = dateValue(cfg)
		}
		if cfg.LevelField != "" {
			combined[cfg.LevelField] = levelValue(cfg, params.LogLevel)
		}
		// Stringify: emit a JSON object after the message instead of logfmt.
		if cfg.Stringify {
			if headline == "" {
				return []any{maybeStringify(combined, true)}
			}
			return []any{headline, maybeStringify(combined, true)}
		}
	}

	if len(combined) > 0 {
		suffix := renderLogfmt(combined)
		if headline == "" {
			return []any{suffix}
		}
		return []any{headline, suffix}
	}
	if headline == "" {
		return nil
	}
	return []any{headline}
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

// renderLogfmt formats a map as logfmt key=value pairs separated by spaces.
// Keys are emitted in sorted order so output is stable across runs.
// Scalar values render directly (numbers, bools, time.Time as RFC3339Nano);
// strings are quoted when they contain spaces, equals signs, quotes, or
// control characters. Nested values (maps, structs, slices) are JSON-encoded
// and treated as a string.
func renderLogfmt(data map[string]any) string {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteByte(' ')
		}
		writeLogfmtString(&b, k)
		b.WriteByte('=')
		writeLogfmtValue(&b, data[k])
	}
	return b.String()
}

func writeLogfmtValue(b *strings.Builder, v any) {
	switch x := v.(type) {
	case nil:
		b.WriteString("null")
	case string:
		writeLogfmtString(b, x)
	case bool:
		if x {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	case int:
		b.WriteString(strconv.Itoa(x))
	case int8:
		b.WriteString(strconv.FormatInt(int64(x), 10))
	case int16:
		b.WriteString(strconv.FormatInt(int64(x), 10))
	case int32:
		b.WriteString(strconv.FormatInt(int64(x), 10))
	case int64:
		b.WriteString(strconv.FormatInt(x, 10))
	case uint:
		b.WriteString(strconv.FormatUint(uint64(x), 10))
	case uint8:
		b.WriteString(strconv.FormatUint(uint64(x), 10))
	case uint16:
		b.WriteString(strconv.FormatUint(uint64(x), 10))
	case uint32:
		b.WriteString(strconv.FormatUint(uint64(x), 10))
	case uint64:
		b.WriteString(strconv.FormatUint(x, 10))
	case float32:
		b.WriteString(strconv.FormatFloat(float64(x), 'g', -1, 32))
	case float64:
		b.WriteString(strconv.FormatFloat(x, 'g', -1, 64))
	case time.Time:
		writeLogfmtString(b, x.Format(time.RFC3339Nano))
	case error:
		writeLogfmtString(b, x.Error())
	case fmt.Stringer:
		writeLogfmtString(b, x.String())
	default:
		if encoded, err := json.Marshal(x); err == nil {
			writeLogfmtString(b, string(encoded))
		} else {
			writeLogfmtString(b, fmt.Sprintf("%v", x))
		}
	}
}

func writeLogfmtString(b *strings.Builder, s string) {
	if !needsLogfmtQuote(s) {
		b.WriteString(s)
		return
	}
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteByte(c)
		}
	}
	b.WriteByte('"')
}

func needsLogfmtQuote(s string) bool {
	if s == "" {
		return true
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == ' ' || c == '=' || c == '"' || c == '\\' || c < 0x20 {
			return true
		}
	}
	return false
}
