// Separate module so consumers who don't import the zap wrapper
// don't pull go.uber.org/zap into their go.sum. See AGENTS.md
// "When to Split a Transport into Its Own Module".
module go.loglayer.dev/transports/zap

go 1.25.0

replace go.loglayer.dev => ../..

require (
	go.loglayer.dev v0.0.0-00010101000000-000000000000
	go.uber.org/zap v1.27.1
)

require (
	github.com/goccy/go-json v0.10.6 // indirect
	go.uber.org/multierr v1.10.0 // indirect
)
