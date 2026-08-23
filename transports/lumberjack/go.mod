module go.loglayer.dev/transports/lumberjack/v3

go 1.25.0

require (
	go.loglayer.dev/transports/structured/v3 v3.0.0-00010101000000-000000000000
	go.loglayer.dev/v3 v3.0.0
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
)

require github.com/goccy/go-json v0.10.6 // indirect

replace go.loglayer.dev/transports/structured/v3 => ../structured
