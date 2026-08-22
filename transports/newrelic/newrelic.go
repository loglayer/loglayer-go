// Package newrelic sends log entries to the New Relic Logs HTTP intake API.
//
// Wraps transports/http with New Relic-specific defaults:
//   - Site-aware intake URL (US, EU)
//   - Api-Key header from Config.LicenseKey
//   - Encoder that emits the New Relic log shape (timestamp, level, log,
//     attributes) with attribute validation enforced at encode time
//
// API reference:
// https://docs.newrelic.com/docs/logs/log-api/introduction-log-api/
//
// See https://go.loglayer.dev for usage guides and the full API reference.
package newrelic

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/goccy/go-json"

	httptr "go.loglayer.dev/transports/http/v3"
	"go.loglayer.dev/v3"
	"go.loglayer.dev/v3/transport"
)

// Site identifies the New Relic region. Affects only the intake URL.
type Site string

const (
	SiteUS Site = "US" // log-api.newrelic.com (default)
	SiteEU Site = "EU" // log-api.eu.newrelic.com
)

// IntakeURL returns the New Relic logs intake endpoint for the site. An
// unknown or empty site falls back to SiteUS.
func (s Site) IntakeURL() string {
	switch s {
	case SiteEU:
		return "https://log-api.eu.newrelic.com/log/v1"
	default:
		return "https://log-api.newrelic.com/log/v1"
	}
}

// Config holds New Relic transport configuration.
type Config struct {
	transport.BaseConfig

	// LicenseKey is the New Relic license key or user key. Required.
	//
	// Tagged json:"-" so that log.WithMetadata(cfg).Info(...) through
	// any JSON-emitting transport (structured, zerolog, zap, slog,
	// etc.) won't ship the key in the rendered log. Direct field
	// access by the transport's own Build() is unaffected.
	LicenseKey string `json:"-"`

	// Site selects the New Relic region. Defaults to SiteUS. Ignored
	// when URL is set.
	Site Site

	// URL overrides the Site-derived intake URL. Use it for on-prem
	// deployments or for testing against a mock endpoint. When set,
	// Site is ignored.
	URL string

	// AllowInsecureURL permits Config.URL to use a non-https scheme. The
	// license key is sent in the Api-Key header on every request; without
	// this flag, Build refuses a non-https URL to keep the key off the
	// wire in plaintext. Set true only when a TLS-terminating proxy
	// or private network carries the cleartext hop. The Site-derived
	// intake URLs are always https and unaffected.
	AllowInsecureURL bool

	// HTTP overrides batching, client, error handling, and any other
	// transports/http settings. URL and Encoder are managed by this
	// package and cannot be overridden via this field.
	HTTP httptr.Config
}

// String returns a redacted form of the config so that an accidental
// log.Info(cfg) or fmt.Sprintf("%v", cfg) can't ship the license key.
//
// Note: Go's fmt verbs %+v and %#v bypass the Stringer interface;
// the json:"-" tag on LicenseKey prevents the JSON-via-transport path.
func (c Config) String() string {
	masked := c
	if masked.LicenseKey != "" {
		masked.LicenseKey = "***redacted***"
	}
	return fmt.Sprintf(
		"newrelic.Config{LicenseKey:%q Site:%q URL:%q}",
		masked.LicenseKey, masked.Site, masked.URL,
	)
}

// Transport wraps a transports/http.Transport with New Relic-specific
// encoding and defaults.
type Transport struct {
	*httptr.Transport
}

// New constructs a New Relic Transport. Panics if Config.LicenseKey is
// empty. Use Build for an error-returning variant.
func New(cfg Config) *Transport {
	t, err := Build(cfg)
	if err != nil {
		panic(err)
	}
	return t
}

// Build constructs a New Relic Transport like New but returns
// ErrLicenseKeyRequired instead of panicking when cfg.LicenseKey is empty.
// Use this when the license key is loaded at runtime (e.g. from an
// environment variable) and you want to handle the missing-config case
// explicitly.
func Build(cfg Config) (*Transport, error) {
	if cfg.LicenseKey == "" {
		return nil, ErrLicenseKeyRequired
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
	httpCfg.Encoder = newEncoder()

	// Clone Headers so we don't mutate the caller's map by adding Api-Key.
	merged := make(map[string]string, len(cfg.HTTP.Headers)+1)
	for k, v := range cfg.HTTP.Headers {
		merged[k] = v
	}
	merged["Api-Key"] = cfg.LicenseKey
	httpCfg.Headers = merged

	httpT, err := httptr.Build(httpCfg)
	if err != nil {
		return nil, err
	}
	return &Transport{Transport: httpT}, nil
}

// newEncoder produces the JSON-array encoder for New Relic's intake format.
// Each entry is a JSON object with timestamp, level, log, and attributes
// (merged data + metadata), matching the TypeScript transport format.
func newEncoder() httptr.Encoder {
	return httptr.EncoderFunc(func(entries []httptr.Entry) ([]byte, string, error) {
		objs := make([]map[string]any, len(entries))
		for i, e := range entries {
			obj := map[string]any{
				"timestamp": e.Time.UnixMilli(),
				"level":     loglevelFor(e.Level),
				"log":       transport.JoinMessages(e.Messages),
			}
			attrs := mergeAttributes(e)
			if len(attrs) > 0 {
				obj["attributes"] = attrs
			}
			objs[i] = obj
		}
		body, err := json.Marshal(objs)
		return body, "application/json", err
	})
}

const (
	maxAttributes           = 255
	maxAttributeNameLength  = 255
	maxAttributeValueLength = 4094
)

// mergeAttributes merges entry data and metadata into a single attributes
// map, enforcing New Relic's API constraints: max 255 attributes, max 255-
// char attribute names, and values truncated at 4094 chars. Reserved fields
// (timestamp, level, log) are excluded to prevent collisions.
func mergeAttributes(e httptr.Entry) map[string]any {
	attrs := make(map[string]any)
	for k, v := range e.Data {
		if reserved(k) {
			continue
		}
		if len(attrs)+1 > maxAttributes {
			break
		}
		setAttr(attrs, k, v)
	}

	if m, ok := transport.MetadataAsRootMap(e.Metadata); ok {
		for k, v := range m {
			if reserved(k) {
				continue
			}
			if len(attrs)+1 > maxAttributes {
				break
			}
			setAttr(attrs, k, v)
		}
	}

	return attrs
}

// reserved returns true if k is a top-level New Relic log field that should
// not appear inside the attributes map.
func reserved(k string) bool {
	switch k {
	case "timestamp", "level", "log":
		return true
	}
	return false
}

// setAttr validates and sets a single attribute, truncating string values
// that exceed the New Relic limit.
func setAttr(attrs map[string]any, key string, val any) {
	if len(key) > maxAttributeNameLength {
		return
	}
	if s, ok := val.(string); ok && len(s) > maxAttributeValueLength {
		val = s[:maxAttributeValueLength]
	}
	attrs[key] = val
}

// loglevelFor maps a loglayer LogLevel to New Relic's loglevel string.
// Fatal and Panic both map to "critical" (New Relic's highest-severity
// log level).
func loglevelFor(l loglayer.LogLevel) string {
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
	case loglayer.LogLevelFatal, loglayer.LogLevelPanic:
		return "critical"
	default:
		return "info"
	}
}
