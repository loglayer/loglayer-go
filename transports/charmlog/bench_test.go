package charmlog_test

import (
	"testing"

	clog "github.com/charmbracelet/log"

	llcharm "go.loglayer.dev/transports/charmlog/v2"
	"go.loglayer.dev/v2"
	"go.loglayer.dev/v2/transport"
	"go.loglayer.dev/v2/transport/benchtest"
)

func newDirect() *clog.Logger {
	return clog.NewWithOptions(benchtest.Discard, clog.Options{
		Level:     clog.InfoLevel,
		Formatter: clog.JSONFormatter,
	})
}

func newWrapped() *loglayer.LogLayer {
	return loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transport: llcharm.New(llcharm.Config{
			BaseConfig: transport.BaseConfig{ID: "charmlog"},
			Logger:     newDirect(),
		}),
	})
}

func BenchmarkDirect_Charmlog_SimpleMessage(b *testing.B) {
	log := newDirect()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.Info(benchtest.Msg)
	}
}

func BenchmarkDirect_Charmlog_MapFields(b *testing.B) {
	log := newDirect()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.Info(benchtest.Msg, "id", 42, "name", "Alice", "email", "alice@example.com")
	}
}

func BenchmarkWrapped_Charmlog_SimpleMessage(b *testing.B)  { benchtest.RunSimple(b, newWrapped()) }
func BenchmarkWrapped_Charmlog_MapMetadata(b *testing.B)    { benchtest.RunMap(b, newWrapped()) }
func BenchmarkWrapped_Charmlog_StructMetadata(b *testing.B) { benchtest.RunStruct(b, newWrapped()) }
