module go.loglayer.dev/transports/zap/v2

go 1.25.0

replace go.loglayer.dev/v2 => ../..

require (
	go.loglayer.dev/v2 v2.0.0-00010101000000-000000000000
	go.uber.org/zap v1.27.1
)

require (
	github.com/goccy/go-json v0.10.6 // indirect
	go.uber.org/multierr v1.11.0 // indirect
)
