// Separate module so the network transports can version independently
// of the framework core (new auth schemes, batching strategies, encoder
// shapes evolve at their own cadence).
module go.loglayer.dev/transports/http

go 1.25.0

replace go.loglayer.dev => ../..

require (
	github.com/goccy/go-json v0.10.6
	go.loglayer.dev v0.0.0-00010101000000-000000000000
	go.uber.org/goleak v1.3.0
)
