package datadog_test

import (
	"testing"

	"go.uber.org/goleak"
)

// TestMain wraps the suite in goleak.VerifyTestMain to catch goroutine
// leaks. The Datadog transport spawns an HTTP-style worker for batched
// shipping; tests must call tr.Close() to shut it down.
//
// httptest's connection-shutdown goroutines occasionally outlive the
// test cleanup, so the http.* persist-conn stacks are filtered out.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		goleak.IgnoreTopFunction("net/http.(*Server).Shutdown"),
		goleak.IgnoreAnyFunction("net/http.(*persistConn).readLoop"),
		goleak.IgnoreAnyFunction("net/http.(*persistConn).writeLoop"),
	)
}
