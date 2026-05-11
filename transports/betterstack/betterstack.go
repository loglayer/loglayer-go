// Package betterstack sends log entries to Better Stack's log management
// platform via their HTTP intake API.
//
// Wraps transports/http with Better Stack-specific defaults:
//   - Authorization header from Config.SourceToken
//   - Encoder that emits Better Stack's expected log shape (message, level,
//     metadata fields, dt timestamp)
//
// API reference: https://betterstack.com/docs/logs/http-api
//
// See https://go.loglayer.dev for usage guides and the full API reference.
package betterstack

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/goccy/go-json"

	httptr "go.loglayer.dev/transports/http/v2"
	"go.loglayer.dev/v2"
	"go.loglayer.dev/v2/transport"
)

// Config holds Better Stack transport configuration.
type Config struct {
	transport.BaseConfig

	// SourceToken is the Better Stack source token for authentication.
	// Required.
	//
	// Tagged json:"-" so that log.WithMetadata(cfg).Info(...) through
	// any JSON-emitting transport won't ship the token in the rendered
	// log. Direct field access by the transport's own Build() is unaffected.
	SourceToken string `json:"-"`

	// URL is the Better Stack HTTP logs intake endpoint. Defaults to
	// https://in.logs.betterstack.com. Use this for on-prem deployments
	// or testing against a mock endpoint.
	URL string

	// TimestampField is the field name for the timestamp. Defaults to "dt".
	TimestampField string

	// AllowInsecureURL permits Config.URL to use a non-https scheme.
	// The source token is sent in the Authorization header on every
	// request; without this flag, Build refuses a non-https URL to keep
	// the token off the wire in plaintext. Set true only when an on-prem
	// forwarder terminates TLS upstream and a private network carries
	// the cleartext hop.
	AllowInsecureURL bool

	// HTTP overrides batching, client, error handling, and any other
	// transports/http settings. The URL, Encoder, and Authorization header
	// are set by this package and cannot be overridden via this field.
	HTTP httptr.Config
}

// String returns a redacted form of the config so that an accidental
// log.Info(cfg) (or fmt.Sprintf("%v", cfg)) can't ship the source token.
func (c Config) String() string {
	masked := c
	if masked.SourceToken != "" {
		masked.SourceToken = "***redacted***"
	}
	return fmt.Sprintf(
		"betterstack.Config{SourceToken:%q URL:%q TimestampField:%q}",
		masked.SourceToken, masked.URL, masked.TimestampField,
	)
}

// Transport wraps a transports/http.Transport with Better Stack-specific
// encoding and defaults.
type Transport struct {
	*httptr.Transport
}

// New constructs a Better Stack Transport. Panics if Config.SourceToken is empty.
func New(cfg Config) *Transport {
	t, err := Build(cfg)
	if err != nil {
		panic(err)
	}
	return t
}

// Build constructs a Better Stack Transport like New but returns
// ErrSourceTokenRequired instead of panicking when cfg.SourceToken is empty.
func Build(cfg Config) (*Transport, error) {
	if cfg.SourceToken == "" {
		return nil, ErrSourceTokenRequired(cfg.SourceToken)
	}
	if cfg.HTTP.URL != "" || cfg.HTTP.Encoder != nil {
		return nil, fmt.Errorf("betterstack: HTTP.URL and HTTP.Encoder cannot be overridden")
	}

	httpCfg := cfg.HTTP
	httpCfg.BaseConfig = cfg.BaseConfig

	urlStr := cfg.URL
	if urlStr == "" {
		urlStr = "https://in.logs.betterstack.com"
	}

	if !cfg.AllowInsecureURL {
		u, err := url.Parse(urlStr)
		if err != nil || !strings.EqualFold(u.Scheme, "https") {
			return nil, fmt.Errorf("betterstack: URL must be https when AllowInsecureURL is false")
		}
	}
	httpCfg.URL = urlStr

	timestampField := cfg.TimestampField
	if timestampField == "" {
		timestampField = "dt"
	}
	httpCfg.Encoder = newEncoder(timestampField)

	merged := make(map[string]string, len(httpCfg.Headers)+2)
	for k, v := range httpCfg.Headers {
		if strings.EqualFold(k, "authorization") || strings.EqualFold(k, "content-type") {
			continue // betterstack sets these; ignore any user-provided values
		}
		merged[k] = v
	}
	merged["Authorization"] = fmt.Sprintf("Bearer %s", cfg.SourceToken)
	merged["Content-Type"] = "application/json"
	httpCfg.Headers = merged

	httpT, err := httptr.Build(httpCfg)
	if err != nil {
		return nil, err
	}
	return &Transport{Transport: httpT}, nil
}

// newEncoder produces the JSON-array encoder for Better Stack's intake format.
func newEncoder(timestampField string) httptr.Encoder {
	return httptr.EncoderFunc(func(entries []httptr.Entry) ([]byte, string, error) {
		objs := make([]map[string]any, len(entries))
		for i, e := range entries {
			obj := make(map[string]any, 3+len(e.Data))
			obj["message"] = transport.JoinMessages(e.Messages)
			obj[timestampField] = e.Time.UTC().Format("2006-01-02T15:04:05.000Z")

			levelStr := statusFor(e.Level)
			if levelStr != "" {
				obj["level"] = levelStr
			}

			transport.MergeIntoMap(obj, e.Data, e.Metadata, e.Schema.MetadataFieldName)
			objs[i] = obj
		}
		body, err := json.Marshal(objs)
		return body, "application/json", err
	})
}

// statusFor maps a loglayer LogLevel to Better Stack's level string.
func statusFor(l loglayer.LogLevel) string {
	switch l {
	case loglayer.LogLevelTrace:
		return "trace"
	case loglayer.LogLevelDebug:
		return "debug"
	case loglayer.LogLevelInfo:
		return "info"
	case loglayer.LogLevelWarn:
		return "warn"
	case loglayer.LogLevelError:
		return "error"
	case loglayer.LogLevelFatal:
		return "fatal"
	case loglayer.LogLevelPanic:
		return "panic"
	default:
		return ""
	}
}
