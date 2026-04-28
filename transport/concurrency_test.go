package transport_test

import (
	"sync"
	"testing"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
	"go.loglayer.dev/transports/blank"
)

// TestSetEnabledNoRace exercises the contract that BaseTransport.SetEnabled
// is safe to call concurrently with emission. The -race detector runs this
// test; if SetEnabled isn't atomic, this test must fail.
func TestSetEnabledNoRace(t *testing.T) {
	tr := blank.New(blank.Config{BaseConfig: transport.BaseConfig{ID: "b"}})
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
