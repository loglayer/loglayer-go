module go.loglayer.dev/transports/central/v2

go 1.25.0

replace go.loglayer.dev/v2 => ../..

replace go.loglayer.dev/transports/http/v2 => ../http

require (
	github.com/goccy/go-json v0.10.6
	go.loglayer.dev/v2 v2.0.0-00010101000000-000000000000
	go.loglayer.dev/transports/http/v2 v2.0.0-00010101000000-000000000000
	go.uber.org/goleak v1.3.0
)
