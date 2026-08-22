module go.loglayer.dev/plugins/datadogtrace/v2

go 1.25.0

require (
	go.loglayer.dev/plugins/plugintest/v2 v2.0.1
	go.loglayer.dev/transports/testing/v3 v3.0.0-00010101000000-000000000000
	go.loglayer.dev/v3 v3.0.0
)

require (
	github.com/goccy/go-json v0.10.6 // indirect
	go.loglayer.dev/transports/testing/v2 v2.0.1 // indirect
	go.loglayer.dev/v2 v2.0.1 // indirect
)

replace go.loglayer.dev/transports/testing/v3 => ../../transports/testing
