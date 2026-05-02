// Package pretty provides a colorized terminal transport that renders log
// entries with theme-aware formatting and three view modes (inline,
// message-only, expanded). Inspired by loglayer's simple-pretty-terminal.
//
// See https://go.loglayer.dev for usage guides and the full API reference.
package pretty

import (
	"fmt"
	"io"
	"strings"
	"time"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
	"go.loglayer.dev/utils/sanitize"
)

// ViewMode controls how each log entry is rendered.
type ViewMode string

const (
	// ViewModeInline renders the entry on one line: timestamp, level, message,
	// then structured data inline as `key=value`. The default.
	ViewModeInline ViewMode = "inline"

	// ViewModeMessageOnly renders only timestamp, level, and message. Structured
	// data is dropped from output.
	ViewModeMessageOnly ViewMode = "message-only"

	// ViewModeExpanded renders timestamp, level, and message on the first line,
	// then walks structured data on indented lines beneath it (YAML-like).
	ViewModeExpanded ViewMode = "expanded"
)

// Config holds configuration for the pretty transport.
type Config struct {
	transport.BaseConfig

	// ViewMode selects the rendering style. Defaults to ViewModeInline.
	ViewMode ViewMode

	// Theme overrides the default Moonlight theme. Use Sunlight, Neon, Nature,
	// Pastel, or your own *Theme. Ignored when NoColor is true.
	Theme *Theme

	// NoColor disables ANSI escape codes entirely. Useful for non-TTY output
	// (CI logs, files) and tests.
	NoColor bool

	// ShowLogID prints a short pseudo-random identifier for each log entry,
	// helpful when correlating multi-line expanded output.
	ShowLogID bool

	// TimestampFormat is a Go time format string. Defaults to "15:04:05.000".
	TimestampFormat string

	// TimestampFn overrides TimestampFormat for full control over the timestamp
	// rendering. Receives the time at which the entry was processed.
	TimestampFn func(time.Time) string

	// MaxInlineDepth limits nested data rendering in inline mode before
	// summarizing as `{...}`. Defaults to 4.
	MaxInlineDepth int

	// Writer overrides the default os.Stdout output target.
	Writer io.Writer
}

// Transport implements loglayer.Transport with colorized terminal output.
type Transport struct {
	transport.BaseTransport
	cfg   Config
	theme *Theme
	idSeq uint64
}

// New constructs a pretty Transport from the given Config.
func New(cfg Config) *Transport {
	if cfg.ViewMode == "" {
		cfg.ViewMode = ViewModeInline
	}
	if cfg.TimestampFormat == "" {
		cfg.TimestampFormat = "15:04:05.000"
	}
	if cfg.MaxInlineDepth <= 0 {
		cfg.MaxInlineDepth = 4
	}

	theme := cfg.Theme
	if theme == nil {
		theme = Moonlight()
	}
	if cfg.NoColor {
		theme = noColorTheme()
	}

	return &Transport{
		BaseTransport: transport.NewBaseTransport(cfg.BaseConfig),
		cfg:           cfg,
		theme:         theme,
	}
}

// GetLoggerInstance returns nil; pretty has no underlying logger.
func (t *Transport) GetLoggerInstance() any { return nil }

// SendToLogger implements loglayer.Transport.
func (t *Transport) SendToLogger(params loglayer.TransportParams) {
	if !t.ShouldProcess(params.LogLevel) {
		return
	}
	// Preserve the v1 "prefix folded into Messages[0]" rendering;
	// the core no longer mutates messages, transports own it now.
	params.Messages = transport.JoinPrefixAndMessages(params.Prefix, params.Messages)
	combined := combineData(params)

	timestamp := t.formatTimestamp()
	chevron := t.formatChevron(params.LogLevel)
	logID := t.formatLogID()
	// Sanitize the message before it reaches the terminal: a CRLF or
	// ANSI ESC in a user-controlled string could forge log lines or
	// smuggle terminal-coloring sequences. Field/metadata values are
	// rendered separately through the theme's Style functions; see
	// pretty's doc for the threat-model boundary (this transport is
	// for human terminals, not for log pipelines).
	message := sanitize.Message(transport.JoinMessages(params.Messages))
	if message == "" {
		message = "(no message)"
	}

	w := t.writer()

	switch t.cfg.ViewMode {
	case ViewModeMessageOnly:
		fmt.Fprintf(w, "%s %s%s\n",
			timestamp, chevron, prefixSpace(logID)+message)

	case ViewModeExpanded:
		fmt.Fprintf(w, "%s %s%s\n",
			timestamp, chevron, prefixSpace(logID)+message)
		t.renderExpanded(w, combined)

	default: // ViewModeInline
		inline := t.renderInline(combined, t.cfg.MaxInlineDepth)
		if inline != "" {
			fmt.Fprintf(w, "%s %s%s %s\n",
				timestamp, chevron, prefixSpace(logID)+message, inline)
		} else {
			fmt.Fprintf(w, "%s %s%s\n",
				timestamp, chevron, prefixSpace(logID)+message)
		}
	}
}

func (t *Transport) writer() io.Writer {
	return transport.WriterOrStdout(t.cfg.Writer)
}

func (t *Transport) formatTimestamp() string {
	now := time.Now()
	var s string
	if t.cfg.TimestampFn != nil {
		s = t.cfg.TimestampFn(now)
	} else {
		s = now.Format(t.cfg.TimestampFormat)
	}
	return t.theme.Timestamp(s)
}

func (t *Transport) formatChevron(level loglayer.LogLevel) string {
	style := t.styleForLevel(level)
	label := strings.ToUpper(level.String())
	return style(fmt.Sprintf("▶ %s ", label))
}

func (t *Transport) styleForLevel(level loglayer.LogLevel) Style {
	switch level {
	case loglayer.LogLevelDebug:
		return t.theme.Debug
	case loglayer.LogLevelInfo:
		return t.theme.Info
	case loglayer.LogLevelWarn:
		return t.theme.Warn
	case loglayer.LogLevelError:
		return t.theme.Error
	case loglayer.LogLevelFatal:
		return t.theme.Fatal
	default:
		return t.theme.Info
	}
}

func (t *Transport) formatLogID() string {
	if !t.cfg.ShowLogID {
		return ""
	}
	t.idSeq++
	id := fmt.Sprintf("[%06x]", t.idSeq&0xFFFFFF)
	return t.theme.LogID(id)
}

// prefixSpace returns "" if s is empty, otherwise s + " ". Keeps the
// timestamp/chevron prefix from getting a stray trailing space when the log ID
// is disabled.
func prefixSpace(s string) string {
	if s == "" {
		return ""
	}
	return s + " "
}
