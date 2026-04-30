package datadog_test

import (
	"testing"

	"go.uber.org/goleak"
)

// TestMain wraps the suite in goleak.VerifyTestMain to catch goroutine
// leaks. The Datadog transport spawns an HTTP-style worker for batched
// shipping; tests must call tr.Close() to shut it down.
//
// HTTP connection-pool goroutines occasionally outlive the test
// cleanup; the persist-conn (HTTP/1.1) and http2ClientConn (HTTP/2)
// read/write loops are ignored. Live tests against us3/eu1/ap1 hit
// HTTP/2 endpoints, which is why the http2 ignore is needed.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		goleak.IgnoreTopFunction("net/http.(*Server).Shutdown"),
		goleak.IgnoreAnyFunction("net/http.(*persistConn).readLoop"),
		goleak.IgnoreAnyFunction("net/http.(*persistConn).writeLoop"),
		goleak.IgnoreAnyFunction("net/http.(*http2ClientConn).readLoop"),
	)
}
