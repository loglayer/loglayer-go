package sloghandler_test

import (
	"context"
	"log/slog"
	"testing"

	"go.loglayer.dev"
	"go.loglayer.dev/integrations/sloghandler"
)

// noopTransport drops everything; used to isolate handler dispatch cost
// from any rendering work.
type noopTransport struct{}

func (n *noopTransport) ID() string                              { return "noop" }
func (n *noopTransport) IsEnabled() bool                         { return true }
func (n *noopTransport) SendToLogger(_ loglayer.TransportParams) {}
func (n *noopTransport) GetLoggerInstance() any                  { return nil }

func newBenchLogger() *slog.Logger {
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transport:        &noopTransport{},
	})
	return slog.New(sloghandler.New(log))
}

// Bare slog.Info path: no attrs, no groups, no scope. The dominant cost
// here is the slog → loglayer translation plus the standard dispatch.
func BenchmarkSlogHandler_SimpleMessage(b *testing.B) {
	l := newBenchLogger()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Info("simple")
	}
}

// Inline attrs become per-call fields (one slog.Attr → one Fields entry).
// This pair plus the simple bench tells you "how much do attrs cost on
// the slog path vs no attrs."
func BenchmarkSlogHandler_ThreeAttrs(b *testing.B) {
	l := newBenchLogger()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Info("attrs",
			slog.String("requestId", "abc"),
			slog.Int("status", 200),
			slog.String("path", "/users"),
		)
	}
}

// Handler.WithAttrs: persistent attrs accumulated on the derived
// handler. Once installed, every emission carries them. Measures the
// per-emission cost of folding the persisted attrs in (without rebuilding
// the handler chain).
func BenchmarkSlogHandler_PersistentAttrs(b *testing.B) {
	l := newBenchLogger().With(
		"app", "api",
		"env", "prod",
	)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Info("persistent")
	}
}

// WithGroup nests subsequent attrs under a named map. Adds one Map
// allocation per emission (the nested group map). Tells you the
// overhead of slog's group-namespacing feature.
func BenchmarkSlogHandler_WithGroup(b *testing.B) {
	l := newBenchLogger().WithGroup("http")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Info("served", "method", "GET", "path", "/users")
	}
}

// LogValuer is resolved at emission time; this benches the resolution
// cost vs a plain attr.
type benchValuer struct{ id int }

func (v benchValuer) LogValue() slog.Value { return slog.IntValue(v.id) }

func BenchmarkSlogHandler_LogValuer(b *testing.B) {
	l := newBenchLogger()
	v := benchValuer{id: 42}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Info("valuer", "user", v)
	}
}

// InfoContext path: ctx forwarded to dispatch-time plugin hooks. With
// no plugins registered, this should be ~free vs Info.
func BenchmarkSlogHandler_InfoContext(b *testing.B) {
	l := newBenchLogger()
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.InfoContext(ctx, "with-ctx")
	}
}

// Source forwarding: slog.New always populates Record.PC, so the handler
// always calls SourceFromPC. Confirms that overhead with a baseline
// comparable to BenchmarkSlogHandler_SimpleMessage on the same hardware.
// (No "off" variant: there's no slog.HandlerOptions equivalent on this
// side; the PC is captured by slog regardless.)
func BenchmarkSlogHandler_WithSource(b *testing.B) {
	l := newBenchLogger()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Info("with-source")
	}
}
