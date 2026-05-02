module go.loglayer.dev/examples/otel-end-to-end/v2

go 1.25.0

replace (
	go.loglayer.dev/plugins/oteltrace/v2 => ../../plugins/oteltrace
	go.loglayer.dev/plugins/plugintest/v2 => ../../plugins/plugintest
	go.loglayer.dev/transports/otellog/v2 => ../../transports/otellog
	go.loglayer.dev/transports/testing/v2 => ../../transports/testing
	go.loglayer.dev/v2 => ../..
)

require (
	go.loglayer.dev/plugins/oteltrace/v2 v2.0.0-00010101000000-000000000000
	go.loglayer.dev/transports/otellog/v2 v2.0.0-00010101000000-000000000000
	go.loglayer.dev/v2 v2.0.0-00010101000000-000000000000
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
