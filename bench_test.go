package loglayer_test

// Benchmarks comparing loglayer (with each transport) against the underlying
// libraries used directly. Three groups of benchmarks:
//
//   1. Direct/*  - the underlying library used directly, writing to a discard
//                  sink. Baseline cost of each library on its own.
//   2. Wrapped/* - LogLayer wrapping the same library via the wrapper transport,
//                  writing to the same discard sink. The delta vs Direct is the
//                  overhead the wrapper adds.
//   3. Render/*  - LogLayer with a renderer transport (structured, console,
//                  pretty, testing) writing to the discard sink. Self-contained
//                  formatters; no underlying library to compare against.
//
// Plus the original Loglayer/* group using a no-op transport so I/O cost is
// excluded entirely. That isolates the pure assembly + dispatch cost in the core.
//
// Note on the writer: we use a custom discardWriter rather than io.Discard
// because charmbracelet/log detects io.Discard and skips its formatting
// pipeline entirely, which would understate its real cost. discardWriter
// looks like any other writer so every library exercises its full write path.
//
// Run with:
//   go test -bench=. -benchmem -run=^$ -benchtime=1s .

import (
	"testing"

	clog "github.com/charmbracelet/log"
	plog "github.com/phuslu/log"
	"github.com/rs/zerolog"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
	"go.loglayer.dev/transports/charmlog"
	"go.loglayer.dev/transports/console"
	llogrus "go.loglayer.dev/transports/logrus"
	"go.loglayer.dev/transports/phuslu"
	"go.loglayer.dev/transports/pretty"
	"go.loglayer.dev/transports/structured"
	lltest "go.loglayer.dev/transports/testing"
	llzap "go.loglayer.dev/transports/zap"
	llzero "go.loglayer.dev/transports/zerolog"
)

type benchUser struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

var benchTestUser = benchUser{ID: 42, Name: "Alice", Email: "alice@example.com"}

const benchMsg = "user logged in"

// discardWriter is an opaque writer that does no work. Used in place of
// io.Discard so libraries that special-case io.Discard (charmbracelet/log)
// can't bypass their formatting pipeline.
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

func BenchmarkDirect_Zerolog_SimpleMessage(b *testing.B) {
	log := zerolog.New(discard)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.Info().Msg(benchMsg)
	}
}

func BenchmarkDirect_Zerolog_MapFields(b *testing.B) {
	log := zerolog.New(discard)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.Info().
			Int("id", 42).
			Str("name", "Alice").
			Str("email", "alice@example.com").
			Msg(benchMsg)
	}
}

func BenchmarkDirect_Zap_SimpleMessage(b *testing.B) {
	log := newDirectZap()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.Info(benchMsg)
	}
}

func BenchmarkDirect_Zap_MapFields(b *testing.B) {
	log := newDirectZap()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.Info(benchMsg,
			zap.Int("id", 42),
			zap.String("name", "Alice"),
			zap.String("email", "alice@example.com"),
		)
	}
}

func BenchmarkDirect_Phuslu_SimpleMessage(b *testing.B) {
	log := newDirectPhuslu()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.Info().Msg(benchMsg)
	}
}

func BenchmarkDirect_Phuslu_MapFields(b *testing.B) {
	log := newDirectPhuslu()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.Info().
			Int("id", 42).
			Str("name", "Alice").
			Str("email", "alice@example.com").
			Msg(benchMsg)
	}
}

func BenchmarkDirect_Logrus_SimpleMessage(b *testing.B) {
	log := newDirectLogrus()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.Info(benchMsg)
	}
}

func BenchmarkDirect_Logrus_MapFields(b *testing.B) {
	log := newDirectLogrus()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.WithFields(logrus.Fields{
			"id":    42,
			"name":  "Alice",
			"email": "alice@example.com",
		}).Info(benchMsg)
	}
}

func BenchmarkDirect_Charmlog_SimpleMessage(b *testing.B) {
	log := newDirectCharmlog()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.Info(benchMsg)
	}
}

func BenchmarkDirect_Charmlog_MapFields(b *testing.B) {
	log := newDirectCharmlog()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.Info(benchMsg, "id", 42, "name", "Alice", "email", "alice@example.com")
	}
}

func BenchmarkWrapped_Zerolog_SimpleMessage(b *testing.B) {
	log := newWrappedZerolog()
	runSimple(b, log)
}

func BenchmarkWrapped_Zerolog_MapMetadata(b *testing.B) {
	log := newWrappedZerolog()
	runMap(b, log)
}

func BenchmarkWrapped_Zerolog_StructMetadata(b *testing.B) {
	log := newWrappedZerolog()
	runStruct(b, log)
}

func BenchmarkWrapped_Zap_SimpleMessage(b *testing.B) {
	log := newWrappedZap()
	runSimple(b, log)
}

func BenchmarkWrapped_Zap_MapMetadata(b *testing.B) {
	log := newWrappedZap()
	runMap(b, log)
}

func BenchmarkWrapped_Zap_StructMetadata(b *testing.B) {
	log := newWrappedZap()
	runStruct(b, log)
}

func BenchmarkWrapped_Phuslu_SimpleMessage(b *testing.B) {
	log := newWrappedPhuslu()
	runSimple(b, log)
}

func BenchmarkWrapped_Phuslu_MapMetadata(b *testing.B) {
	log := newWrappedPhuslu()
	runMap(b, log)
}

func BenchmarkWrapped_Phuslu_StructMetadata(b *testing.B) {
	log := newWrappedPhuslu()
	runStruct(b, log)
}

func BenchmarkWrapped_Logrus_SimpleMessage(b *testing.B) {
	log := newWrappedLogrus()
	runSimple(b, log)
}

func BenchmarkWrapped_Logrus_MapMetadata(b *testing.B) {
	log := newWrappedLogrus()
	runMap(b, log)
}

func BenchmarkWrapped_Logrus_StructMetadata(b *testing.B) {
	log := newWrappedLogrus()
	runStruct(b, log)
}

func BenchmarkWrapped_Charmlog_SimpleMessage(b *testing.B) {
	log := newWrappedCharmlog()
	runSimple(b, log)
}

func BenchmarkWrapped_Charmlog_MapMetadata(b *testing.B) {
	log := newWrappedCharmlog()
	runMap(b, log)
}

func BenchmarkWrapped_Charmlog_StructMetadata(b *testing.B) {
	log := newWrappedCharmlog()
	runStruct(b, log)
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

func BenchmarkRender_Pretty_SimpleMessage(b *testing.B) {
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transport: pretty.New(pretty.Config{
			BaseConfig: transport.BaseConfig{ID: "pretty"},
			Writer:     discard,
			NoColor:    true,
		}),
	})
	runSimple(b, log)
}

func BenchmarkRender_Pretty_MapMetadata(b *testing.B) {
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transport: pretty.New(pretty.Config{
			BaseConfig: transport.BaseConfig{ID: "pretty"},
			Writer:     discard,
			NoColor:    true,
		}),
	})
	runMap(b, log)
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

func newDirectZap() *zap.Logger {
	enc := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	core := zapcore.NewCore(enc, zapcore.AddSync(discard), zapcore.InfoLevel)
	return zap.New(core)
}

func newDirectPhuslu() *plog.Logger {
	return &plog.Logger{
		Level:  plog.InfoLevel,
		Writer: &plog.IOWriter{Writer: discard},
	}
}

func newDirectLogrus() *logrus.Logger {
	l := logrus.New()
	l.Out = discard
	l.Formatter = &logrus.JSONFormatter{}
	l.Level = logrus.InfoLevel
	return l
}

func newDirectCharmlog() *clog.Logger {
	return clog.NewWithOptions(discard, clog.Options{
		Level:     clog.InfoLevel,
		Formatter: clog.JSONFormatter,
	})
}

func newWrappedZerolog() *loglayer.LogLayer {
	z := zerolog.New(discard)
	return loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transport: llzero.New(llzero.Config{
			BaseConfig: transport.BaseConfig{ID: "zerolog"},
			Logger:     &z,
		}),
	})
}

func newWrappedZap() *loglayer.LogLayer {
	return loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transport: llzap.New(llzap.Config{
			BaseConfig: transport.BaseConfig{ID: "zap"},
			Logger:     newDirectZap(),
		}),
	})
}

func newWrappedPhuslu() *loglayer.LogLayer {
	return loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transport: phuslu.New(phuslu.Config{
			BaseConfig: transport.BaseConfig{ID: "phuslu"},
			Logger:     newDirectPhuslu(),
		}),
	})
}

func newWrappedLogrus() *loglayer.LogLayer {
	return loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transport: llogrus.New(llogrus.Config{
			BaseConfig: transport.BaseConfig{ID: "logrus"},
			Logger:     newDirectLogrus(),
		}),
	})
}

func newWrappedCharmlog() *loglayer.LogLayer {
	return loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transport: charmlog.New(charmlog.Config{
			BaseConfig: transport.BaseConfig{ID: "charmlog"},
			Logger:     newDirectCharmlog(),
		}),
	})
}

type benchErr string

func (e benchErr) Error() string { return string(e) }
