// Per-request logger via integrations/loghttp middleware.
//
// Run:
//
//	go run ./examples/http-server
//
// Then in another terminal:
//
//	curl http://localhost:8080/users
//	curl -H 'X-Request-ID: my-trace-id' http://localhost:8080/users
package main

import (
	"net/http"

	"go.loglayer.dev/integrations/loghttp/v2"
	"go.loglayer.dev/transports/structured/v2"
	"go.loglayer.dev/v2"
)

var serverLog = loglayer.New(loglayer.Config{
	Transport: structured.New(structured.Config{}),
})

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/users", usersHandler)
	mux.HandleFunc("/healthz", healthHandler)

	handler := loghttp.Middleware(serverLog, loghttp.Config{})(mux)

	serverLog.Info("listening on :8080")
	_ = http.ListenAndServe(":8080", handler)
}

func usersHandler(w http.ResponseWriter, r *http.Request) {
	// loghttp.FromRequest returns the per-request logger with requestId,
	// method, and path already attached.
	log := loghttp.FromRequest(r)

	log.WithMetadata(loglayer.Metadata{"action": "lookup"}).Info("looking up user")
	_, _ = w.Write([]byte(`{"id":1,"name":"Alice"}`))
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
