// Package datadog sends log entries to the Datadog Logs HTTP intake API.
//
// Wraps transports/http with Datadog-specific defaults:
//   - Site-aware intake URL (us1, us3, us5, eu1, ap1)
//   - DD-API-KEY header from Config.APIKey
//   - Encoder that emits Datadog's expected log shape (ddsource, service,
//     hostname, ddtags, status, message, date) merged with the user's fields
//     and metadata
//
// API reference: https://docs.datadoghq.com/api/latest/logs/#send-logs
package datadog

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/goccy/go-json"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
	httptr "go.loglayer.dev/transports/http"
)

// Site identifies the Datadog region. Affects only the intake URL.
type Site string

const (
	SiteUS1 Site = "us1" // datadoghq.com (default)
	SiteUS3 Site = "us3" // us3.datadoghq.com
	SiteUS5 Site = "us5" // us5.datadoghq.com
	SiteEU  Site = "eu1" // datadoghq.eu
	SiteAP1 Site = "ap1" // ap1.datadoghq.com
)

// IntakeURL returns the HTTP logs intake endpoint for the site. An unknown
// or empty site falls back to SiteUS1.
func (s Site) IntakeURL() string {
	switch s {
	case SiteUS3:
		return "https://http-intake.logs.us3.datadoghq.com/api/v2/logs"
	case SiteUS5:
		return "https://http-intake.logs.us5.datadoghq.com/api/v2/logs"
	case SiteEU:
		return "https://http-intake.logs.datadoghq.eu/api/v2/logs"
	case SiteAP1:
		return "https://http-intake.logs.ap1.datadoghq.com/api/v2/logs"
	default:
		return "https://http-intake.logs.datadoghq.com/api/v2/logs"
	}
}

// Config holds Datadog transport configuration.
type Config struct {
	transport.BaseConfig

	// APIKey is the Datadog API key. Required.
	//
	// Tagged json:"-" so that log.WithMetadata(cfg).Info(...) through
	// any JSON-emitting transport (structured, zerolog, zap, slog,
	// etc.) won't ship the key in the rendered log. Direct field
	// access by the transport's own Build() is unaffected.
	APIKey string `json:"-"`

	// Site selects the Datadog region. Defaults to SiteUS1. Ignored
	// when URL is set.
	Site Site

	// URL overrides the Site-derived intake URL. Use it for on-prem
	// Datadog deployments or for testing against a mock endpoint.
	// When set, Site is ignored. Must be a full HTTPS URL ending at
	// the logs intake path (Datadog SaaS shape:
	// https://http-intake.logs.<site>/api/v2/logs).
	URL string

	// Source maps to the ddsource field. Conventionally a short string
	// identifying the producing technology, e.g. "go". Optional.
	Source string

	// Service maps to the service field. The application or service name.
	// Optional.
	Service string

	// Hostname maps to the hostname field. Optional.
	Hostname string

	// Tags maps to the ddtags field as a single string. Conventionally
	// comma-separated key:value pairs, e.g. "env:prod,team:platform".
	// Optional.
	Tags string

	// AllowInsecureURL permits Config.URL to use a non-https scheme. The
	// API key is sent in the DD-API-KEY header on every request; without
	// this flag, Build refuses a non-https URL to keep the key off the
	// wire in plaintext. Set true only when an on-prem forwarder
	// terminates TLS upstream and a private network carries the cleartext
	// hop. The Site-derived intake URLs are always https and unaffected.
	AllowInsecureURL bool

	// HTTP overrides batching, client, error handling, and any other
	// transports/http settings. The URL, Encoder, and DD-API-KEY header are
	// set by this package and cannot be overridden via this field.
	HTTP httptr.Config
}

// String returns a redacted form of the config so that an accidental
// log.Info(cfg) (or fmt.Sprintf("%v", cfg)) can't ship the API key.
// The key is replaced with a fixed mask regardless of length.
//
// Note: Go's fmt verbs %+v and %#v intentionally bypass Stringer and
// always print struct fields. Code that uses those verbs against
// Config will see the raw APIKey. Reserve %+v / %#v for debugger-style
// inspection, never for production logs. The json:"-" tag on APIKey
// prevents the JSON-via-transport leak path; this method covers the
// fmt.Sprintf path; %+v / %#v are explicitly out of scope.
func (c Config) String() string {
	masked := c
	if masked.APIKey != "" {
		masked.APIKey = "***redacted***"
	}
	// Spell out the fields explicitly rather than %+v on `masked` so a
	// future field addition doesn't silently expose new sensitive
	// content. Keep the order matching the struct for readability.
	return fmt.Sprintf(
		"datadog.Config{APIKey:%q Site:%q URL:%q Source:%q Service:%q Hostname:%q Tags:%q}",
		masked.APIKey, masked.Site, masked.URL, masked.Source,
		masked.Service, masked.Hostname, masked.Tags,
	)
}

// Transport wraps a transports/http.Transport with Datadog-specific encoding
// and defaults.
type Transport struct {
	*httptr.Transport
}

// New constructs a Datadog Transport. Panics if Config.APIKey is empty.
// Use Build for an error-returning variant.
func New(cfg Config) *Transport {
	t, err := Build(cfg)
	if err != nil {
		panic(err)
	}
	return t
}

// Build constructs a Datadog Transport like New but returns
// ErrAPIKeyRequired instead of panicking when cfg.APIKey is empty. Use
// this when the API key is loaded at runtime (e.g. from an environment
// variable) and you want to handle the missing-config case explicitly.
func Build(cfg Config) (*Transport, error) {
	if cfg.APIKey == "" {
		return nil, ErrAPIKeyRequired
	}
	if cfg.HTTP.URL != "" || cfg.HTTP.Encoder != nil {
		return nil, ErrHTTPOverrideForbidden
	}

	httpCfg := cfg.HTTP
	httpCfg.BaseConfig = cfg.BaseConfig
	if cfg.URL != "" {
		if !cfg.AllowInsecureURL {
			u, err := url.Parse(cfg.URL)
			if err != nil || !strings.EqualFold(u.Scheme, "https") {
				return nil, ErrInsecureURL
			}
		}
		httpCfg.URL = cfg.URL
	} else {
		httpCfg.URL = cfg.Site.IntakeURL()
	}
	httpCfg.Encoder = newEncoder(cfg)

	// Clone Headers so we don't mutate the caller's map by adding DD-API-KEY.
	merged := make(map[string]string, len(cfg.HTTP.Headers)+1)
	for k, v := range cfg.HTTP.Headers {
		merged[k] = v
	}
	merged["DD-API-KEY"] = cfg.APIKey
	httpCfg.Headers = merged

	httpT, err := httptr.Build(httpCfg)
	if err != nil {
		return nil, err
	}
	return &Transport{Transport: httpT}, nil
}

// newEncoder produces the JSON-array encoder for Datadog's intake format.
// Each entry merges the configured ddsource/service/hostname/ddtags with the
// per-call message, status, and timestamp, plus any user fields and metadata.
func newEncoder(cfg Config) httptr.Encoder {
	return httptr.EncoderFunc(func(entries []httptr.Entry) ([]byte, string, error) {
		objs := make([]map[string]any, len(entries))
		for i, e := range entries {
			obj := make(map[string]any, 7+len(e.Data))
			obj["message"] = transport.JoinMessages(e.Messages)
			obj["status"] = statusFor(e.Level)
			obj["date"] = e.Time.UTC().Format("2006-01-02T15:04:05.000Z")
			if cfg.Source != "" {
				obj["ddsource"] = cfg.Source
			}
			if cfg.Service != "" {
				obj["service"] = cfg.Service
			}
			if cfg.Hostname != "" {
				obj["hostname"] = cfg.Hostname
			}
			if cfg.Tags != "" {
				obj["ddtags"] = cfg.Tags
			}
			transport.MergeIntoMap(obj, e.Data, e.Metadata)
			objs[i] = obj
		}
		body, err := json.Marshal(objs)
		return body, "application/json", err
	})
}

// statusFor maps a loglayer LogLevel to Datadog's status string.
// See https://docs.datadoghq.com/logs/log_collection/#reserved-attributes.
func statusFor(l loglayer.LogLevel) string {
	switch l {
	case loglayer.LogLevelDebug:
		return "debug"
	case loglayer.LogLevelInfo:
		return "info"
	case loglayer.LogLevelWarn:
		return "warning"
	case loglayer.LogLevelError:
		return "error"
	case loglayer.LogLevelFatal:
		return "critical"
	default:
		return "info"
	}
}
