// Separate module so consumers who don't import the phuslu wrapper
// don't pull github.com/phuslu/log into their go.sum. See AGENTS.md
// "When to Split a Transport into Its Own Module".
module go.loglayer.dev/transports/phuslu

go 1.25.0

replace go.loglayer.dev => ../..

require (
	github.com/phuslu/log v1.0.124
	go.loglayer.dev v0.0.0-00010101000000-000000000000
)

require github.com/goccy/go-json v0.10.6 // indirect
