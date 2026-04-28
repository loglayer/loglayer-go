// Example showing transports/otellog and plugins/oteltrace together,
// with a real OpenTelemetry SDK wired to a stdout exporter so the
// example is runnable without an OTel collector. Lives in its own
// module because both otellog and oteltrace are themselves split out
// to avoid binding the main loglayer module to the OTel SDK's Go floor.
module go.loglayer.dev/examples/otel-end-to-end

go 1.25.0

replace (
	go.loglayer.dev => ../..
	go.loglayer.dev/plugins/oteltrace => ../../plugins/oteltrace
	go.loglayer.dev/transports/otellog => ../../transports/otellog
)

require (
	go.loglayer.dev v0.0.0-00010101000000-000000000000
	go.loglayer.dev/plugins/oteltrace v0.0.0-00010101000000-000000000000
	go.loglayer.dev/transports/otellog v0.0.0-00010101000000-000000000000
	go.opentelemetry.io/otel v1.43.0
	go.opentelemetry.io/otel/exporters/stdout/stdoutlog v0.19.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.43.0
	go.opentelemetry.io/otel/sdk v1.43.0
	go.opentelemetry.io/otel/sdk/log v0.19.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/goccy/go-json v0.10.6 // indirect
	github.com/google/uuid v1.6.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel/log v0.19.0 // indirect
	go.opentelemetry.io/otel/metric v1.43.0 // indirect
	go.opentelemetry.io/otel/trace v1.43.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
)
