package structured_test

// Render-path benchmarks for the structured transport. Lives here rather
// than in the main module's bench_test.go because main can't import this
// package without a require cycle.

import (
	"testing"

	"go.loglayer.dev/transports/structured/v2"
	"go.loglayer.dev/v2"
	"go.loglayer.dev/v2/transport"
	"go.loglayer.dev/v2/transport/benchtest"
)

func BenchmarkRender_Structured_SimpleMessage(b *testing.B) {
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transport: structured.New(structured.Config{
			BaseConfig: transport.BaseConfig{ID: "structured"},
			Writer:     benchtest.Discard,
		}),
	})
	benchtest.RunSimple(b, log)
}

func BenchmarkRender_Structured_MapMetadata(b *testing.B) {
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transport: structured.New(structured.Config{
			BaseConfig: transport.BaseConfig{ID: "structured"},
			Writer:     benchtest.Discard,
		}),
	})
	benchtest.RunMap(b, log)
}

func BenchmarkRender_Structured_StructMetadata(b *testing.B) {
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transport: structured.New(structured.Config{
			BaseConfig: transport.BaseConfig{ID: "structured"},
			Writer:     benchtest.Discard,
		}),
	})
	benchtest.RunStruct(b, log)
}
