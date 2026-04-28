package loglayer_test

// Example* functions documented per Go convention. Each is a runnable test
// (the `// Output:` comment is verified by `go test`). They surface in
// pkg.go.dev next to the corresponding type/method, and IDEs show them in
// hover popups via the gopls "View Examples" affordance.
//
// All examples use the structured transport with a fixed DateFn so the
// output is deterministic. The transport writes a fixed `level, time, msg`
// header followed by user-supplied keys; Go map iteration is randomized,
// so examples keep at most one user key per logger to make the output
// reproducible.

import (
	"context"
	"errors"
	"os"

	"go.loglayer.dev"
	"go.loglayer.dev/transports/structured"
)

// fixedTime returns a deterministic timestamp for example output.
func fixedTime() string { return "2026-04-26T12:00:00Z" }

// exampleLogger builds a logger that writes to stdout with a deterministic
// date so example output is reproducible by `go test`.
func exampleLogger() *loglayer.LogLayer {
	return loglayer.New(loglayer.Config{
		Transport: structured.New(structured.Config{
			Writer: os.Stdout,
			DateFn: fixedTime,
		}),
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
		Transport: structured.New(structured.Config{
			Writer: os.Stdout,
			DateFn: fixedTime,
		}),
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

// WithCtx attaches a context.Context to one log call. Transports can read
// trace IDs, deadlines, and other request-scoped values from it.
func ExampleLogLayer_WithCtx() {
	log := exampleLogger()
	ctx := context.Background()
	log.WithCtx(ctx).Info("request received")
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
