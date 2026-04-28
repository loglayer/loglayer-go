package pretty_test

import (
	"testing"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
	"go.loglayer.dev/transports/pretty"
)

const benchMsg = "user logged in"

func benchMetadata() loglayer.Metadata {
	return loglayer.Metadata{"id": 42, "name": "Alice", "email": "alice@example.com"}
}

type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }

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

func newPrettyLogger() *loglayer.LogLayer {
	return loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transport: pretty.New(pretty.Config{
			BaseConfig: transport.BaseConfig{ID: "pretty"},
			Writer:     discard,
			NoColor:    true,
		}),
	})
}

func BenchmarkRender_Pretty_SimpleMessage(b *testing.B) { runSimple(b, newPrettyLogger()) }
func BenchmarkRender_Pretty_MapMetadata(b *testing.B)   { runMap(b, newPrettyLogger()) }
