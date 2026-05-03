module go.loglayer.dev/examples/multi-transport/v2

go 1.25.0

replace (
	go.loglayer.dev/transports/pretty/v2 => ../../transports/pretty
	go.loglayer.dev/transports/structured/v2 => ../../transports/structured
	go.loglayer.dev/v2 => ../..
)

require (
	go.loglayer.dev/transports/pretty/v2 v2.0.0-00010101000000-000000000000
	go.loglayer.dev/transports/structured/v2 v2.0.0-00010101000000-000000000000
	go.loglayer.dev/v2 v2.1.0
)

require (
	github.com/fatih/color v1.19.0 // indirect
	github.com/goccy/go-json v0.10.6 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	golang.org/x/sys v0.42.0 // indirect
)
