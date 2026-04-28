module go.loglayer.dev/examples/datadog-shipping

go 1.25.0

replace (
	go.loglayer.dev => ../..
	go.loglayer.dev/transports/datadog => ../../transports/datadog
	go.loglayer.dev/transports/http => ../../transports/http
	go.loglayer.dev/transports/pretty => ../../transports/pretty
)

require (
	go.loglayer.dev v0.0.0-00010101000000-000000000000
	go.loglayer.dev/transports/datadog v0.0.0-00010101000000-000000000000
	go.loglayer.dev/transports/http v0.0.0-00010101000000-000000000000
	go.loglayer.dev/transports/pretty v0.0.0-00010101000000-000000000000
)

require (
	github.com/fatih/color v1.19.0 // indirect
	github.com/goccy/go-json v0.10.6 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	golang.org/x/sys v0.42.0 // indirect
)
