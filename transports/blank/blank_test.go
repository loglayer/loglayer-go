package blank_test

import (
	"sync/atomic"
	"testing"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
	"go.loglayer.dev/transports/blank"
)

func TestBlank_CallsShipToLogger(t *testing.T) {
	var calls int
	var captured loglayer.TransportParams

	tr := blank.New(blank.Config{
		BaseConfig: transport.BaseConfig{ID: "blank"},
		ShipToLogger: func(p loglayer.TransportParams) {
			calls++
			captured = p
		},
	})

	log := loglayer.New(loglayer.Config{
		Transport:        tr,
		DisableFatalExit: true,
	})

	log.WithMetadata(loglayer.Metadata{"k": "v"}).Info("hello")

	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
	if captured.LogLevel != loglayer.LogLevelInfo {
		t.Errorf("level: got %s", captured.LogLevel)
	}
	if m, _ := captured.Metadata.(loglayer.Metadata); m["k"] != "v" {
		t.Errorf("metadata: got %v", captured.Metadata)
	}
}

func TestBlank_NilFnSilentlyDrops(t *testing.T) {
	tr := blank.New(blank.Config{
		BaseConfig: transport.BaseConfig{ID: "blank"},
	})
	log := loglayer.New(loglayer.Config{
		Transport:        tr,
		DisableFatalExit: true,
	})
	// Should not panic.
	log.Info("dropped")
	log.WithMetadata(loglayer.Metadata{"k": 1}).Error("also dropped")
}

func TestBlank_HonorsLevelFilter(t *testing.T) {
	var calls int
	tr := blank.New(blank.Config{
		BaseConfig: transport.BaseConfig{ID: "blank", Level: loglayer.LogLevelError},
		ShipToLogger: func(p loglayer.TransportParams) {
			calls++
		},
	})
	log := loglayer.New(loglayer.Config{
		Transport:        tr,
		DisableFatalExit: true,
	})
	log.Info("filtered")
	log.Warn("filtered")
	if calls != 0 {
		t.Errorf("expected 0 calls below threshold, got %d", calls)
	}
	log.Error("passes")
	if calls != 1 {
		t.Errorf("expected 1 call at threshold, got %d", calls)
	}
}

func TestBlank_GetLoggerInstanceIsNil(t *testing.T) {
	tr := blank.New(blank.Config{BaseConfig: transport.BaseConfig{ID: "blank"}})
	log := loglayer.New(loglayer.Config{
		Transport:        tr,
		DisableFatalExit: true,
	})
	if got := log.GetLoggerInstance("blank"); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestBlank_ConcurrentSafe(t *testing.T) {
	// Verify the wrapper itself is safe under concurrent emission.
	// (Whether the user's ShipToLogger is concurrency-safe is the user's
	// problem; here we use atomic to keep the test self-contained.)
	var calls int64
	tr := blank.New(blank.Config{
		BaseConfig: transport.BaseConfig{ID: "blank"},
		ShipToLogger: func(p loglayer.TransportParams) {
			atomic.AddInt64(&calls, 1)
		},
	})
	log := loglayer.New(loglayer.Config{
		Transport:        tr,
		DisableFatalExit: true,
	})

	const goroutines = 8
	const perGoroutine = 100
	done := make(chan struct{}, goroutines)
	for g := 0; g < goroutines; g++ {
		go func() {
			defer func() { done <- struct{}{} }()
			for i := 0; i < perGoroutine; i++ {
				log.Info("concurrent")
			}
		}()
	}
	for i := 0; i < goroutines; i++ {
		<-done
	}
	if got := atomic.LoadInt64(&calls); got != int64(goroutines*perGoroutine) {
		t.Errorf("expected %d calls, got %d", goroutines*perGoroutine, got)
	}
}
