module go.loglayer.dev/examples/http-server

go 1.25.0

replace (
	go.loglayer.dev => ../..
	go.loglayer.dev/integrations/loghttp => ../../integrations/loghttp
)

require (
	go.loglayer.dev v0.0.0-00010101000000-000000000000
	go.loglayer.dev/integrations/loghttp v0.0.0-00010101000000-000000000000
)
