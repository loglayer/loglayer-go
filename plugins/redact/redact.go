// Package redact provides a LogLayer plugin that replaces sensitive values
// in metadata and fields before they reach a transport.
//
// Three matching modes:
//
//   - Keys: exact, case-sensitive key/field names. Honors `json` tags
//     when matching struct fields.
//   - Patterns: regular expressions matched against string values at any
//     depth.
//   - (implicit) struct walking: structs are introspected by default. The
//     metadata's runtime type is preserved (struct in → struct out, map
//     in → map out, slice in → slice out).
//
// Caller's input is never mutated; the plugin clones whatever it touches.
//
// Usage:
//
//	log := loglayer.New(loglayer.Config{
//	    Transport: structured.New(structured.Config{}),
//	})
//	log.AddPlugin(redact.New(redact.Config{
//	    Keys: []string{"password", "apiKey", "ssn"},
//	}))
//
// See https://go.loglayer.dev for usage guides and the full API reference.
package redact

import (
	"regexp"

	"go.loglayer.dev"
	"go.loglayer.dev/utils/maputil"
)

// Config holds redactor configuration.
type Config struct {
	// ID for the plugin. Defaults to "redact". Override when registering
	// multiple redactors at once (e.g. one for PII, one for secrets).
	ID string

	// Keys whose values are replaced with Censor wherever they appear.
	// Matches map keys (string-keyed maps only) and struct field names
	// (json tag preferred, fallback to the Go field name). Walks into
	// nested maps, structs, slices, arrays, and pointers at any depth.
	// Match is exact and case-sensitive.
	Keys []string

	// Patterns are regular expressions matched against string values
	// (not keys) at any depth. A value matching any pattern is replaced
	// with Censor.
	Patterns []*regexp.Regexp

	// Censor is the replacement value. Defaults to "[REDACTED]".
	//
	// For string-typed fields (struct field, map value, slice element),
	// the censor is stringified via fmt.Sprintf if non-string. For
	// interface{} values, the censor is passed through. For other typed
	// fields (int, time.Time, etc.) we can't safely substitute a foreign
	// value; the field is set to its zero value.
	Censor any
}

// New constructs a redaction plugin from the config. The returned plugin
// implements [loglayer.MetadataHook], [loglayer.FieldsHook], and
// [loglayer.DataHook]. The DataHook re-walks the assembled Data map so
// the framework-built error subtree (`{"message": err.Error()}` from
// WithError) is also redacted; without it a Pattern-style redactor
// would catch secrets in metadata/fields but miss the same secrets
// rendered by an error's Error() string.
func New(cfg Config) loglayer.Plugin {
	id := cfg.ID
	if id == "" {
		id = "redact"
	}
	censor := cfg.Censor
	if censor == nil {
		censor = "[REDACTED]"
	}

	keySet := make(map[string]struct{}, len(cfg.Keys))
	for _, k := range cfg.Keys {
		keySet[k] = struct{}{}
	}

	cloner := &maputil.Cloner{
		Censor: censor,
	}
	if len(keySet) > 0 {
		cloner.MatchKey = func(k string) bool {
			_, ok := keySet[k]
			return ok
		}
	}
	if len(cfg.Patterns) > 0 {
		patterns := cfg.Patterns
		cloner.MatchValue = func(s string) bool {
			for _, p := range patterns {
				if p.MatchString(s) {
					return true
				}
			}
			return false
		}
	}
	return &plugin{id: id, cloner: cloner}
}

type plugin struct {
	id     string
	cloner *maputil.Cloner
}

func (p *plugin) ID() string { return p.id }

func (p *plugin) OnMetadataCalled(metadata any) any {
	return p.cloner.Clone(metadata)
}

func (p *plugin) OnFieldsCalled(fields loglayer.Fields) loglayer.Fields {
	cloned := p.cloner.Clone(map[string]any(fields))
	if cloned == nil {
		return nil
	}
	return loglayer.Fields(cloned.(map[string]any))
}

func (p *plugin) OnBeforeDataOut(params loglayer.BeforeDataOutParams) loglayer.Data {
	if params.Data == nil {
		return nil
	}
	cloned := p.cloner.Clone(map[string]any(params.Data))
	if cloned == nil {
		return params.Data
	}
	return loglayer.Data(cloned.(map[string]any))
}
