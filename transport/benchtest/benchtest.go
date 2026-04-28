// Package benchtest provides shared benchmark helpers for LogLayer
// transport and framework benchmarks. Mirrors the role of
// transport/transporttest for contract tests: lets every wrapper-module
// bench file share a single set of fixtures and runner shapes so the
// numbers are directly comparable and so per-module copies can't drift.
package benchtest

import (
	"testing"

	"go.loglayer.dev"
)

// Msg is the standard benchmark message. Use it as the argument to
// every Info call inside a runner so the per-call payload is constant
// across modules.
const Msg = "user logged in"

// User is the standard benchmark struct. Use [TestUser] for the
// pre-populated value; declare a fresh one only when you need to vary
// the field shapes for a specific scenario.
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// TestUser is the standard benchmark value passed to every struct-
// metadata runner.
var TestUser = User{ID: 42, Name: "Alice", Email: "alice@example.com"}

// Metadata returns the standard benchmark map[string]any. A fresh map
// is returned each call so map-mutating plugins under benchmark don't
// pollute later iterations.
func Metadata() loglayer.Metadata {
	return loglayer.Metadata{
		"id":    42,
		"name":  "Alice",
		"email": "alice@example.com",
	}
}

// Discard is a no-op io.Writer suitable for any transport whose
// throughput is being measured. We use this in place of io.Discard
// because some libraries (charmbracelet/log) detect io.Discard and
// short-circuit their formatting pipeline; that would understate their
// real cost. Discard looks like any other writer so every library
// exercises its full write path.
var Discard discardWriter

type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }

// RunSimple drives the simple-message path. The bench reports allocs
// and resets the timer before the loop.
func RunSimple(b *testing.B, log *loglayer.LogLayer) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.Info(Msg)
	}
}

// RunMap drives the map-metadata path with a fresh [Metadata] each
// iteration.
func RunMap(b *testing.B, log *loglayer.LogLayer) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.WithMetadata(Metadata()).Info(Msg)
	}
}

// RunStruct drives the struct-metadata path using [TestUser].
func RunStruct(b *testing.B, log *loglayer.LogLayer) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		log.WithMetadata(TestUser).Info(Msg)
	}
}
