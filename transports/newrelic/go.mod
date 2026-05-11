module go.loglayer.dev/transports/newrelic

go 1.25.0

require (
	github.com/goccy/go-json v0.10.6
	go.loglayer.dev/transports/http/v2 v2.0.1
	go.loglayer.dev/v2 v2.0.1
	go.uber.org/goleak v1.3.0
)

replace go.loglayer.dev => ../..

replace go.loglayer.dev/transports/http/v2 => ../http
