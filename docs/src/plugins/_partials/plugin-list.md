| Name | Description |
|------|-------------|
| [Redact](/plugins/redact) | Replace values for a configured set of keys (and optional regex patterns) before metadata or fields reach a transport. Walks structs, maps, and slices via reflection; preserves the runtime type. |
| [Datadog APM Trace Injector](/plugins/datadogtrace) | Inject the active Datadog APM trace and span IDs (`dd.trace_id`, `dd.span_id`) into every log entry that carries a context, enabling Datadog's log/trace correlation. Tracer-agnostic; bring your own `dd-trace-go` (v1 or v2). |
