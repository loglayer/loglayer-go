module go.loglayer.dev/examples/http-server/v2

go 1.25.0

replace (
	go.loglayer.dev/integrations/loghttp/v2 => ../../integrations/loghttp
	go.loglayer.dev/transports/structured/v3 => ../../transports/structured
	go.loglayer.dev/transports/testing/v3 => ../../transports/testing
	go.loglayer.dev/v3 => ../..
)

require (
	go.loglayer.dev/transports/structured/v3 v3.0.0-00010101000000-000000000000
	go.loglayer.dev/v3 v3.0.0
)

require github.com/goccy/go-json v0.10.6 // indirect
