// Separate module so consumers who don't import the logrus wrapper
// don't pull github.com/sirupsen/logrus into their go.sum. See AGENTS.md
// "When to Split a Transport into Its Own Module".
module go.loglayer.dev/transports/logrus

go 1.25.0

replace go.loglayer.dev => ../..

require (
	github.com/sirupsen/logrus v1.9.4
	go.loglayer.dev v0.0.0-00010101000000-000000000000
)

require (
	github.com/goccy/go-json v0.10.6 // indirect
	golang.org/x/sys v0.42.0 // indirect
)
