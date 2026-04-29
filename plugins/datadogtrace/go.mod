module go.loglayer.dev/plugins/datadogtrace

go 1.25.0

replace (
	go.loglayer.dev => ../..
	go.loglayer.dev/plugins/plugintest => ../plugintest
	go.loglayer.dev/transports/testing => ../../transports/testing
)

require (
	go.loglayer.dev v0.0.0-00010101000000-000000000000
	go.loglayer.dev/plugins/plugintest v0.0.0-00010101000000-000000000000
)

require (
	github.com/goccy/go-json v0.10.6 // indirect
	go.loglayer.dev/transports/testing v0.0.0-00010101000000-000000000000 // indirect
)
