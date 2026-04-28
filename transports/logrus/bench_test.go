package logrus_test

import (
	"testing"

	logrusbase "github.com/sirupsen/logrus"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
	lllogrus "go.loglayer.dev/transports/logrus"
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

func newDirect() *logrusbase.Logger {
	l := logrusbase.New()
	l.Out = discard
	l.Formatter = &logrusbase.JSONFormatter{}
	l.Level = logrusbase.InfoLevel
	return l
}

func newWrapped() *loglayer.LogLayer {
	return loglayer.New(loglayer.Config{
		DisableFatalExit: true,
		Transport: lllogrus.New(lllogrus.Config{
			BaseConfig: transport.BaseConfig{ID: "logrus"},
			Logger:     newDirect(),
		}),
	})
}

func BenchmarkDirect_Logrus_SimpleMessage(b *testing.B) {
	log := newDirect()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.Info(benchMsg)
	}
}

func BenchmarkDirect_Logrus_MapFields(b *testing.B) {
	log := newDirect()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.WithFields(logrusbase.Fields{
			"id":    42,
			"name":  "Alice",
			"email": "alice@example.com",
		}).Info(benchMsg)
	}
}

func BenchmarkWrapped_Logrus_SimpleMessage(b *testing.B)  { runSimple(b, newWrapped()) }
func BenchmarkWrapped_Logrus_MapMetadata(b *testing.B)    { runMap(b, newWrapped()) }
func BenchmarkWrapped_Logrus_StructMetadata(b *testing.B) { runStruct(b, newWrapped()) }
