// Package newrelic sends log entries to the New Relic Logs API.
//
// Wraps transports/http with New Relic-specific defaults:
//   - Log API endpoint (https://log-api.newrelic.com/log/v1)
//   - Api-Key header from Config.APIKey
//   - NDJSON encoder emitting New Relic's expected log shape
//     (timestamp, message, loglevel, hostname, and arbitrary attributes)
//
// API reference: https://docs.newrelic.com/docs/logs/log-api/introduction-log-api/
//
// See https://go.loglayer.dev for usage guides and the full API reference.
package newrelic

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"

	"github.com/goccy/go-json"

	httptr "go.loglayer.dev/transports/http/v2"
	"go.loglayer.dev/v2"
	"go.loglayer.dev/v2/transport"
)

const defaultURL = "https://log-api.newrelic.com/log/v1"

// IntakeZone identifies the New Relic log API region.
type IntakeZone string

const (
	// ZoneUS is the default (US) New Relic log API region.
	ZoneUS IntakeZone = "US"
	// ZoneEU is the EU New Relic log API region.
	ZoneEU IntakeZone = "EU"
)

// URL returns the Log API endpoint for the zone.
func (z IntakeZone) URL() string {
	switch z {
	case ZoneEU:
		return "https://log-api.eu.newrelic.com/log/v1"
	default:
		return defaultURL
	}
}

// Config holds New Relic transport configuration.
type Config struct {
	transport.BaseConfig

	// APIKey is the New Relic user key. Required. The Api-Key header is
	// set from this value on every request.
	//
	// Tagged json:"-" so that log.WithMetadata(cfg).Info(...) through
	// any JSON-emitting transport won't ship the key in the rendered log.
	APIKey string `json:"-"`

	// LicenseKey is the New Relic ingest license key. When set, the
	// X-License-Key header is included. Optional; use it when your
	// account requires it (some New Relic accounts enforce license keys
	// instead of, or alongside, API keys).
	//
	// Tagged json:"-" for the same defense-in-depth reasons as APIKey.
	LicenseKey string `json:"-"`

	// Zone selects the New Relic region. Defaults to ZoneUS. Ignored
	// when URL is set.
	Zone IntakeZone

	// URL overrides the Zone-derived Log API endpoint. Use it for on-prem
	// deployments or for testing against a mock endpoint. When set, Zone
	// is ignored.
	URL string

	// Hostname maps to the "hostname" attribute on each log entry.
	// The application or host name identifying this source. Optional.
	Hostname string

	// AllowInsecureURL permits Config.URL to use a non-https scheme. The
	// API key is sent in the Api-Key header on every request; without
	// this flag, Build refuses a non-https URL to keep the key off the
	// wire in cleartext. Set true only when an on-prem forwarder
	// terminates TLS upstream and a private network carries the cleartext
	// hop. The Zone-derived endpoints are always https and unaffected.
	AllowInsecureURL bool

	// HTTP overrides batching, client, error handling, and any other
	// transports/http settings. The URL, Encoder, Api-Key header, and
	// X-License-Key header are set by this package and cannot be
	// overridden via this field.
	HTTP httptr.Config
}

// String returns a redacted form of the config so that an accidental
// log.Info(cfg) (or fmt.Sprintf("%v", cfg)) can't ship the API key.
// Both APIKey and LicenseKey are replaced with a fixed mask regardless
// of length.
//
// Note: Go's fmt verbs %+v and %#v intentionally bypass Stringer and
// always print struct fields. Code that uses those verbs against
// Config will see the raw keys. Reserve %+v / %#v for debugger-style
// inspection, never for production logs. The json:"-" tags on APIKey
// and LicenseKey prevent the JSON-via-transport leak path; this method
// covers the fmt.Sprintf path; %+v / %#v are explicitly out of scope.
func (c Config) String() string {
	apiKey := c.APIKey
	if apiKey != "" {
		apiKey = "***redacted***"
	}
	licenseKey := c.LicenseKey
	if licenseKey != "" {
		licenseKey = "***redacted***"
	}
	return fmt.Sprintf(
		"newrelic.Config{APIKey:%q LicenseKey:%q Zone:%q URL:%q Hostname:%q}",
		apiKey, licenseKey, c.Zone, c.URL, c.Hostname,
	)
}

// Transport wraps a transports/http.Transport with New Relic-specific
// encoding and defaults.
type Transport struct {
	*httptr.Transport
}

// New constructs a New Relic Transport. Panics if Config.APIKey is empty.
// Use Build for an error-returning variant.
func New(cfg Config) *Transport {
	t, err := Build(cfg)
	if err != nil {
		panic(err)
	}
	return t
}

// Build constructs a New Relic Transport like New but returns
// ErrAPIKeyRequired instead of panicking when cfg.APIKey is empty. Use
// this when the API key is loaded at runtime (e.g. from an environment
// variable) and you want to handle the missing-config case explicitly.
func Build(cfg Config) (*Transport, error) {
	if cfg.APIKey == "" {
		return nil, ErrAPIKeyRequired
	}

	httpCfg := cfg.HTTP
	httpCfg.BaseConfig = cfg.BaseConfig
	if cfg.HTTP.URL != "" || cfg.HTTP.Encoder != nil {
		return nil, ErrHTTPOverrideForbidden
	}

	if cfg.URL != "" {
		if !cfg.AllowInsecureURL {
			parsed, err := url.Parse(cfg.URL)
			if err != nil || !strings.EqualFold(parsed.Scheme, "https") {
				return nil, ErrInsecureURL
			}
		}
		httpCfg.URL = cfg.URL
	} else {
		httpCfg.URL = cfg.Zone.URL()
	}
	httpCfg.Encoder = newEncoder(cfg)

	// Clone Headers so we don't mutate the caller's map.
	merged := make(map[string]string, len(cfg.HTTP.Headers)+2)
	for k, v := range cfg.HTTP.Headers {
		merged[k] = v
	}
	merged["Api-Key"] = cfg.APIKey
	if cfg.LicenseKey != "" {
		merged["X-License-Key"] = cfg.LicenseKey
	}
	httpCfg.Headers = merged

	httpT, err := httptr.Build(httpCfg)
	if err != nil {
		return nil, err
	}
	return &Transport{Transport: httpT}, nil
}

// Close drains the queue and stops the background worker.
// Safe to call multiple times.
func (t *Transport) Close() error {
	return t.Transport.Close()
}

// GetLoggerInstance returns nil; the New Relic transport has no underlying logger.
func (t *Transport) GetLoggerInstance() any { return nil }

// newEncoder produces the NDJSON encoder for New Relic's Log API format.
// Each entry is serialized as a JSON object with timestamp, message, loglevel,
// optional hostname, and user fields/metadata merged as root-level attributes.
// Lines are joined with "\n" for NDJSON.
func newEncoder(cfg Config) httptr.Encoder {
	return httptr.EncoderFunc(func(entries []httptr.Entry) ([]byte, string, error) {
		var buf bytes.Buffer
		for i, e := range entries {
			obj := make(map[string]any, 4+len(e.Data))
			obj["timestamp"] = e.Time.UTC().Format("2006-01-02T15:04:05.000Z07:00")
			obj["message"] = transport.JoinMessages(e.Messages)
			obj["loglevel"] = loglevelFor(e.Level)
			if cfg.Hostname != "" {
				obj["hostname"] = cfg.Hostname
			}
			transport.MergeIntoMap(obj, e.Data, e.Metadata, e.Schema.MetadataFieldName)
			line, err := json.Marshal(obj)
			if err != nil {
				return nil, "", fmt.Errorf("newrelic: marshal entry %d: %w", i, err)
			}
			buf.Write(line)
			if i < len(entries)-1 {
				buf.WriteByte('\n')
			}
		}
		return buf.Bytes(), "application/json", nil
	})
}

// loglevelFor maps a loglayer LogLevel to New Relic's loglevel string.
// New Relic recognizes: debug, info, warning, error, critical.
// Trace folds into "debug"; Fatal and Panic map to "critical".
func loglevelFor(l loglayer.LogLevel) string {
	switch l {
	case loglayer.LogLevelTrace, loglayer.LogLevelDebug:
		return "debug"
	case loglayer.LogLevelInfo:
		return "info"
	case loglayer.LogLevelWarn:
		return "warning"
	case loglayer.LogLevelError:
		return "error"
	case loglayer.LogLevelFatal, loglayer.LogLevelPanic:
		return "critical"
	default:
		return "info"
	}
}
