// Package sentrytransport provides a LogLayer transport backed by a
// caller-supplied [sentry.Logger]. The package directory is
// `transports/sentry`; the package name is `sentrytransport` to avoid
// colliding with the imported `sentry` identifier.
//
// The user owns Sentry initialization; this transport just forwards
// each entry to the supplied logger via the chain-builder API
// (Trace/Debug/Info/Warn/Error/LFatal, attribute methods, then Emit).
//
// Fatal-level events are routed through [sentry.Logger.LFatal] rather
// than [sentry.Logger.Fatal], so loglayer's core decides whether to
// call os.Exit (per Config.DisableFatalExit). The Panic level maps to
// LFatal as well; loglayer's core does the panic separately.
//
// See https://go.loglayer.dev for usage guides and the full API reference.
package sentrytransport

import (
	"fmt"

	"github.com/getsentry/sentry-go"
	"github.com/goccy/go-json"

	"go.loglayer.dev/v2"
	"go.loglayer.dev/v2/transport"
)

// Config holds configuration options for the Sentry transport.
type Config struct {
	transport.BaseConfig

	// Logger is the Sentry logger to forward entries to. Required.
	// Typically obtained via sentry.NewLogger(ctx) after sentry.Init.
	Logger sentry.Logger
}

// Transport sends log entries to a sentry.Logger.
type Transport struct {
	transport.BaseTransport
	cfg Config
}

// New constructs a Sentry Transport. Panics if Config.Logger is nil.
// Use Build for an error-returning variant.
func New(cfg Config) *Transport {
	t, err := Build(cfg)
	if err != nil {
		panic(err)
	}
	return t
}

// Build constructs a Sentry Transport like New but returns
// ErrLoggerRequired instead of panicking when cfg.Logger is nil. Use
// this when the logger is loaded at runtime and you want to handle the
// missing-config case explicitly.
func Build(cfg Config) (*Transport, error) {
	if cfg.Logger == nil {
		return nil, ErrLoggerRequired
	}
	return &Transport{
		BaseTransport: transport.NewBaseTransport(cfg.BaseConfig),
		cfg:           cfg,
	}, nil
}

// GetLoggerInstance returns the underlying sentry.Logger so callers can
// use it directly when they need Sentry-specific features.
func (t *Transport) GetLoggerInstance() any { return t.cfg.Logger }

// SendToLogger implements loglayer.Transport.
func (t *Transport) SendToLogger(params loglayer.TransportParams) {
	if !t.ShouldProcess(params.LogLevel) {
		return
	}
	// Fold the prefix into Messages[0] for the rendered output;
	// transports own this rendering choice.
	params.Messages = transport.JoinPrefixAndMessages(params.Prefix, params.Messages)

	entry := entryForLevel(t.cfg.Logger, params.LogLevel)
	if params.Ctx != nil {
		entry = entry.WithCtx(params.Ctx)
	}

	for k, v := range params.Data {
		entry = setAttr(entry, k, v)
	}

	if params.Metadata != nil {
		if key := params.Schema.MetadataFieldName; key != "" {
			// Whole metadata value nests under the key as a JSON-encoded
			// string. Sentry's LogEntry has no Map / Any setter, so a
			// typed forwarding pass through the chain isn't possible for
			// a non-scalar value at a single key.
			entry = entry.String(key, jsonString(params.Metadata))
		} else if m, ok := transport.MetadataAsRootMap(params.Metadata); ok {
			for k, v := range m {
				entry = setAttr(entry, k, v)
			}
		} else {
			entry = entry.String("metadata", jsonString(params.Metadata))
		}
	}

	entry.Emit(transport.JoinMessages(params.Messages))
}

// entryForLevel maps a loglayer level to the matching Sentry log-entry
// builder. Fatal and Panic both route to LFatal so the Sentry SDK
// doesn't call os.Exit / panic itself; loglayer's core handles that
// (subject to Config.DisableFatalExit).
func entryForLevel(l sentry.Logger, level loglayer.LogLevel) sentry.LogEntry {
	switch level {
	case loglayer.LogLevelTrace:
		return l.Trace()
	case loglayer.LogLevelDebug:
		return l.Debug()
	case loglayer.LogLevelInfo:
		return l.Info()
	case loglayer.LogLevelWarn:
		return l.Warn()
	case loglayer.LogLevelError:
		return l.Error()
	case loglayer.LogLevelFatal, loglayer.LogLevelPanic:
		return l.LFatal()
	default:
		return l.Info()
	}
}

// setAttr attaches one key/value to a Sentry log entry, dispatching by
// Go type to the corresponding typed-attribute method. Values that
// don't match a typed setter (structs, nested maps, mixed-type slices,
// etc.) are rendered as JSON strings so the structure is preserved in
// Sentry's UI rather than the Go-default fmt.Sprintf form.
func setAttr(e sentry.LogEntry, key string, value any) sentry.LogEntry {
	switch v := value.(type) {
	case nil:
		return e.String(key, "")
	case string:
		return e.String(key, v)
	case bool:
		return e.Bool(key, v)
	case int:
		return e.Int(key, v)
	case int64:
		return e.Int64(key, v)
	case float64:
		return e.Float64(key, v)
	case []string:
		return e.StringSlice(key, v)
	case []int64:
		return e.Int64Slice(key, v)
	case []float64:
		return e.Float64Slice(key, v)
	case []bool:
		return e.BoolSlice(key, v)
	default:
		return e.String(key, jsonString(v))
	}
}

// jsonString renders v as a JSON-encoded string for inclusion in a
// Sentry attribute. Falls back to fmt.Sprintf("%v", v) if the value
// can't be marshalled (cyclic structs, unsupported types like channels
// or funcs) so no value is silently dropped.
func jsonString(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}
