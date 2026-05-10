package newrelic_test

import (
	"testing"

	"go.uber.org/goleak"
)

// TestMain wraps the suite in goleak.VerifyTestMain to catch goroutine
// leaks. The New Relic transport wraps an HTTP transport that spawns
// a worker goroutine per Transport via the internal async worker; tests
// must call tr.Close() to shut it down.
//
// httptest's connection-shutdown goroutines occasionally outlive the
// test cleanup, so we ignore the http.(*Server).Shutdown stacks. Any
// other unexpected goroutine fails the suite.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		goleak.IgnoreTopFunction("net/http.(*Server).Shutdown"),
		goleak.IgnoreAnyFunction("net/http.(*persistConn).readLoop"),
		goleak.IgnoreAnyFunction("net/http.(*persistConn).writeLoop"),
	)
}
