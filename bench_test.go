package loglayer_test

// Benchmarks for the framework core, the in-main renderer transports
// (structured/console/testing), and shared dispatch paths.
//
// Per-vendor wrapper benchmarks (Direct_Zerolog vs Wrapped_Zerolog,
// etc.) live in each wrapper's own module so the main module doesn't
// pull every vendor SDK into its dependency graph. Render_Pretty
// benchmarks live in transports/pretty for the same reason.
//
// Run with:
//   go test -bench=. -benchmem -run=^$ -benchtime=1s .
//
// Note on the writer: we use a custom discardWriter rather than io.Discard
// because charmbracelet/log detects io.Discard and skips its formatting
// pipeline entirely, which would understate its real cost. discardWriter
// looks like any other writer so every library exercises its full write path.

import (
	"testing"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
	"go.loglayer.dev/transports/console"
	"go.loglayer.dev/transports/structured"
	lltest "go.loglayer.dev/transports/testing"
)

type benchUser struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

var benchTestUser = benchUser{ID: 42, Name: "Alice", Email: "alice@example.com"}

const benchMsg = "user logged in"

type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }

var discard discardWriter

func benchMetadata() loglayer.Metadata {
	return loglayer.Metadata{
		"id":    42,
		"name":  "Alice",
		"email": "alice@example.com",
	}
}

type noopTransport struct{}

func (n *noopTransport) ID() string                              { return "noop" }
func (n *noopTransport) IsEnabled() bool                         { return true }
func (n *noopTransport) SendToLogger(_ loglayer.TransportParams) {}
func (n *noopTransport) GetLoggerInstance() any                  { return nil }

func runSimple(b *testing.B, log *loglayer.LogLayer) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.Info(benchMsg)
	}
}

func runMap(b *testing.B, log *loglayer.LogLayer) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.WithMetadata(benchMetadata()).Info(benchMsg)
	}
}

func runStruct(b *testing.B, log *loglayer.LogLayer) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.WithMetadata(benchTestUser).Info(benchMsg)
	}
}

func BenchmarkRender_Structured_SimpleMessage(b *testing.B) {
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transport: structured.New(structured.Config{
			BaseConfig: transport.BaseConfig{ID: "structured"},
			Writer:     discard,
		}),
	})
	runSimple(b, log)
}

func BenchmarkRender_Structured_MapMetadata(b *testing.B) {
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transport: structured.New(structured.Config{
			BaseConfig: transport.BaseConfig{ID: "structured"},
			Writer:     discard,
		}),
	})
	runMap(b, log)
}

func BenchmarkRender_Structured_StructMetadata(b *testing.B) {
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transport: structured.New(structured.Config{
			BaseConfig: transport.BaseConfig{ID: "structured"},
			Writer:     discard,
		}),
	})
	runStruct(b, log)
}

func BenchmarkRender_Console_SimpleMessage(b *testing.B) {
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transport: console.New(console.Config{
			BaseConfig: transport.BaseConfig{ID: "console"},
			Writer:     discard,
		}),
	})
	runSimple(b, log)
}

func BenchmarkRender_Console_MapMetadata(b *testing.B) {
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transport: console.New(console.Config{
			BaseConfig: transport.BaseConfig{ID: "console"},
			Writer:     discard,
		}),
	})
	runMap(b, log)
}

func BenchmarkRender_Testing_SimpleMessage(b *testing.B) {
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transport: lltest.New(lltest.Config{
			BaseConfig: transport.BaseConfig{ID: "test"},
		}),
	})
	runSimple(b, log)
}

func BenchmarkRender_Testing_MapMetadata(b *testing.B) {
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transport: lltest.New(lltest.Config{
			BaseConfig: transport.BaseConfig{ID: "test"},
		}),
	})
	runMap(b, log)
}

func BenchmarkLoglayer_SimpleMessage(b *testing.B) {
	log := loglayer.New(loglayer.Config{DisableFatalExit: true, Transport: &noopTransport{}})
	runSimple(b, log)
}

func BenchmarkLoglayer_MapMetadata(b *testing.B) {
	log := loglayer.New(loglayer.Config{DisableFatalExit: true, Transport: &noopTransport{}})
	runMap(b, log)
}

func BenchmarkLoglayer_StructMetadata(b *testing.B) {
	log := loglayer.New(loglayer.Config{DisableFatalExit: true, Transport: &noopTransport{}})
	runStruct(b, log)
}

func BenchmarkLoglayer_WithFields(b *testing.B) {
	log := loglayer.New(loglayer.Config{DisableFatalExit: true, Transport: &noopTransport{}})
	log = log.WithFields(loglayer.Fields{"requestId": "abc-123", "service": "api"})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.Info("request handled")
	}
}

func BenchmarkLoglayer_WithError(b *testing.B) {
	log := loglayer.New(loglayer.Config{DisableFatalExit: true, Transport: &noopTransport{}})
	err := benchErr("something went wrong")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.WithError(err).Error("operation failed")
	}
}

// Custom ErrorSerializer path. The default serializer builds a single
// {"message": err.Error()} map. A custom one runs user code on every
// error-bearing entry; this benchmark measures the indirection cost.
func BenchmarkLoglayer_WithError_CustomSerializer(b *testing.B) {
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transport:        &noopTransport{},
		ErrorSerializer: func(err error) map[string]any {
			return map[string]any{"message": err.Error(), "kind": "bench"}
		},
	})
	err := benchErr("something went wrong")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.WithError(err).Error("operation failed")
	}
}

// Plugin pipeline: every dispatch-time hook fires once per emission.
// Measures the per-hook overhead so a regression in the dispatch
// loop's plugin walk shows up here. The plugins themselves are
// trivial; the cost is the framework's per-hook iteration and
// recover() defer.
func BenchmarkLoglayer_PluginPipeline(b *testing.B) {
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transport:        &noopTransport{},
		Plugins: []loglayer.Plugin{
			loglayer.NewDataHook("tag", func(p loglayer.BeforeDataOutParams) loglayer.Data {
				return loglayer.Data{"tagged": true}
			}),
			loglayer.NewLevelHook("level-passthrough", func(p loglayer.TransformLogLevelParams) (loglayer.LogLevel, bool) {
				return 0, false
			}),
			loglayer.NewSendGate("send-all", func(p loglayer.ShouldSendParams) bool { return true }),
		},
	})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.Info("traffic")
	}
}

type benchErr string

func (e benchErr) Error() string { return string(e) }
