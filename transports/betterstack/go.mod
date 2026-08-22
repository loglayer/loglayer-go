module go.loglayer.dev/transports/betterstack

go 1.25.0

require (
	github.com/goccy/go-json v0.10.6
	go.loglayer.dev/transports/http/v3 v3.0.0-00010101000000-000000000000
	go.loglayer.dev/v3 v3.0.0
)

replace go.loglayer.dev => ../..

replace go.loglayer.dev/transports/http => ../http

replace go.loglayer.dev/transports/http/v3 => ../http
