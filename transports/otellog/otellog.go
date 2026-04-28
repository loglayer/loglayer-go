// Package otellog provides a LogLayer transport that emits log entries
// to an OpenTelemetry log.Logger.
//
// Use this transport when your service is wired against the OpenTelemetry
// SDK and you want LogLayer's API to feed entries into the same OTel
// pipeline as your traces and metrics. The transport produces a
// log.Record per emission with severity, body, attributes, and the
// caller's context.Context (so OTel SDK processors can correlate the
// entry with the active span automatically).
//
// The package name is otellog (not otel) to avoid a name collision with
// go.opentelemetry.io/otel. Import path remains
// go.loglayer.dev/transports/otellog.
//
// Wiring with the global LoggerProvider (most common):
//
//	import (
//	    "go.loglayer.dev"
//	    "go.loglayer.dev/transports/otellog"
//	)
//
//	tr := otellog.New(otellog.Config{Name: "checkout-api"})
//	log := loglayer.New(loglayer.Config{Transport: tr})
//
// Wiring with an explicit LoggerProvider (e.g. SDK-constructed):
//
//	import (
//	    sdklog "go.opentelemetry.io/otel/sdk/log"
//	)
//
//	provider := sdklog.NewLoggerProvider(sdklog.WithProcessor(processor))
//	tr := otellog.New(otellog.Config{
//	    Name:           "checkout-api",
//	    Version:        "1.2.3",
//	    LoggerProvider: provider,
//	})
package otellog

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
)

// ErrNameRequired is returned by Build when Config.Logger is nil and
// Config.Name is empty. The instrumentation scope name is required by
// the OpenTelemetry log API to construct a Logger from a LoggerProvider.
var ErrNameRequired = errors.New("loglayer/transports/otellog: Config.Name is required when Logger is nil")

// Config holds configuration options for the otellog transport.
type Config struct {
	transport.BaseConfig

	// Name is the instrumentation scope name passed to
	// LoggerProvider.Logger. Conventionally the importable name of the
	// calling package or the service identifier (e.g. "checkout-api").
	// Required when Logger is nil; ignored when Logger is set.
	Name string

	// Version is the optional instrumentation scope version. Surfaced
	// to backends as the scope's version attribute.
	Version string

	// SchemaURL is the optional schema URL for the instrumentation scope.
	SchemaURL string

	// LoggerProvider is the source of the OTel log.Logger. When nil the
	// process-wide global provider (global.GetLoggerProvider) is used.
	// If no global has been registered, OTel returns a no-op provider:
	// safe, but emits nothing.
	LoggerProvider otellog.LoggerProvider

	// Logger is an already-constructed log.Logger to write to. When set,
	// takes precedence over LoggerProvider/Name/Version/SchemaURL. Useful
	// for tests and when callers want full control over Logger options.
	Logger otellog.Logger

	// MetadataFieldName is the attribute key under which non-map
	// metadata values (structs, scalars, slices) are emitted. Map
	// metadata is always merged at the root as individual attributes.
	// Defaults to "metadata".
	MetadataFieldName string
}

// Transport sends log entries to an OpenTelemetry log.Logger.
type Transport struct {
	transport.BaseTransport
	cfg    Config
	logger otellog.Logger
}

// New creates an otellog Transport from the given Config. Panics on
// misconfiguration (see Build for the error-returning variant).
func New(cfg Config) *Transport {
	t, err := Build(cfg)
	if err != nil {
		panic(err)
	}
	return t
}

// Build creates an otellog Transport from the given Config. Returns
// ErrNameRequired if Config.Logger is nil and Config.Name is empty. Use
// Build when the LoggerProvider is constructed at runtime (e.g. with
// configuration loaded from environment variables) so the failure mode
// is recoverable.
func Build(cfg Config) (*Transport, error) {
	if cfg.MetadataFieldName == "" {
		cfg.MetadataFieldName = "metadata"
	}
	logger := cfg.Logger
	if logger == nil {
		if cfg.Name == "" {
			return nil, ErrNameRequired
		}
		provider := cfg.LoggerProvider
		if provider == nil {
			provider = global.GetLoggerProvider()
		}
		var opts []otellog.LoggerOption
		if cfg.Version != "" {
			opts = append(opts, otellog.WithInstrumentationVersion(cfg.Version))
		}
		if cfg.SchemaURL != "" {
			opts = append(opts, otellog.WithSchemaURL(cfg.SchemaURL))
		}
		logger = provider.Logger(cfg.Name, opts...)
	}
	return &Transport{
		BaseTransport: transport.NewBaseTransport(cfg.BaseConfig),
		cfg:           cfg,
		logger:        logger,
	}, nil
}

// GetLoggerInstance returns the underlying log.Logger.
func (t *Transport) GetLoggerInstance() any { return t.logger }

// SendToLogger implements loglayer.Transport.
//
// Builds a log.Record (timestamp = time.Now, severity from the log
// level, body = joined messages, attributes from data + metadata) and
// emits it. The caller's context.Context (from WithCtx) is forwarded so
// OTel SDK processors can correlate the entry with the active span.
//
// Fatal entries are emitted at SeverityFatal; the core's
// Config.DisableFatalExit decides whether os.Exit follows.
func (t *Transport) SendToLogger(params loglayer.TransportParams) {
	if !t.ShouldProcess(params.LogLevel) {
		return
	}

	var rec otellog.Record
	rec.SetTimestamp(time.Now())
	rec.SetSeverity(toOtelSeverity(params.LogLevel))
	rec.SetSeverityText(params.LogLevel.String())
	rec.SetBody(otellog.StringValue(transport.JoinMessages(params.Messages)))

	if n := transport.FieldEstimate(params); n > 0 {
		attrs := make([]otellog.KeyValue, 0, n)
		for k, v := range params.Data {
			attrs = append(attrs, otellog.KeyValue{Key: k, Value: toValue(v)})
		}
		if m, ok := transport.MetadataAsRootMap(params.Metadata); ok {
			for k, v := range m {
				attrs = append(attrs, otellog.KeyValue{Key: k, Value: toValue(v)})
			}
		} else if m := transport.MetadataAsMap(params.Metadata); m != nil {
			// JSON-roundtrip non-map metadata so structs land as a
			// nested MapValue. Metadata that JSON can't marshal
			// (channels, funcs, etc.) is dropped: fmt.Sprintf-stringifying
			// it would lose the structure the user intended anyway.
			attrs = append(attrs, otellog.KeyValue{
				Key:   t.cfg.MetadataFieldName,
				Value: toValue(m),
			})
		}
		rec.AddAttributes(attrs...)
	}

	ctx := params.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	t.logger.Emit(ctx, rec)
}

// toOtelSeverity maps loglayer levels onto the OTel severity scale.
// OTel defines four sub-levels per severity bucket (Debug1-4, Info1-4,
// etc.); we use the first of each bucket since loglayer has a single
// level per bucket.
func toOtelSeverity(l loglayer.LogLevel) otellog.Severity {
	switch l {
	case loglayer.LogLevelDebug:
		return otellog.SeverityDebug
	case loglayer.LogLevelInfo:
		return otellog.SeverityInfo
	case loglayer.LogLevelWarn:
		return otellog.SeverityWarn
	case loglayer.LogLevelError:
		return otellog.SeverityError
	case loglayer.LogLevelFatal:
		return otellog.SeverityFatal
	default:
		return otellog.SeverityInfo
	}
}

// toValue converts a Go value to an OTel log.Value, recursing into
// nested maps and slices. Unknown types fall back to fmt.Sprintf("%v").
func toValue(v any) otellog.Value {
	switch x := v.(type) {
	case nil:
		return otellog.Value{}
	case string:
		return otellog.StringValue(x)
	case bool:
		return otellog.BoolValue(x)
	case int:
		return otellog.IntValue(x)
	case int8:
		return otellog.Int64Value(int64(x))
	case int16:
		return otellog.Int64Value(int64(x))
	case int32:
		return otellog.Int64Value(int64(x))
	case int64:
		return otellog.Int64Value(x)
	case uint:
		return otellog.Int64Value(int64(x))
	case uint8:
		return otellog.Int64Value(int64(x))
	case uint16:
		return otellog.Int64Value(int64(x))
	case uint32:
		return otellog.Int64Value(int64(x))
	case uint64:
		// uint64 above MaxInt64 silently truncates; the alternative is
		// stringifying, which loses arithmetic affordances on the
		// backend. Match log/slog's behavior here.
		return otellog.Int64Value(int64(x))
	case float32:
		return otellog.Float64Value(float64(x))
	case float64:
		return otellog.Float64Value(x)
	case []byte:
		return otellog.BytesValue(x)
	case time.Time:
		return otellog.StringValue(x.Format(time.RFC3339Nano))
	case error:
		return otellog.StringValue(x.Error())
	case fmt.Stringer:
		return otellog.StringValue(x.String())
	case map[string]any:
		kvs := make([]otellog.KeyValue, 0, len(x))
		for k, vv := range x {
			kvs = append(kvs, otellog.KeyValue{Key: k, Value: toValue(vv)})
		}
		return otellog.MapValue(kvs...)
	case []any:
		vs := make([]otellog.Value, len(x))
		for i, vv := range x {
			vs[i] = toValue(vv)
		}
		return otellog.SliceValue(vs...)
	default:
		return otellog.StringValue(fmt.Sprintf("%v", v))
	}
}
