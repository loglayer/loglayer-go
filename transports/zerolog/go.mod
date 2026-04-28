// Separate module so consumers who don't import the zerolog wrapper
// don't pull github.com/rs/zerolog into their go.sum. See AGENTS.md
// "When to Split a Transport into Its Own Module".
module go.loglayer.dev/transports/zerolog

go 1.25.0

replace go.loglayer.dev => ../..

require (
	github.com/rs/zerolog v1.35.1
	go.loglayer.dev v0.0.0-00010101000000-000000000000
)

require (
	github.com/goccy/go-json v0.10.6 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	golang.org/x/sys v0.42.0 // indirect
)
