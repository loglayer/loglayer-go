package logrus_test

import (
	"testing"

	logrusbase "github.com/sirupsen/logrus"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
	"go.loglayer.dev/transport/benchtest"
	lllogrus "go.loglayer.dev/transports/logrus"
)

func newDirect() *logrusbase.Logger {
	l := logrusbase.New()
	l.Out = benchtest.Discard
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
		log.Info(benchtest.Msg)
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
		}).Info(benchtest.Msg)
	}
}

func BenchmarkWrapped_Logrus_SimpleMessage(b *testing.B)  { benchtest.RunSimple(b, newWrapped()) }
func BenchmarkWrapped_Logrus_MapMetadata(b *testing.B)    { benchtest.RunMap(b, newWrapped()) }
func BenchmarkWrapped_Logrus_StructMetadata(b *testing.B) { benchtest.RunStruct(b, newWrapped()) }
