package charmlog_test

import (
	"testing"

	clog "github.com/charmbracelet/log"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
	llcharm "go.loglayer.dev/transports/charmlog"
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

// discardWriter is opaque so charmlog can't bypass its formatter via
// the io.Discard fast-path.
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

func newDirect() *clog.Logger {
	return clog.NewWithOptions(discard, clog.Options{
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
		log.Info(benchMsg)
	}
}

func BenchmarkDirect_Charmlog_MapFields(b *testing.B) {
	log := newDirect()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.Info(benchMsg, "id", 42, "name", "Alice", "email", "alice@example.com")
	}
}

func BenchmarkWrapped_Charmlog_SimpleMessage(b *testing.B)  { runSimple(b, newWrapped()) }
func BenchmarkWrapped_Charmlog_MapMetadata(b *testing.B)    { runMap(b, newWrapped()) }
func BenchmarkWrapped_Charmlog_StructMetadata(b *testing.B) { runStruct(b, newWrapped()) }
