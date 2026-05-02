package testing_test

// Render-path benchmarks for the lltest transport. Lives here rather
// than in the main module's bench_test.go because main can't import this
// package without a require cycle.

import (
	"testing"

	lltest "go.loglayer.dev/transports/testing/v2"
	"go.loglayer.dev/v2"
	"go.loglayer.dev/v2/transport"
	"go.loglayer.dev/v2/transport/benchtest"
)

func BenchmarkRender_Testing_SimpleMessage(b *testing.B) {
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transport: lltest.New(lltest.Config{
			BaseConfig: transport.BaseConfig{ID: "test"},
		}),
	})
	benchtest.RunSimple(b, log)
}

func BenchmarkRender_Testing_MapMetadata(b *testing.B) {
	log := loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transport: lltest.New(lltest.Config{
			BaseConfig: transport.BaseConfig{ID: "test"},
		}),
	})
	benchtest.RunMap(b, log)
}
