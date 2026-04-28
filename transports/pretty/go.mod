// Separate module so consumers who don't use the pretty terminal
// renderer don't pull github.com/fatih/color (and its mattn/go-isatty
// closure) into their dependency graph.
module go.loglayer.dev/transports/pretty

go 1.25.0

replace go.loglayer.dev => ../..

require (
	github.com/fatih/color v1.19.0
	github.com/goccy/go-json v0.10.6
	go.loglayer.dev v0.0.0-00010101000000-000000000000
)

require (
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	golang.org/x/sys v0.42.0 // indirect
)
