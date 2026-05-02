module go.loglayer.dev/transports/lumberjack/v2

go 1.25.0

replace go.loglayer.dev/v2 => ../..

replace go.loglayer.dev/transports/structured/v2 => ../structured

require (
	go.loglayer.dev/transports/structured/v2 v2.0.0-00010101000000-000000000000
	go.loglayer.dev/v2 v2.0.0-00010101000000-000000000000
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
)

require github.com/goccy/go-json v0.10.6 // indirect
