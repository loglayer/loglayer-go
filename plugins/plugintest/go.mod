module go.loglayer.dev/plugins/plugintest/v2

go 1.25.0

replace (
	go.loglayer.dev/v2 => ../..
	go.loglayer.dev/transports/testing/v2 => ../../transports/testing
)

require (
	go.loglayer.dev/v2 v2.0.0-00010101000000-000000000000
	go.loglayer.dev/transports/testing/v2 v2.0.0-00010101000000-000000000000
)

require github.com/goccy/go-json v0.10.6 // indirect
