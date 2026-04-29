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
	"os"
	"sort"
	"strings"

	"go.loglayer.dev"
)

// fixedTime returns a deterministic timestamp for example output.
func fixedTime() string { return "2026-04-26T12:00:00Z" }

// exampleTransport emits one JSON line per entry to stdout with a fixed
// `level, time, msg` prefix followed by data + metadata keys in
// alphabetical order. Matches the on-disk format the godoc `// Output:`
// comments below assert against.
type exampleTransport struct{}

func (exampleTransport) ID() string             { return "example" }
func (exampleTransport) IsEnabled() bool        { return true }
func (exampleTransport) GetLoggerInstance() any { return nil }
func (exampleTransport) SendToLogger(p loglayer.TransportParams) {
	parts := make([]string, 0, 8)
	parts = append(parts, fmt.Sprintf(`"level":%q`, p.LogLevel.String()))
	parts = append(parts, fmt.Sprintf(`"time":%q`, fixedTime()))

	msg := ""
	if len(p.Messages) > 0 {
		bits := make([]string, len(p.Messages))
		for i, m := range p.Messages {
			bits[i] = fmt.Sprint(m)
		}
		msg = strings.Join(bits, " ")
	}
	parts = append(parts, fmt.Sprintf(`"msg":%q`, msg))

	merged := map[string]any{}
	for k, v := range p.Data {
		merged[k] = v
	}
	if md, ok := p.Metadata.(loglayer.Metadata); ok {
		for k, v := range md {
			merged[k] = v
		}
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

// Construct a logger and emit a basic message.
func Example() {
	log := exampleLogger()
	log.Info("hello world")
	// Output: {"level":"info","time":"2026-04-26T12:00:00Z","msg":"hello world"}
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
