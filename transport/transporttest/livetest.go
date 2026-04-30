package transporttest

import (
	"context"
	"errors"

	"go.loglayer.dev"
)

// EmitLivetestSurface emits a representative sample of the LogLayer API
// surface against the supplied logger. Each entry carries the given
// livetestID under the "livetest_id" persistent field so the operator can
// filter the destination's UI to the entries this run produced, plus a
// per-step "step" field so individual shapes can be located.
//
// Intended for use inside `//go:build livetest`-tagged tests against real
// SDKs / managed services. The caller should configure the logger with
// DisableFatalExit so the Fatal-level emit doesn't terminate the test
// process.
//
// Panic-level emits are deliberately omitted; calling [LogLayer.Panic]
// would panic the goroutine and cut the variety pack short.
func EmitLivetestSurface(log *loglayer.LogLayer, livetestID string) {
	log = log.WithFields(loglayer.Fields{"livetest_id": livetestID})

	// step adorns the next emission with a step name so the operator can
	// scan for a specific shape without re-reading the test source.
	step := func(name string) *loglayer.LogLayer {
		return log.WithFields(loglayer.Fields{"step": name})
	}

	// Levels (Trace through Fatal). Fatal emits at fatal severity but does
	// not exit, given DisableFatalExit on the supplied logger.
	step("01-trace").Trace("trace severity entry")
	step("02-debug").Debug("debug severity entry")
	step("03-info-simple").Info("info severity entry")
	step("04-info-multi").Info("multi", "argument", "info entry")
	step("05-warn").Warn("warn severity entry")
	step("06-error-plain").Error("error severity entry")
	step("07-fatal").Fatal("fatal severity entry (no exit)")

	// Map metadata: typed scalars + slices, exercising every shape a
	// destination is likely to render natively.
	step("08-map-metadata").WithMetadata(loglayer.Metadata{
		"requestId": "req-12345",
		"userId":    42,
		"score":     1.5,
		"isAdmin":   true,
		"tags":      []string{"alpha", "beta"},
	}).Info("with map metadata")

	// Struct metadata: exercises the per-transport "non-map metadata"
	// placement (nested under MetadataFieldName for most wrappers).
	type sample struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	step("09-struct-metadata").WithMetadata(sample{ID: 7, Name: "Alice"}).Info("with struct metadata")

	// Three-level-deep nested metadata. Exercises each transport's
	// rendering of nested maps (YAML for pretty, nested JSON for
	// structured, attribute paths for the wrapper transports).
	step("10-deeply-nested-metadata").WithMetadata(loglayer.Metadata{
		"user": loglayer.Metadata{
			"id": 42,
			"address": loglayer.Metadata{
				"city": "Brooklyn",
				"zip":  "11201",
			},
		},
	}).Info("with deeply nested metadata")

	// Persistent fields plus per-call metadata.
	step("11-fields-and-metadata").WithMetadata(loglayer.Metadata{
		"durationMs": 184,
	}).Info("served")

	// Error attached.
	step("12-with-error").
		WithError(errors.New("simulated downstream timeout")).
		Error("operation failed")

	// Error + metadata + multiple message parts.
	step("13-with-error-and-metadata").
		WithMetadata(loglayer.Metadata{"attempt": 3, "stage": "commit"}).
		WithError(errors.New("retry exhausted")).
		Error("operation failed", "after retries")

	// MetadataOnly: emits at info level with no message body.
	step("14-metadata-only").MetadataOnly(loglayer.Metadata{
		"event": "heartbeat",
		"ok":    true,
	})

	// ErrorOnly: emits at error level with the error as the body.
	step("15-error-only").ErrorOnly(errors.New("standalone error"))

	// Raw: bypasses the builder. Useful for forwarding from another
	// logging system. Carries its own fields/metadata bag.
	step("16-raw").Raw(loglayer.RawLogEntry{
		LogLevel: loglayer.LogLevelWarn,
		Messages: []any{"raw log entry"},
		Fields:   loglayer.Fields{"source": "external"},
		Metadata: loglayer.Metadata{"raw_attempt": 1},
	})

	// WithContext: forwards a context.Context to transports that consume it
	// (OTel, slog, Sentry's hub-aware emit). The context here carries no
	// trace/span; the point is to verify the dispatch path doesn't break.
	step("17-with-context").
		WithContext(context.Background()).
		Info("with context")
}

// LivetestVariant names a loglayer.Config shape worth covering in a
// livetest run, alongside the suffix that disambiguates entries from
// that variant in the destination's UI.
type LivetestVariant struct {
	// Name is appended to the base livetest ID to form a unique
	// per-variant tag (e.g. "<base>-default", "<base>-keyed"). It
	// also surfaces in the test's verification log line.
	Name string
	// Config is layered on top of the per-test base config (Transport,
	// DisableFatalExit). Set FieldsKey, MetadataFieldName, etc. to
	// exercise the keyed shape; leave zero for the default shape.
	Config loglayer.Config
}

// LivetestVariants is the standard pair every wrapper-style livetest
// should exercise: the unconfigured default shape (fields and map
// metadata flatten at root, non-map metadata under each transport's
// hardcoded "metadata") and the keyed shape (fields under "context",
// all metadata uniformly under "metadata").
var LivetestVariants = []LivetestVariant{
	{Name: "default", Config: loglayer.Config{}},
	{Name: "keyed", Config: loglayer.Config{
		FieldsKey:         "context",
		MetadataFieldName: "metadata",
	}},
}

// SendLivetestVariants emits [EmitLivetestSurface] against the supplied
// transport once per [LivetestVariants] entry. Returns the per-variant
// livetest IDs (in the same order as LivetestVariants) so the caller
// can print verification instructions tailored to each variant.
//
// The transport is shared across passes; the caller is responsible for
// flushing or closing it after this returns. baseID is suffixed with
// each variant's Name to form the per-pass tag.
func SendLivetestVariants(tr loglayer.Transport, baseID string) []string {
	ids := make([]string, len(LivetestVariants))
	for i, v := range LivetestVariants {
		cfg := v.Config
		cfg.Transport = tr
		cfg.DisableFatalExit = true
		log := loglayer.New(cfg)
		ids[i] = baseID + "-" + v.Name
		EmitLivetestSurface(log, ids[i])
	}
	return ids
}
