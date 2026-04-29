package transport_test

import (
	"sync"
	"testing"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
	lltest "go.loglayer.dev/transports/testing"
)

// TestSetEnabledNoRace exercises the contract that BaseTransport.SetEnabled
// is safe to call concurrently with emission. The -race detector runs this
// test; if SetEnabled isn't atomic, this test must fail.
//
// Uses the in-process lltest transport (which stays bundled in the main
// module) rather than transports/blank, because blank now lives in its own
// module and importing it here would create a require cycle with main.
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
