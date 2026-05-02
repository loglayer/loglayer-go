package console_test

// Render-path benchmarks for the console transport. Lives here rather
// than in the main module's bench_test.go because main can't import this
// package without a require cycle.

import (
	"testing"

	"go.loglayer.dev/transports/console/v2"
	"go.loglayer.dev/v2"
	"go.loglayer.dev/v2/transport"
	"go.loglayer.dev/v2/transport/benchtest"
)

func BenchmarkRender_Console_SimpleMessage(b *testing.B) {
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transport: console.New(console.Config{
			BaseConfig: transport.BaseConfig{ID: "console"},
			Writer:     benchtest.Discard,
		}),
	})
	benchtest.RunSimple(b, log)
}

func BenchmarkRender_Console_MapMetadata(b *testing.B) {
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transport: console.New(console.Config{
			BaseConfig: transport.BaseConfig{ID: "console"},
			Writer:     benchtest.Discard,
		}),
	})
	benchtest.RunMap(b, log)
}

func BenchmarkRender_Console_StructMetadata(b *testing.B) {
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transport: console.New(console.Config{
			BaseConfig: transport.BaseConfig{ID: "console"},
			Writer:     benchtest.Discard,
		}),
	})
	benchtest.RunStruct(b, log)
}
