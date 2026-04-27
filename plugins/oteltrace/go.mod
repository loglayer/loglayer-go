// Separate module so the OpenTelemetry API's Go 1.25 floor doesn't
// drag the main go.loglayer.dev module up. See AGENTS.md "Go Version
// Floors" for the policy.
module go.loglayer.dev/plugins/oteltrace

go 1.25.0

toolchain go1.26.2

replace go.loglayer.dev => ../..

require (
	go.loglayer.dev v0.0.0-00010101000000-000000000000
	go.opentelemetry.io/otel v1.43.0
	go.opentelemetry.io/otel/sdk v1.43.0
	go.opentelemetry.io/otel/trace v1.43.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel/metric v1.43.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
)
