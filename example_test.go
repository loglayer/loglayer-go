package loglayer_test

// Example* functions documented per Go convention. Each is a runnable test
// (the `// Output:` comment is verified by `go test`). They surface in
// pkg.go.dev next to the corresponding type/method.
//
// Examples use a small in-file `exampleTransport` that emits one JSON
// line per entry with a fixed time so the output is deterministic. It's
// private to this file because the public transports live in their own
// Go modules and main can't import them without a require cycle.
//
// Each example keeps at most one user-supplied key per emission to avoid
// any map-iteration-order ambiguity.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"sort"
	"strings"

	"go.loglayer.dev/v2"
	"go.loglayer.dev/v2/transport"
)

// fixedTime returns a deterministic timestamp for example output.
func fixedTime() string { return "2026-04-26T12:00:00Z" }

// exampleTransport emits one JSON line per entry to stdout with a fixed
// `level, time, msg` prefix followed by data + metadata keys in
// alphabetical order. Matches the on-disk format the godoc `// Output:`
// comments below assert against.
type exampleTransport struct{ id string }

func (t exampleTransport) ID() string           { return t.id }
func (exampleTransport) IsEnabled() bool        { return true }
func (exampleTransport) GetLoggerInstance() any { return nil }
func (exampleTransport) SendToLogger(p loglayer.TransportParams) {
	parts := make([]string, 0, 8)
	parts = append(parts, fmt.Sprintf(`"level":%q`, p.LogLevel.String()))
	parts = append(parts, fmt.Sprintf(`"time":%q`, fixedTime()))

	// Preserve the v1 "prefix folded into the message" rendering
	// for these examples; in v2 the prefix arrives on p.Prefix and
	// each transport decides how to render it.
	msgs := transport.JoinPrefixAndMessages(p.Prefix, p.Messages)
	msg := ""
	if len(msgs) > 0 {
		bits := make([]string, len(msgs))
		for i, m := range msgs {
			bits[i] = fmt.Sprint(m)
		}
		msg = strings.Join(bits, " ")
	}
	parts = append(parts, fmt.Sprintf(`"msg":%q`, msg))

	merged := map[string]any{}
	maps.Copy(merged, p.Data)
	if md, ok := p.Metadata.(loglayer.Metadata); ok {
		maps.Copy(merged, md)
	}
	keys := make([]string, 0, len(merged))
	for k := range merged {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		b, _ := json.Marshal(merged[k])
		parts = append(parts, fmt.Sprintf(`%q:%s`, k, b))
	}
	fmt.Fprintln(os.Stdout, "{"+strings.Join(parts, ",")+"}")
}

// exampleLogger builds a logger that uses exampleTransport so example
// output is reproducible by `go test`.
func exampleLogger() *loglayer.LogLayer {
	return loglayer.New(loglayer.Config{
		Transport:        exampleTransport{},
		DisableFatalExit: true,
	})
}

// tagTransport prefixes its ID onto each emitted message. Used by
// Examples that need to show which of several transports received an
// entry (AddTransport, multi-transport routing).
type tagTransport struct{ id string }

func (t tagTransport) ID() string           { return t.id }
func (tagTransport) IsEnabled() bool        { return true }
func (tagTransport) GetLoggerInstance() any { return nil }
func (t tagTransport) SendToLogger(p loglayer.TransportParams) {
	msg := ""
	if len(p.Messages) > 0 {
		msg = fmt.Sprint(p.Messages[0])
	}
	fmt.Printf("[%s] %s\n", t.id, msg)
}

// Construct a logger and chain persistent fields and per-call metadata
// onto a single log entry. The exampleTransport defined in this file
// emits a deterministic JSON line; production code uses one of the
// transports under go.loglayer.dev/transports/<name>.
func Example() {
	log := loglayer.New(loglayer.Config{
		Transport:        exampleTransport{},
		DisableFatalExit: true,
	})
	log.WithFields(loglayer.Fields{"requestId": "abc"}).
		WithMetadata(loglayer.Metadata{"durationMs": 42}).
		Info("served")
	// Output: {"level":"info","time":"2026-04-26T12:00:00Z","msg":"served","durationMs":42,"requestId":"abc"}
}

func ExampleNew() {
	log := loglayer.New(loglayer.Config{
		Transport: exampleTransport{},
	})
	log.Info("hello")
	// Output: {"level":"info","time":"2026-04-26T12:00:00Z","msg":"hello"}
}

// Build returns an error instead of panicking on misconfiguration.
func ExampleBuild() {
	_, err := loglayer.Build(loglayer.Config{}) // no transport
	if errors.Is(err, loglayer.ErrNoTransport) {
		os.Stdout.WriteString("missing transport\n")
	}
	// Output: missing transport
}

// NewMock returns a silent logger for tests; calls accept everything but
// emit nothing.
func ExampleNewMock() {
	log := loglayer.NewMock()
	log.WithFields(loglayer.Fields{"requestId": "abc"}).Info("silent")
	os.Stdout.WriteString("test ran\n")
	// Output: test ran
}

// WithFields returns a new logger with persistent key/value pairs that
// appear on every subsequent log entry. Always assign the result.
func ExampleLogLayer_WithFields() {
	log := exampleLogger()
	log = log.WithFields(loglayer.Fields{"requestId": "abc-123"})
	log.Info("processing")
	// Output: {"level":"info","time":"2026-04-26T12:00:00Z","msg":"processing","requestId":"abc-123"}
}

// WithoutFields removes specific keys (or all keys when called with no args).
func ExampleLogLayer_WithoutFields() {
	log := exampleLogger()
	log = log.WithFields(loglayer.Fields{"keep": "yes", "drop": "no"})
	log = log.WithoutFields("drop")
	log.Info("partial")
	// Output: {"level":"info","time":"2026-04-26T12:00:00Z","msg":"partial","keep":"yes"}
}

// WithMetadata accepts any value for one log entry only. Maps merge at root.
func ExampleLogLayer_WithMetadata() {
	log := exampleLogger()
	log.WithMetadata(loglayer.Metadata{"durationMs": 42}).Info("served")
	// Output: {"level":"info","time":"2026-04-26T12:00:00Z","msg":"served","durationMs":42}
}

// WithError attaches an error to one log entry. The default serializer
// emits {"message": err.Error()}.
func ExampleLogLayer_WithError() {
	log := exampleLogger()
	log.WithError(errors.New("connection refused")).Error("query failed")
	// Output: {"level":"error","time":"2026-04-26T12:00:00Z","msg":"query failed","err":{"message":"connection refused"}}
}

// WithContext attaches a context.Context to one log call. Transports can read
// trace IDs, deadlines, and other request-scoped values from it.
func ExampleLogLayer_WithContext() {
	log := exampleLogger()
	ctx := context.Background()
	log.WithContext(ctx).Info("request received")
	// Output: {"level":"info","time":"2026-04-26T12:00:00Z","msg":"request received"}
}

// Child returns an independent clone. Mutations on the child don't bleed
// back to the parent.
func ExampleLogLayer_Child() {
	parent := exampleLogger().WithFields(loglayer.Fields{"who": "parent"})
	child := parent.WithFields(loglayer.Fields{"who": "child"})

	child.Info("hi")
	parent.Info("hi")
	// Output: {"level":"info","time":"2026-04-26T12:00:00Z","msg":"hi","who":"child"}
	// {"level":"info","time":"2026-04-26T12:00:00Z","msg":"hi","who":"parent"}
}

// WithPrefix returns a new logger with a string prepended to every message.
func ExampleLogLayer_WithPrefix() {
	log := exampleLogger().WithPrefix("[auth]")
	log.Info("login attempt")
	// Output: {"level":"info","time":"2026-04-26T12:00:00Z","msg":"[auth] login attempt"}
}

// MetadataOnly logs just the metadata, with no message text. Useful for
// metric-style entries.
func ExampleLogLayer_MetadataOnly() {
	log := exampleLogger()
	log.MetadataOnly(loglayer.Metadata{"queueDepth": 17})
	// Output: {"level":"info","time":"2026-04-26T12:00:00Z","msg":"","queueDepth":17}
}

// ErrorOnly logs just an error. Override the level via opts.
func ExampleLogLayer_ErrorOnly() {
	log := exampleLogger()
	log.ErrorOnly(errors.New("disk full"))
	// Output: {"level":"error","time":"2026-04-26T12:00:00Z","msg":"","err":{"message":"disk full"}}
}

// Raw bypasses the builder and dispatches a fully-specified entry. Useful
// when forwarding from another logging system.
func ExampleLogLayer_Raw() {
	log := exampleLogger()
	log.Raw(loglayer.RawLogEntry{
		LogLevel: loglayer.LogLevelWarn,
		Messages: []any{"upstream timeout"},
		Metadata: loglayer.Metadata{"retries": 3},
	})
	// Output: {"level":"warn","time":"2026-04-26T12:00:00Z","msg":"upstream timeout","retries":3}
}

// SetLevel raises the threshold so entries below the given level are
// dropped. Mutates the logger in place; the return value is the same
// instance and exists only for chaining.
func ExampleLogLayer_SetLevel() {
	log := exampleLogger()
	log.SetLevel(loglayer.LogLevelWarn)
	log.Info("dropped")
	log.Warn("emitted")
	// Output: {"level":"warn","time":"2026-04-26T12:00:00Z","msg":"emitted"}
}

// NewFieldsHook installs a callback that runs whenever WithFields is
// called. Use it for cross-cutting transformations: redaction, default
// fields, or normalization.
func ExampleNewFieldsHook() {
	addApp := loglayer.NewFieldsHook("add-app", func(in loglayer.Fields) loglayer.Fields {
		in["app"] = "billing"
		return in
	})
	log := loglayer.New(loglayer.Config{
		Transport:        exampleTransport{},
		DisableFatalExit: true,
		Plugins:          []loglayer.Plugin{addApp},
	})
	log.WithFields(loglayer.Fields{"requestId": "abc"}).Info("served")
	// Output: {"level":"info","time":"2026-04-26T12:00:00Z","msg":"served","app":"billing","requestId":"abc"}
}

// Lazy defers a value until log emit time, and only when the level is
// enabled. Use it for fields whose computation is too expensive to pay
// on every call site.
func ExampleLazy() {
	log := exampleLogger().WithFields(loglayer.Fields{
		"computed": loglayer.Lazy(func() any { return 42 }),
	})
	log.Info("done")
	// Output: {"level":"info","time":"2026-04-26T12:00:00Z","msg":"done","computed":42}
}

// FromContext retrieves a *LogLayer attached upstream by NewContext.
// Pair the two in middleware so handlers don't have to thread the
// logger through every call signature.
func ExampleFromContext() {
	log := exampleLogger().WithFields(loglayer.Fields{"requestId": "abc"})
	ctx := loglayer.NewContext(context.Background(), log)

	handler := func(ctx context.Context) {
		loglayer.FromContext(ctx).Info("handling")
	}
	handler(ctx)
	// Output: {"level":"info","time":"2026-04-26T12:00:00Z","msg":"handling","requestId":"abc"}
}

// WithGroup tags entries so they only route to the transports listed
// for that group in Config.Routing. Tagged entries reach the audit
// transport; the general transport is skipped.
func ExampleLogLayer_WithGroup() {
	audit := exampleTransport{id: "audit"}
	general := exampleTransport{id: "general"}
	log := loglayer.New(loglayer.Config{
		Transports:       []loglayer.Transport{audit, general},
		DisableFatalExit: true,
		Routing: loglayer.RoutingConfig{
			Groups: map[string]loglayer.LogGroup{
				"audit": {Transports: []string{"audit"}},
			},
		},
	})
	log.WithGroup("audit").Info("user signed in")
	// Output: {"level":"info","time":"2026-04-26T12:00:00Z","msg":"user signed in"}
}

// NewMetadataHook installs a callback that runs whenever WithMetadata
// or MetadataOnly is called.
func ExampleNewMetadataHook() {
	addEnv := loglayer.NewMetadataHook("add-env", func(in any) any {
		md, ok := in.(loglayer.Metadata)
		if !ok {
			return in
		}
		out := make(loglayer.Metadata, len(md)+1)
		maps.Copy(out, md)
		out["env"] = "prod"
		return out
	})
	log := loglayer.New(loglayer.Config{
		Transport:        exampleTransport{},
		DisableFatalExit: true,
		Plugins:          []loglayer.Plugin{addEnv},
	})
	log.WithMetadata(loglayer.Metadata{"durationMs": 42}).Info("served")
	// Output: {"level":"info","time":"2026-04-26T12:00:00Z","msg":"served","durationMs":42,"env":"prod"}
}

// NewDataHook fires once per emission, after fields and serialized
// error are merged into the assembled Data. Returned keys merge into
// that map; missing keys are left alone.
func ExampleNewDataHook() {
	tagHost := loglayer.NewDataHook("tag-host", func(p loglayer.BeforeDataOutParams) loglayer.Data {
		return loglayer.Data{"host": "web-01"}
	})
	log := loglayer.New(loglayer.Config{
		Transport:        exampleTransport{},
		DisableFatalExit: true,
		Plugins:          []loglayer.Plugin{tagHost},
	})
	log.Info("served")
	// Output: {"level":"info","time":"2026-04-26T12:00:00Z","msg":"served","host":"web-01"}
}

// NewMessageHook fires once per emission and rewrites the messages
// slice.
func ExampleNewMessageHook() {
	prefix := loglayer.NewMessageHook("prefix", func(p loglayer.BeforeMessageOutParams) []any {
		return append([]any{"[svc]"}, p.Messages...)
	})
	log := loglayer.New(loglayer.Config{
		Transport:        exampleTransport{},
		DisableFatalExit: true,
		Plugins:          []loglayer.Plugin{prefix},
	})
	log.Info("served")
	// Output: {"level":"info","time":"2026-04-26T12:00:00Z","msg":"[svc] served"}
}

// NewLevelHook can override the level of an entry just before
// per-transport dispatch.
func ExampleNewLevelHook() {
	bumpRetry := loglayer.NewLevelHook("retry-bump", func(p loglayer.TransformLogLevelParams) (loglayer.LogLevel, bool) {
		if len(p.Messages) == 0 {
			return p.LogLevel, false
		}
		s, ok := p.Messages[0].(string)
		if !ok || !strings.HasPrefix(s, "RETRY") {
			return p.LogLevel, false
		}
		return loglayer.LogLevelWarn, true
	})
	log := loglayer.New(loglayer.Config{
		Transport:        exampleTransport{},
		DisableFatalExit: true,
		Plugins:          []loglayer.Plugin{bumpRetry},
	})
	log.Info("RETRY connection")
	// Output: {"level":"warn","time":"2026-04-26T12:00:00Z","msg":"RETRY connection"}
}

// NewSendGate gates per-(entry, transport) dispatch. Return false to
// drop the entry for that transport; other transports are unaffected.
// When multiple gates are installed, the entry sends only when every
// one returns true.
func ExampleNewSendGate() {
	noPings := loglayer.NewSendGate("no-pings", func(p loglayer.ShouldSendParams) bool {
		for _, m := range p.Messages {
			if s, ok := m.(string); ok && strings.Contains(s, "ping") {
				return false
			}
		}
		return true
	})
	log := loglayer.New(loglayer.Config{
		Transport:        exampleTransport{},
		DisableFatalExit: true,
		Plugins:          []loglayer.Plugin{noPings},
	})
	log.Info("ping")
	log.Info("served")
	// Output: {"level":"info","time":"2026-04-26T12:00:00Z","msg":"served"}
}

// AddTransport registers a transport on a running logger. Subsequent
// emissions fan out to every registered transport, in registration
// order. A new transport whose ID matches an existing one replaces
// (and closes) the prior instance.
func ExampleLogLayer_AddTransport() {
	log := loglayer.New(loglayer.Config{
		Transport:        tagTransport{id: "primary"},
		DisableFatalExit: true,
	})
	log.Info("before")

	log.AddTransport(tagTransport{id: "audit"})
	log.Info("after")
	// Output:
	// [primary] before
	// [primary] after
	// [audit] after
}

type queryErr struct{ msg string }

func (e *queryErr) Error() string { return e.msg }

// ErrorSerializer customizes how errors attached via WithError are
// rendered in the assembled Data. The default emits {"message":
// err.Error()}; override to add type, stack, or wrapped-cause fields.
func ExampleErrorSerializer() {
	log := loglayer.New(loglayer.Config{
		Transport:        exampleTransport{},
		DisableFatalExit: true,
		ErrorSerializer: func(err error) map[string]any {
			return map[string]any{
				"message": err.Error(),
				"type":    fmt.Sprintf("%T", err),
			}
		},
	})
	log.WithError(&queryErr{msg: "connection refused"}).Error("query failed")
	// Output: {"level":"error","time":"2026-04-26T12:00:00Z","msg":"query failed","err":{"message":"connection refused","type":"*loglayer_test.queryErr"}}
}
