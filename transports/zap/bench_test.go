package zap_test

import (
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	llzap "go.loglayer.dev/transports/zap/v2"
	"go.loglayer.dev/v2"
	"go.loglayer.dev/v2/transport"
	"go.loglayer.dev/v2/transport/benchtest"
)

func newDirect() *zap.Logger {
	enc := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	core := zapcore.NewCore(enc, zapcore.AddSync(benchtest.Discard), zapcore.InfoLevel)
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
		log.Info(benchtest.Msg)
	}
}

func BenchmarkDirect_Zap_MapFields(b *testing.B) {
	log := newDirect()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.Info(benchtest.Msg,
			zap.Int("id", 42),
			zap.String("name", "Alice"),
			zap.String("email", "alice@example.com"),
		)
	}
}

func BenchmarkWrapped_Zap_SimpleMessage(b *testing.B)  { benchtest.RunSimple(b, newWrapped()) }
func BenchmarkWrapped_Zap_MapMetadata(b *testing.B)    { benchtest.RunMap(b, newWrapped()) }
func BenchmarkWrapped_Zap_StructMetadata(b *testing.B) { benchtest.RunStruct(b, newWrapped()) }
