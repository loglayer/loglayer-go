package loglayer_test

// Benchmarks for the framework core and shared dispatch paths.
//
// Per-transport render benchmarks (Render_Structured, Render_Console,
// Render_Testing, Render_Pretty, Direct_Zerolog vs Wrapped_Zerolog,
// etc.) live in each transport module's own bench_test.go so the main
// module doesn't have to require every transport (which would create
// require cycles for the transports that depend on main).
//
// Shared fixtures (discard writer, struct/map data, runner shapes)
// live in transport/benchtest so every module's numbers are directly
// comparable.

import (
	"testing"

	"go.loglayer.dev"
	"go.loglayer.dev/transport/benchtest"
)

type noopTransport struct{}

func (n *noopTransport) ID() string                              { return "noop" }
func (n *noopTransport) IsEnabled() bool                         { return true }
func (n *noopTransport) SendToLogger(_ loglayer.TransportParams) {}
func (n *noopTransport) GetLoggerInstance() any                  { return nil }

func BenchmarkLoglayer_SimpleMessage(b *testing.B) {
	log := loglayer.New(loglayer.Config{DisableFatalExit: true, Transport: &noopTransport{}})
	benchtest.RunSimple(b, log)
}

func BenchmarkLoglayer_MapMetadata(b *testing.B) {
	log := loglayer.New(loglayer.Config{DisableFatalExit: true, Transport: &noopTransport{}})
	benchtest.RunMap(b, log)
}

func BenchmarkLoglayer_StructMetadata(b *testing.B) {
	log := loglayer.New(loglayer.Config{DisableFatalExit: true, Transport: &noopTransport{}})
	benchtest.RunStruct(b, log)
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

// AddSource off vs on: measures the cost of runtime.Caller capture.
// The "off" variant should match BenchmarkLoglayer_SimpleMessage; the
// "on" variant adds one runtime.Caller + FuncForPC + a heap-allocated
// *Source per emission.
func BenchmarkLoglayer_SimpleMessage_AddSourceOff(b *testing.B) {
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transport:        &noopTransport{},
	})
	benchtest.RunSimple(b, log)
}

func BenchmarkLoglayer_SimpleMessage_AddSourceOn(b *testing.B) {
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Source:           loglayer.SourceConfig{Enabled: true},
		Transport:        &noopTransport{},
	})
	benchtest.RunSimple(b, log)
}

// Same pair on the metadata path so the relative cost is clear (the
// metadata path already allocates more, so AddSource's overhead should
// be a smaller fraction of the total).
func BenchmarkLoglayer_MapMetadata_AddSourceOff(b *testing.B) {
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transport:        &noopTransport{},
	})
	benchtest.RunMap(b, log)
}

func BenchmarkLoglayer_MapMetadata_AddSourceOn(b *testing.B) {
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Source:           loglayer.SourceConfig{Enabled: true},
		Transport:        &noopTransport{},
	})
	benchtest.RunMap(b, log)
}

type benchErr string

func (e benchErr) Error() string { return string(e) }

// WithFields paired with a Lazy field. Callback returns a cheap
// constant so the measured delta is the framework's resolution
// overhead, not the user payload.
func BenchmarkLoglayer_WithFields_Lazy(b *testing.B) {
	log := loglayer.New(loglayer.Config{DisableFatalExit: true, Transport: &noopTransport{}})
	log = log.WithFields(loglayer.Fields{
		"requestId": "abc-123",
		"computed":  loglayer.Lazy(func() any { return "computed-value" }),
	})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.Info("request handled")
	}
}
