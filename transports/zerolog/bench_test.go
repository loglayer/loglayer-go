package zerolog_test

import (
	"testing"

	zlog "github.com/rs/zerolog"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
	llzero "go.loglayer.dev/transports/zerolog"
)

const benchMsg = "user logged in"

type benchUser struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

var benchTestUser = benchUser{ID: 42, Name: "Alice", Email: "alice@example.com"}

func benchMetadata() loglayer.Metadata {
	return loglayer.Metadata{
		"id":    42,
		"name":  "Alice",
		"email": "alice@example.com",
	}
}

// discardWriter is an opaque writer; charmbracelet/log special-cases
// io.Discard, and we want every library to exercise its full write path.
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

func runStruct(b *testing.B, log *loglayer.LogLayer) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.WithMetadata(benchTestUser).Info(benchMsg)
	}
}

func newWrapped() *loglayer.LogLayer {
	z := zlog.New(discard)
	return loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transport: llzero.New(llzero.Config{
			BaseConfig: transport.BaseConfig{ID: "zerolog"},
			Logger:     &z,
		}),
	})
}

func BenchmarkDirect_Zerolog_SimpleMessage(b *testing.B) {
	log := zlog.New(discard)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.Info().Msg(benchMsg)
	}
}

func BenchmarkDirect_Zerolog_MapFields(b *testing.B) {
	log := zlog.New(discard)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.Info().
			Int("id", 42).
			Str("name", "Alice").
			Str("email", "alice@example.com").
			Msg(benchMsg)
	}
}

func BenchmarkWrapped_Zerolog_SimpleMessage(b *testing.B)  { runSimple(b, newWrapped()) }
func BenchmarkWrapped_Zerolog_MapMetadata(b *testing.B)    { runMap(b, newWrapped()) }
func BenchmarkWrapped_Zerolog_StructMetadata(b *testing.B) { runStruct(b, newWrapped()) }
