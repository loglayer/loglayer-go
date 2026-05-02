package transport_test

import (
	"sync"
	"testing"

	"go.loglayer.dev/v2"
	"go.loglayer.dev/v2/internal/lltest"
	"go.loglayer.dev/v2/transport"
)

// TestSetEnabledNoRace exercises the contract that BaseTransport.SetEnabled
// is safe to call concurrently with emission. The -race detector runs this
// test; if SetEnabled isn't atomic, this test must fail.
//
// Uses internal/lltest as the sink because the public transport modules
// (blank, testing, etc.) live outside main and importing them here would
// create a require cycle.
func TestSetEnabledNoRace(t *testing.T) {
	tr := lltest.New(lltest.Config{
		BaseConfig: transport.BaseConfig{ID: "b"},
		Library:    &lltest.TestLoggingLibrary{},
	})
	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 5000; i++ {
			log.Info("hi")
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 5000; i++ {
			tr.SetEnabled(i%2 == 0)
		}
	}()
	wg.Wait()
}
