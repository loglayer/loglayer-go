::: info Go version
Most plugins inherit the main module's Go floor (currently **1.25+**). Plugins that bring heavier SDK dependencies live in their own modules so their floors don't drag the main module up: `plugins/oteltrace` ships as `go.loglayer.dev/plugins/oteltrace`, and `plugins/datadogtrace/livetest` (test-only) is also a separate module. Per-plugin pages call out the install path and any stricter floor.
:::

| Name | Description |
|------|-------------|
| [Redact](/plugins/redact) | Replace values for a configured set of keys (and optional regex patterns) before metadata or fields reach a transport. Walks structs, maps, and slices via reflection; preserves the runtime type. |
| [Datadog APM Trace Injector](/plugins/datadogtrace) | Inject the active Datadog APM trace and span IDs (`dd.trace_id`, `dd.span_id`) into every log entry that carries a context, enabling Datadog's log/trace correlation. Tracer-agnostic; bring your own `dd-trace-go` (v1 or v2). |
| [OpenTelemetry Trace Injector](/plugins/oteltrace) | Inject the active OTel `trace_id` and `span_id` (and optional `trace_flags`) into every log entry that carries a context. Use with non-OTel transports for log/trace correlation; the OTel pipeline does this itself when shipping via `transports/otellog`. |
