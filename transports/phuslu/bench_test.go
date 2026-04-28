package phuslu_test

import (
	"testing"

	plog "github.com/phuslu/log"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
	"go.loglayer.dev/transport/benchtest"
	llphuslu "go.loglayer.dev/transports/phuslu"
)

func newDirect() *plog.Logger {
	return &plog.Logger{
		Level:  plog.InfoLevel,
		Writer: &plog.IOWriter{Writer: benchtest.Discard},
	}
}

func newWrapped() *loglayer.LogLayer {
	return loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transport: llphuslu.New(llphuslu.Config{
			BaseConfig: transport.BaseConfig{ID: "phuslu"},
			Logger:     newDirect(),
		}),
	})
}

func BenchmarkDirect_Phuslu_SimpleMessage(b *testing.B) {
	log := newDirect()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.Info().Msg(benchtest.Msg)
	}
}

func BenchmarkDirect_Phuslu_MapFields(b *testing.B) {
	log := newDirect()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.Info().
			Int("id", 42).
			Str("name", "Alice").
			Str("email", "alice@example.com").
			Msg(benchtest.Msg)
	}
}

func BenchmarkWrapped_Phuslu_SimpleMessage(b *testing.B)  { benchtest.RunSimple(b, newWrapped()) }
func BenchmarkWrapped_Phuslu_MapMetadata(b *testing.B)    { benchtest.RunMap(b, newWrapped()) }
func BenchmarkWrapped_Phuslu_StructMetadata(b *testing.B) { benchtest.RunStruct(b, newWrapped()) }
