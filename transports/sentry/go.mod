module go.loglayer.dev/transports/sentry

go 1.25.0

replace go.loglayer.dev => ../..

require (
	github.com/getsentry/sentry-go v0.46.1
	github.com/goccy/go-json v0.10.6
	go.loglayer.dev v0.0.0-00010101000000-000000000000
)

require (
	golang.org/x/sys v0.18.0 // indirect
	golang.org/x/text v0.14.0 // indirect
)
