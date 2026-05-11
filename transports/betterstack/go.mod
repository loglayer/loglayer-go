module go.loglayer.dev/transports/betterstack

go 1.25.0

require (
	github.com/goccy/go-json v0.10.6
	go.loglayer.dev/transports/http/v2 v2.1.0
	go.loglayer.dev/v2 v2.0.1
)

replace go.loglayer.dev => ../..

replace go.loglayer.dev/transports/http => ../http
