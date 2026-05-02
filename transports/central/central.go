// Package central sends log entries to the LogLayer Central log aggregation
// server's HTTP intake.
//
// Wraps transports/http with Central-specific defaults:
//   - Base URL of http://localhost:9800 (overridable via Config.BaseURL)
//   - POST to /api/logs as a JSON array
//   - Encoder that emits Central's expected log shape (service, message,
//     level, timestamp, instanceId, context, metadata, error, groups, tags)
//     split out from the assembled fields and per-call metadata
package central

import (
	"cmp"
	"strings"

	"github.com/goccy/go-json"

	httptr "go.loglayer.dev/transports/http/v2"
	"go.loglayer.dev/v2/transport"
)

// DefaultPort is the port the Central server listens on by default.
const DefaultPort = 9800

// DefaultBaseURL is the default base URL for the Central server.
const DefaultBaseURL = "http://localhost:9800"

// IntakePath is the path appended to the base URL for log ingestion.
const IntakePath = "/api/logs"

// Config holds Central transport configuration.
type Config struct {
	transport.BaseConfig

	// BaseURL is the base URL of the Central server. Defaults to
	// "http://localhost:9800". The transport appends "/api/logs" to this
	// when building the intake URL.
	BaseURL string

	// Service identifies the source application in the Central server.
	// Required.
	Service string

	// InstanceID differentiates between multiple instances of the same
	// service. Optional.
	InstanceID string

	// Tags are static tags attached to every entry. Conventionally
	// key:value strings (e.g. []string{"env:prod", "region:us-east"}).
	// Optional.
	Tags []string

	// HTTP overrides batching, client, error handling, and any other
	// transports/http settings. The URL and Encoder are set by this
	// package and cannot be overridden via this field.
	HTTP httptr.Config
}

// Transport wraps a transports/http.Transport with Central-specific encoding
// and defaults.
type Transport struct {
	*httptr.Transport
}

// New constructs a Central Transport. Panics if Config.Service is empty.
// Use Build for an error-returning variant.
func New(cfg Config) *Transport {
	t, err := Build(cfg)
	if err != nil {
		panic(err)
	}
	return t
}

// Build constructs a Central Transport like New but returns
// ErrServiceRequired instead of panicking when cfg.Service is empty. Use
// this when the service identifier is loaded at runtime (e.g. from an
// environment variable) and you want to handle the missing-config case
// explicitly.
func Build(cfg Config) (*Transport, error) {
	if cfg.Service == "" {
		return nil, ErrServiceRequired
	}
	if cfg.HTTP.URL != "" || cfg.HTTP.Encoder != nil {
		return nil, ErrHTTPOverrideForbidden
	}

	httpCfg := cfg.HTTP
	httpCfg.BaseConfig = cfg.BaseConfig
	httpCfg.URL = strings.TrimRight(cmp.Or(cfg.BaseURL, DefaultBaseURL), "/") + IntakePath
	httpCfg.Encoder = newEncoder(cfg)

	httpT, err := httptr.Build(httpCfg)
	if err != nil {
		return nil, err
	}
	return &Transport{Transport: httpT}, nil
}

// newEncoder produces the JSON-array encoder for Central's intake format.
// See TestCentral_PayloadShape for the full per-entry shape.
func newEncoder(cfg Config) httptr.Encoder {
	return httptr.EncoderFunc(func(entries []httptr.Entry) ([]byte, string, error) {
		objs := make([]map[string]any, len(entries))
		for i, e := range entries {
			obj := make(map[string]any, 8)
			obj["service"] = cfg.Service
			obj["message"] = transport.JoinMessages(e.Messages)
			obj["level"] = e.Level.String()
			obj["timestamp"] = e.Time.UTC().Format("2006-01-02T15:04:05.000Z")

			if cfg.InstanceID != "" {
				obj["instanceId"] = cfg.InstanceID
			}
			if len(e.Groups) > 0 {
				obj["groups"] = e.Groups
			}
			if len(cfg.Tags) > 0 {
				obj["tags"] = cfg.Tags
			}

			// Lift the error out of Data into the top-level "error" field;
			// the remaining Data lands as "context". The error key is
			// whatever loglayer.Config.ErrorFieldName resolved to, exposed
			// via Schema.
			errKey := e.Schema.ErrorFieldName
			if errVal, ok := e.Data[errKey]; ok {
				obj["error"] = errVal
				if len(e.Data) > 1 {
					ctx := make(map[string]any, len(e.Data)-1)
					for k, v := range e.Data {
						if k == errKey {
							continue
						}
						ctx[k] = v
					}
					obj["context"] = ctx
				}
			} else if len(e.Data) > 0 {
				obj["context"] = e.Data
			}

			if e.Metadata != nil {
				obj["metadata"] = e.Metadata
			}

			objs[i] = obj
		}
		body, err := json.Marshal(objs)
		return body, "application/json", err
	})
}
