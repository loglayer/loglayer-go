// Package cli provides a Transport tuned for command-line application
// output rather than diagnostic logging.
//
// What makes it different from the other terminal-shaped transports
// ([go.loglayer.dev/transports/console/v3], [go.loglayer.dev/transports/pretty/v3]):
//
//   - No timestamp, no log-id, no level label embedded in info/debug
//     output. The message string is printed as-is.
//   - Warn / error / fatal messages get a short cargo / eslint-style
//     prefix ("warning: ", "error: ", "fatal: ") so the urgency is
//     unambiguous when a CLI run mixes levels.
//   - Info / debug write to stdout; warn / error / fatal write to
//     stderr, matching long-standing CLI convention.
//   - ANSI color is gated by per-stream TTY detection: info / debug
//     follow stdout, warn / error / fatal follow stderr. Pipe only
//     stdout (e.g. `cli ... | less`) and severity lines stay colored.
//     Override via [Config.Color].
//   - Fields and metadata are dropped by default. CLI users don't
//     want `key=value` noise on user-facing output. Set
//     [Config.ShowFields] to append them when running with `-vv` /
//     debug verbosity.
//
// What this transport is NOT:
//
//   - A diagnostic logger. If you want timestamps and structured
//     fields, use [go.loglayer.dev/transports/console/v3] or
//     [go.loglayer.dev/transports/pretty/v3].
//   - A JSON formatter. Pair this transport with a swap to
//     [go.loglayer.dev/transports/structured/v3] when the CLI's
//     `--json` flag is set.
//
// Recommended plugin pairings:
//
//   - [go.loglayer.dev/plugins/fmtlog/v2] for fmt.Sprintf-style format
//     strings (`log.Info("Applied %d release(s) at %s:", n, sha)`).
//     CLI output almost always wants format-string semantics.
//   - [go.loglayer.dev/plugins/redact/v2] when log values may include
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

	"go.loglayer.dev/v3"
	"go.loglayer.dev/v3/transport"
	"go.loglayer.dev/v3/utils/sanitize"
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

	// MessageFn, when set, formats the entire output line. Its return
	// value replaces the message, the logfmt tail, and the table body
	// with a single user-controlled string. The level prefix (and its
	// color) still applies, so the line keeps its urgency marker.
	//
	// An empty return falls back to the normal rendering for that
	// entry, which lets a caller opt out conditionally.
	//
	// This is the full-takeover escape hatch the console transport
	// doesn't offer: there, the logfmt tail still appends after the
	// MessageFn return value.
	//
	// The return value goes through the same sanitization as messages
	// (ANSI / CRLF / bidi stripping), so a user-controlled format
	// string can't smuggle terminal escapes into the output.
	MessageFn func(params loglayer.TransportParams) string

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

	// TableColumnOrder pins the leading column order for slice-of-map
	// metadata renderings. Keys named here render in the listed order;
	// keys not in the list are sorted lexicographically and appended
	// afterward, so the knob is additive: pin only the leading columns
	// that anchor the row (e.g. an identifier column) and let the rest
	// sort. Pinned keys that don't appear in any row are silently
	// skipped. Nil / empty falls back to fully lexicographic ordering.
	TableColumnOrder []string

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
	cfg             Config
	useANSI         bool
	useANSISeverity bool
	prefix          map[loglayer.LogLevel]string
	colors          map[loglayer.LogLevel]*color.Color
	userPrefixColor *color.Color
}

// New constructs a Transport from cfg. The TTY detection for
// [ColorAuto] runs once here, per stream: info / debug / trace
// follow cfg.Stdout (or os.Stdout), warn / error / fatal / panic
// follow cfg.Stderr (or os.Stderr). Subsequent writes don't
// re-check. This keeps severity lines colored when stdout is piped
// but stderr is still a terminal (e.g. `hmn ... | less`).
//
// ColorAlways and ColorNever override both streams; there is no
// per-stream color mode.
func New(cfg Config) *Transport {
	t := &Transport{
		BaseTransport:   transport.NewBaseTransport(cfg.BaseConfig),
		cfg:             cfg,
		prefix:          defaultPrefixes(),
		colors:          defaultColors(),
		userPrefixColor: color.New(color.FgHiBlack),
	}
	maps.Copy(t.prefix, cfg.LevelPrefix)
	// Sanitize user-supplied prefixes once at construction so a
	// rebrand value loaded from env or a config file can't smuggle
	// ANSI / CRLF into the output stream.
	for level, p := range t.prefix {
		t.prefix[level] = sanitize.Message(p)
	}
	maps.Copy(t.colors, cfg.LevelColor)
	t.useANSI = resolveColor(cfg, cfg.Stdout, false)
	t.useANSISeverity = resolveColor(cfg, cfg.Stderr, true)

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
	//
	// Each level's color is toggled by the stream that level writes
	// to (severity levels go to stderr), so the two streams can
	// carry different resolutions under ColorAuto.
	for level, c := range t.colors {
		if c == nil {
			continue
		}
		cp := *c
		if t.colorOn(level) {
			cp.EnableColor()
		} else {
			cp.DisableColor()
		}
		t.colors[level] = &cp
	}
	// Same shallow-copy + per-instance flag dance for the user-
	// prefix color so a transport with ColorAlways doesn't share
	// the global NoColor with another transport. The user prefix
	// rides on the headline, which uses the level's stream color.
	upc := *t.userPrefixColor
	if t.useANSI || t.useANSISeverity {
		upc.EnableColor()
	} else {
		upc.DisableColor()
	}
	t.userPrefixColor = &upc
	return t
}

// colorOn reports whether the level's stream resolves to ANSI under
// the configured Color mode. Severity levels (warn / error / fatal /
// panic) write to stderr; the rest write to stdout.
func (t *Transport) colorOn(level loglayer.LogLevel) bool {
	switch level {
	case loglayer.LogLevelWarn, loglayer.LogLevelError, loglayer.LogLevelFatal, loglayer.LogLevelPanic:
		return t.useANSISeverity
	default:
		return t.useANSI
	}
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

// format builds the line(s) to print:
//
//	[level prefix][user prefix from WithPrefix][message] [logfmt fields]
//	[table body, if metadata is slice-of-map]
//
// Color: the level prefix and message share the level color
// (yellow / red / etc.). The user prefix gets its own dim-grey
// color so it reads as caller-context rather than urgency. Tables
// render neutral.
func (t *Transport) format(params loglayer.TransportParams) string {
	// MessageFn takes over the entire line. The level prefix and its
	// color still apply; an empty return falls back to the normal
	// rendering so the hook can opt out per entry.
	if t.cfg.MessageFn != nil {
		fnBody := sanitize.Message(t.cfg.MessageFn(params))
		if fnBody != "" {
			return t.renderHeadline(params, fnBody)
		}
	}

	msg := transport.AssembleMessage(params.Messages, sanitize.Message)

	// Append optional logfmt or capture a table.
	body := msg
	var table string
	switch {
	case isTableMetadata(params.Metadata):
		table = renderTable(asTableRows(params.Metadata), t.cfg.TableColumnOrder)
	case t.cfg.ShowFields:
		if fields := renderLogfmt(transport.MergeFieldsAndMetadata(params)); fields != "" {
			if body != "" {
				body = body + " " + fields
			} else {
				body = fields
			}
		}
	}

	// Compose the headline. Level color tints the level prefix and
	// the message body together; the user prefix gets dim-grey.
	headline := t.renderHeadline(params, body)

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

// renderHeadline composes the level prefix, the user prefix, and the
// body into a single line, applying the level's stream color decision.
func (t *Transport) renderHeadline(params loglayer.TransportParams, body string) string {
	levelPrefix := ""
	if !t.cfg.DisableLevelPrefix {
		levelPrefix = t.prefix[params.LogLevel]
	}

	userPrefix := ""
	if params.Prefix != "" {
		userPrefix = sanitize.Message(params.Prefix) + " "
	}

	var levelPart, userPart, bodyPart string
	if t.colorOn(params.LogLevel) {
		if c, ok := t.colors[params.LogLevel]; ok && c != nil {
			levelPart = c.Sprint(levelPrefix)
			bodyPart = c.Sprint(body)
		} else {
			levelPart = levelPrefix
			bodyPart = body
		}
		if userPrefix != "" {
			userPart = t.userPrefixColor.Sprint(userPrefix)
		}
	} else {
		levelPart = levelPrefix
		userPart = userPrefix
		bodyPart = body
	}
	return levelPart + userPart + bodyPart
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
// configured Color mode, against the given stream (cfg.Stdout for
// the info / debug / trace levels, cfg.Stderr for the severity
// levels). ColorAuto checks whether that stream is a TTY at
// construction time, so each stream's decision is pinned per stream.
//
// A nil stream means "the real default for that stream", which is
// what writer() would use: os.Stdout for the severity=false side,
// os.Stderr for the severity side.
func resolveColor(cfg Config, stream io.Writer, severity bool) bool {
	switch cfg.Color {
	case ColorAlways:
		return true
	case ColorNever:
		return false
	}
	if stream == nil {
		if severity {
			stream = os.Stderr
		} else {
			stream = os.Stdout
		}
	}
	if f, ok := stream.(*os.File); ok {
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
	// model as the message-side AssembleMessage.
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
// row built from the union of keys (sorted lexicographically, with any
// keys named in order pinned to the front in the listed order), then
// one row per input map. Missing values render as empty cells. Uses
// two spaces of column padding, matching the conventional CLI table
// shape (`gh`, `kubectl get`, `cargo`).
func renderTable(rows []map[string]any, order []string) string {
	keys := tableColumns(rows, order)

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

// tableColumns returns the union of keys across all rows. Keys named
// in order render first in the listed order; remaining keys sort
// lexicographically and follow. Sorted ordering for the tail is
// required so the output is deterministic regardless of the (random)
// Go map iteration order. Pinned keys absent from every row are
// silently skipped so the knob is forward-compatible with row shapes
// that don't yet exist.
func tableColumns(rows []map[string]any, order []string) []string {
	seen := make(map[string]struct{})
	for _, row := range rows {
		for k := range row {
			seen[k] = struct{}{}
		}
	}

	keys := make([]string, 0, len(seen))
	for _, k := range order {
		if _, ok := seen[k]; ok {
			keys = append(keys, k)
			delete(seen, k)
		}
	}
	rest := make([]string, 0, len(seen))
	for k := range seen {
		rest = append(rest, k)
	}
	sort.Strings(rest)
	return append(keys, rest...)
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
