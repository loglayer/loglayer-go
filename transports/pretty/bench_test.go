package pretty_test

import (
	"testing"

	"go.loglayer.dev/transports/pretty/v2"
	"go.loglayer.dev/v2"
	"go.loglayer.dev/v2/transport"
	"go.loglayer.dev/v2/transport/benchtest"
)

func newPrettyLogger() *loglayer.LogLayer {
	return loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transport: pretty.New(pretty.Config{
			BaseConfig: transport.BaseConfig{ID: "pretty"},
			Writer:     benchtest.Discard,
			NoColor:    true,
		}),
	})
}

func BenchmarkRender_Pretty_SimpleMessage(b *testing.B) {
	benchtest.RunSimple(b, newPrettyLogger())
}

func BenchmarkRender_Pretty_MapMetadata(b *testing.B) {
	benchtest.RunMap(b, newPrettyLogger())
}

func BenchmarkRender_Pretty_StructMetadata(b *testing.B) {
	benchtest.RunStruct(b, newPrettyLogger())
}
