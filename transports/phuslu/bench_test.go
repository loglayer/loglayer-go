package phuslu_test

import (
	"testing"

	plog "github.com/phuslu/log"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
	llphuslu "go.loglayer.dev/transports/phuslu"
)

const benchMsg = "user logged in"

type benchUser struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

var benchTestUser = benchUser{ID: 42, Name: "Alice", Email: "alice@example.com"}

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

func runStruct(b *testing.B, log *loglayer.LogLayer) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.WithMetadata(benchTestUser).Info(benchMsg)
	}
}

func newDirect() *plog.Logger {
	return &plog.Logger{
		Level:  plog.InfoLevel,
		Writer: &plog.IOWriter{Writer: discard},
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
		log.Info().Msg(benchMsg)
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
			Msg(benchMsg)
	}
}

func BenchmarkWrapped_Phuslu_SimpleMessage(b *testing.B)  { runSimple(b, newWrapped()) }
func BenchmarkWrapped_Phuslu_MapMetadata(b *testing.B)    { runMap(b, newWrapped()) }
func BenchmarkWrapped_Phuslu_StructMetadata(b *testing.B) { runStruct(b, newWrapped()) }
