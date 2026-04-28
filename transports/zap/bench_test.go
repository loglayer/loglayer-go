package zap_test

import (
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
	llzap "go.loglayer.dev/transports/zap"
)

const benchMsg = "user logged in"

type benchUser struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

var benchTestUser = benchUser{ID: 42, Name: "Alice", Email: "alice@example.com"}

func benchMetadata() loglayer.Metadata {
	return loglayer.Metadata{"id": 42, "name": "Alice", "email": "alice@example.com"}
}

type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }
func (discardWriter) Sync() error                 { return nil }

var discard discardWriter

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

func newDirect() *zap.Logger {
	enc := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	core := zapcore.NewCore(enc, zapcore.AddSync(discard), zapcore.InfoLevel)
	return zap.New(core)
}

func newWrapped() *loglayer.LogLayer {
	return loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transport: llzap.New(llzap.Config{
			BaseConfig: transport.BaseConfig{ID: "zap"},
			Logger:     newDirect(),
		}),
	})
}

func BenchmarkDirect_Zap_SimpleMessage(b *testing.B) {
	log := newDirect()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.Info(benchMsg)
	}
}

func BenchmarkDirect_Zap_MapFields(b *testing.B) {
	log := newDirect()
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

func BenchmarkWrapped_Zap_SimpleMessage(b *testing.B)  { runSimple(b, newWrapped()) }
func BenchmarkWrapped_Zap_MapMetadata(b *testing.B)    { runMap(b, newWrapped()) }
func BenchmarkWrapped_Zap_StructMetadata(b *testing.B) { runStruct(b, newWrapped()) }
