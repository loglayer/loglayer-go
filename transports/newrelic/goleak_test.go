package newrelic_test

import (
	"testing"

	"go.uber.org/goleak"
)

// TestMain wraps the suite in goleak.VerifyTestMain to catch goroutine
// leaks. The transport spawns an HTTP worker; tests must call tr.Close()
// to shut it down. HTTP connection-pool goroutines are ignored because
// they outlive normal test cleanup.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		goleak.IgnoreTopFunction("net/http.(*Server).Shutdown"),
		goleak.IgnoreAnyFunction("net/http.(*persistConn).readLoop"),
		goleak.IgnoreAnyFunction("net/http.(*persistConn).writeLoop"),
		goleak.IgnoreAnyFunction("net/http.(*http2ClientConn).readLoop"),
	)
}
