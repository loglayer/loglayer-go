// Package cli provides a Transport tuned for command-line application
// output rather than diagnostic logging.
//
// What makes it different from the other terminal-shaped transports
// ([go.loglayer.dev/transports/console], [go.loglayer.dev/transports/pretty]):
//
//   - No timestamp, no log-id, no level label embedded in info/debug
//     output. The message string is printed as-is.
//   - Warn / error / fatal messages get a short cargo / eslint-style
//     prefix ("warning: ", "error: ", "fatal: ") so the urgency is
//     unambiguous when a CLI run mixes levels.
//   - Info / debug write to stdout; warn / error / fatal write to
//     stderr, matching long-standing CLI convention.
//   - ANSI color is gated by TTY detection on stdout. Pipe to a file
//     and the color disappears automatically. Override via
//     [Config.Color].
//   - Fields and metadata are dropped by default. CLI users don't
//     want `key=value` noise on user-facing output. Set
//     [Config.ShowFields] to append them when running with `-vv` /
//     debug verbosity.
//
// What this transport is NOT:
//
//   - A diagnostic logger. If you want timestamps and structured
//     fields, use [go.loglayer.dev/transports/console] or
//     [go.loglayer.dev/transports/pretty].
//   - A JSON formatter. Pair this transport with a swap to
//     [go.loglayer.dev/transports/structured] when the CLI's
//     `--json` flag is set.
//
// Recommended plugin pairings:
//
//   - [go.loglayer.dev/plugins/fmtlog] for fmt.Sprintf-style format
//     strings (`log.Info("Applied %d release(s) at %s:", n, sha)`).
//     CLI output almost always wants format-string semantics.
//   - [go.loglayer.dev/plugins/redact] when log values may include
//     tokens or other secrets that shouldn't reach stdout / stderr.
//
// See https://go.loglayer.dev for usage guides and the full API
// reference.
package cli

import (
	"fmt"
	"io"
	"maps"
	"os"
	"reflect"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
	"go.loglayer.dev/utils/sanitize"
)

// ColorMode controls ANSI color output.
type ColorMode int

const (
	// ColorAuto emits color when the configured stdout is a TTY.
	// Zero value; the typical CLI default.
	ColorAuto ColorMode = iota

	// ColorAlways emits color regardless of the output target.
	// Use for `--color=always` flags or when piping into a paginator
	// that handles ANSI.
	ColorAlways

	// ColorNever disables ANSI escapes entirely. Use for
	// `--color=never` and when writing to a log file.
	ColorNever
)

// Config holds configuration options for [Transport].
type Config struct {
	transport.BaseConfig

	// Stdout overrides os.Stdout. Info / debug entries write here.
	Stdout io.Writer

	// Stderr overrides os.Stderr. Warn / error / fatal entries
	// write here.
	Stderr io.Writer

	// Color controls ANSI color output. Zero value is [ColorAuto].
	Color ColorMode

	// ShowFields, when true, appends fields and metadata after the
	// message in `key=value` form (logfmt). Default false: CLI
	// users don't want structured noise on user-facing output.
	// Useful when wiring `-vv` / `--debug` to a verbose mode that
	// includes diagnostic context.
	ShowFields bool

	// LevelPrefix overrides the default per-level prefix map.
	// Missing entries fall back to the defaults:
	//
	//   Trace: ""
	//   Debug: "debug: "
	//   Info:  ""
	//   Warn:  "warning: "
	//   Error: "error: "
	//   Fatal: "fatal: "
	//   Panic: "panic: "
	//
	// Set an entry to "" to suppress the default prefix for that
	// level. Use a non-default string to localize or rebrand
	// (e.g. "WARN: "). Override only the levels you want to
	// change; the remaining levels keep their defaults.
	//
	// To suppress every prefix at once, set DisableLevelPrefix
	// instead of populating an empty map for every level.
	LevelPrefix map[loglayer.LogLevel]string

	// DisableLevelPrefix, when true, suppresses every per-level
	// prefix unconditionally. Set this when the host CLI already
	// renders its own urgency markers (e.g. an icon column) and
	// the transport's prefixes would be redundant.
	DisableLevelPrefix bool

	// LevelColor overrides the default per-level color map.
	// Missing entries fall back to the defaults:
	//
	//   Trace, Debug:  dim grey  (color.FgHiBlack)
	//   Info:          no color
	//   Warn:          yellow
	//   Error:         red
	//   Fatal, Panic:  bold red
	//
	// Set an entry to nil to render that level without color while
	// keeping all other defaults. Use a custom *color.Color (from
	// fatih/color) to rebrand: e.g. cyan for warn, magenta for
	// fatal. color.New is the only fatih/color symbol you need;
	// the transport shallow-copies each entry and handles the
	// per-instance flag toggling that bypasses fatih/color's
	// process-global NoColor, so a *color.Color passed here can
	// be shared safely across multiple transports.
	//
	// Color is then resolved through Config.Color (Auto / Always /
	// Never), so an override here doesn't force ANSI on a piped
	// stdout unless you also set Color: ColorAlways.
	LevelColor map[loglayer.LogLevel]*color.Color
}

// Transport renders log entries as plain CLI output.
type Transport struct {
	transport.BaseTransport
	cfg     Config
	useANSI bool
	prefix  map[loglayer.LogLevel]string
	colors  map[loglayer.LogLevel]*color.Color
}

// New constructs a Transport from cfg. The TTY detection for
// [ColorAuto] runs once here against cfg.Stdout (or os.Stdout when
// cfg.Stdout is nil); subsequent writes don't re-check.
func New(cfg Config) *Transport {
	t := &Transport{
		BaseTransport: transport.NewBaseTransport(cfg.BaseConfig),
		cfg:           cfg,
		prefix:        defaultPrefixes(),
		colors:        defaultColors(),
	}
	maps.Copy(t.prefix, cfg.LevelPrefix)
	// Sanitize user-supplied prefixes once at construction so a
	// rebrand value loaded from env or a config file can't smuggle
	// ANSI / CRLF into the output stream.
	for level, p := range t.prefix {
		t.prefix[level] = sanitize.Message(p)
	}
	maps.Copy(t.colors, cfg.LevelColor)
	t.useANSI = resolveColor(cfg)

	// fatih/color has a process-global `color.NoColor` flag that
	// the package auto-sets based on stdout TTY detection at
	// package init. Tests, piped output, and non-TTY runs flip it
	// on, which would override our per-instance ColorAlways.
	// Toggle each color's per-instance bypass to lock in our
	// resolved decision.
	//
	// Shallow-copy each *color.Color before toggling: the user may
	// have passed us a color shared with another Transport, and
	// EnableColor / DisableColor mutate per-instance state on the
	// pointer. Copying decouples the two transports' resolutions.
	for level, c := range t.colors {
		if c == nil {
			continue
		}
		cp := *c
		if t.useANSI {
			cp.EnableColor()
		} else {
			cp.DisableColor()
		}
		t.colors[level] = &cp
	}
	return t
}

// GetLoggerInstance returns nil; the cli transport has no underlying
// logger library.
func (t *Transport) GetLoggerInstance() any { return nil }

// SendToLogger implements [loglayer.Transport].
func (t *Transport) SendToLogger(params loglayer.TransportParams) {
	if !t.ShouldProcess(params.LogLevel) {
		return
	}

	body := t.format(params)
	if body == "" {
		// log.MetadataOnly with empty / nil / non-tabular metadata
		// produces no headline and no table. Skip the Fprintln so
		// CLI output isn't peppered with stray blank lines.
		return
	}
	fmt.Fprintln(t.writer(params.LogLevel), body)
}

// format builds the line(s) to print: optional level prefix +
// sanitized message + (table OR optional logfmt fields). Color is
// applied to the headline (prefix + message + logfmt) only; a table
// body, when present, renders neutral so its rows aren't tinted by
// the level color.
func (t *Transport) format(params loglayer.TransportParams) string {
	msg := transport.JoinMessages(sanitizeMessages(params.Messages))

	var head strings.Builder
	if !t.cfg.DisableLevelPrefix {
		if p := t.prefix[params.LogLevel]; p != "" {
			head.WriteString(p)
		}
	}
	head.WriteString(msg)

	var table string
	switch {
	case isTableMetadata(params.Metadata):
		table = renderTable(asTableRows(params.Metadata))
	case t.cfg.ShowFields:
		if fields := renderLogfmt(transport.MergeFieldsAndMetadata(params)); fields != "" {
			if head.Len() > 0 {
				head.WriteByte(' ')
			}
			head.WriteString(fields)
		}
	}

	headline := head.String()
	if t.useANSI {
		if c, ok := t.colors[params.LogLevel]; ok && c != nil {
			headline = c.Sprint(headline)
		}
	}

	switch {
	case table == "":
		return headline
	case headline == "":
		// MetadataOnly with table-shaped metadata: emit the table
		// alone, no leading blank line.
		return table
	default:
		return headline + "\n" + table
	}
}

// writer picks stdout vs stderr by level.
func (t *Transport) writer(level loglayer.LogLevel) io.Writer {
	switch level {
	case loglayer.LogLevelWarn, loglayer.LogLevelError, loglayer.LogLevelFatal, loglayer.LogLevelPanic:
		if t.cfg.Stderr != nil {
			return t.cfg.Stderr
		}
		return os.Stderr
	default:
		if t.cfg.Stdout != nil {
			return t.cfg.Stdout
		}
		return os.Stdout
	}
}

// resolveColor returns the static ANSI on/off decision for cfg's
// configured Color mode. ColorAuto checks whether the resolved
// stdout is a TTY at construction time.
func resolveColor(cfg Config) bool {
	switch cfg.Color {
	case ColorAlways:
		return true
	case ColorNever:
		return false
	}
	out := cfg.Stdout
	if out == nil {
		out = os.Stdout
	}
	if f, ok := out.(*os.File); ok {
		return isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
	}
	return false
}

func defaultPrefixes() map[loglayer.LogLevel]string {
	return map[loglayer.LogLevel]string{
		loglayer.LogLevelTrace: "",
		loglayer.LogLevelDebug: "debug: ",
		loglayer.LogLevelInfo:  "",
		loglayer.LogLevelWarn:  "warning: ",
		loglayer.LogLevelError: "error: ",
		loglayer.LogLevelFatal: "fatal: ",
		loglayer.LogLevelPanic: "panic: ",
	}
}

func defaultColors() map[loglayer.LogLevel]*color.Color {
	return map[loglayer.LogLevel]*color.Color{
		loglayer.LogLevelTrace: color.New(color.FgHiBlack),
		loglayer.LogLevelDebug: color.New(color.FgHiBlack),
		loglayer.LogLevelInfo:  nil, // no color: plain stdout
		loglayer.LogLevelWarn:  color.New(color.FgYellow),
		loglayer.LogLevelError: color.New(color.FgRed),
		loglayer.LogLevelFatal: color.New(color.FgRed, color.Bold),
		loglayer.LogLevelPanic: color.New(color.FgRed, color.Bold),
	}
}

// sanitizeMessages scrubs CRLF and ANSI ESC from each string-shaped
// message so a user-controlled value can't smuggle terminal escapes
// or forge log lines.
func sanitizeMessages(in []any) []any {
	out := make([]any, len(in))
	for i, m := range in {
		if s, ok := m.(string); ok {
			out[i] = sanitize.Message(s)
			continue
		}
		out[i] = m
	}
	return out
}

// renderLogfmt formats data as `key=value key=value`, sorted for
// determinism. Returns "" for empty input.
func renderLogfmt(data map[string]any) string {
	if len(data) == 0 {
		return ""
	}
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
		writeKey(&b, k)
		b.WriteByte('=')
		writeValue(&b, data[k])
	}
	return b.String()
}

func writeKey(b *strings.Builder, k string) {
	if needsQuote(k) {
		fmt.Fprintf(b, "%q", k)
		return
	}
	b.WriteString(k)
}

func writeValue(b *strings.Builder, v any) {
	// Sanitize the rendered value so an ANSI ESC or CRLF embedded
	// in a user-controlled field can't smuggle escape sequences or
	// forge log lines through the ShowFields path. Same threat
	// model as sanitizeMessages.
	s := sanitize.Message(fmt.Sprintf("%v", v))
	if needsQuote(s) {
		fmt.Fprintf(b, "%q", s)
		return
	}
	b.WriteString(s)
}

// isTableMetadata reports whether meta is a slice of map-shaped
// entries that the table renderer can consume. Bails out for
// heterogeneous slices, empty slices, or scalar values.
func isTableMetadata(meta any) bool {
	return asTableRows(meta) != nil
}

// asTableRows normalizes meta into a uniform slice of map[string]any.
// Returns nil when meta is not a slice, when the slice is empty, or
// when any element fails to convert to a map (heterogeneous slices
// or unmappable values are explicitly rejected so the caller doesn't
// get a half-rendered table). Recognized inputs:
//
//   - []map[string]any (fast path)
//   - []loglayer.Metadata (fast path; same underlying shape)
//   - []any of map-shaped or struct-shaped elements
//   - []SomeStruct or []*SomeStruct (each element JSON-roundtripped
//     via [transport.MetadataAsMap], so JSON tags are honored as
//     column headers)
func asTableRows(meta any) []map[string]any {
	if meta == nil {
		return nil
	}
	// Fast paths for the canonical map-shaped slices. Both bail
	// on nil entries so the rendering precedence matches the
	// reflection path: a single missing element drops the entire
	// table rather than producing a half-row.
	switch v := meta.(type) {
	case []map[string]any:
		if len(v) == 0 {
			return nil
		}
		for _, m := range v {
			if m == nil {
				return nil
			}
		}
		return v
	case []loglayer.Metadata:
		if len(v) == 0 {
			return nil
		}
		out := make([]map[string]any, len(v))
		for i, m := range v {
			if m == nil {
				return nil
			}
			out[i] = map[string]any(m)
		}
		return out
	}

	// Reflection fallback for []any, []SomeStruct, []*SomeStruct,
	// and any other slice shape. Each element is converted to a
	// map via the same helper transports use to flatten metadata,
	// which respects JSON struct tags.
	rv := reflect.ValueOf(meta)
	if rv.Kind() != reflect.Slice || rv.Len() == 0 {
		return nil
	}
	out := make([]map[string]any, 0, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		m := elementAsMap(rv.Index(i).Interface())
		if m == nil {
			return nil
		}
		out = append(out, m)
	}
	return out
}

// elementAsMap normalizes one slice element to a map[string]any.
// Maps pass through; structs (and pointers to structs) convert via
// the JSON roundtrip helper.
func elementAsMap(elem any) map[string]any {
	switch v := elem.(type) {
	case nil:
		return nil
	case map[string]any:
		return v
	case loglayer.Metadata:
		return map[string]any(v)
	default:
		return transport.MetadataAsMap(elem)
	}
}

// renderTable produces a tabwriter-aligned table: an uppercase header
// row built from the union of keys (sorted lexicographically), then
// one row per input map. Missing values render as empty cells. Uses
// two spaces of column padding, matching the conventional CLI table
// shape (`gh`, `kubectl get`, `cargo`).
func renderTable(rows []map[string]any) string {
	keys := tableColumns(rows)

	var b strings.Builder
	tw := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)

	headers := make([]string, len(keys))
	for i, k := range keys {
		headers[i] = strings.ToUpper(k)
	}
	fmt.Fprintln(tw, strings.Join(headers, "\t"))

	for _, row := range rows {
		cells := make([]string, len(keys))
		for i, k := range keys {
			if v, ok := row[k]; ok {
				// Sanitize cell content for the same reason as
				// writeValue: prevent ANSI / CRLF leakage from a
				// user-controlled metadata value.
				cells[i] = sanitize.Message(fmt.Sprint(v))
			}
		}
		fmt.Fprintln(tw, strings.Join(cells, "\t"))
	}
	_ = tw.Flush()

	return strings.TrimRight(b.String(), "\n")
}

// tableColumns returns the union of keys across all rows, sorted
// lexicographically. Sorted ordering is required so the output is
// deterministic regardless of the (random) Go map iteration order.
func tableColumns(rows []map[string]any) []string {
	seen := make(map[string]struct{})
	for _, row := range rows {
		for k := range row {
			seen[k] = struct{}{}
		}
	}
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func needsQuote(s string) bool {
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
