module go.loglayer.dev/transports/logrus/v2

go 1.25.0

replace go.loglayer.dev/v2 => ../..

require (
	github.com/sirupsen/logrus v1.9.4
	go.loglayer.dev/v2 v2.0.0-00010101000000-000000000000
)

require (
	github.com/goccy/go-json v0.10.6 // indirect
	golang.org/x/sys v0.42.0 // indirect
)
