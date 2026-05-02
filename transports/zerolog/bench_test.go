package zerolog_test

import (
	"testing"

	zlog "github.com/rs/zerolog"

	llzero "go.loglayer.dev/transports/zerolog/v2"
	"go.loglayer.dev/v2"
	"go.loglayer.dev/v2/transport"
	"go.loglayer.dev/v2/transport/benchtest"
)

func newWrapped() *loglayer.LogLayer {
	z := zlog.New(benchtest.Discard)
	return loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transport: llzero.New(llzero.Config{
			BaseConfig: transport.BaseConfig{ID: "zerolog"},
			Logger:     &z,
		}),
	})
}

func BenchmarkDirect_Zerolog_SimpleMessage(b *testing.B) {
	log := zlog.New(benchtest.Discard)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.Info().Msg(benchtest.Msg)
	}
}

func BenchmarkDirect_Zerolog_MapFields(b *testing.B) {
	log := zlog.New(benchtest.Discard)
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

func BenchmarkWrapped_Zerolog_SimpleMessage(b *testing.B)  { benchtest.RunSimple(b, newWrapped()) }
func BenchmarkWrapped_Zerolog_MapMetadata(b *testing.B)    { benchtest.RunMap(b, newWrapped()) }
func BenchmarkWrapped_Zerolog_StructMetadata(b *testing.B) { benchtest.RunStruct(b, newWrapped()) }
