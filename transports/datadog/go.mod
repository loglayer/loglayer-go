module go.loglayer.dev/transports/datadog

go 1.25.0

replace go.loglayer.dev => ../..

replace go.loglayer.dev/transports/http => ../http

require (
	github.com/goccy/go-json v0.10.6
	go.loglayer.dev v0.0.0-00010101000000-000000000000
	go.loglayer.dev/transports/http v0.0.0-00010101000000-000000000000
	go.uber.org/goleak v1.3.0
)
