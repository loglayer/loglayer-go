| Name | Description |
|------|-------------|
| [Redact](/plugins/redact) | Replace values for a configured set of keys (and optional regex patterns) before metadata or fields reach a transport. Walks structs, maps, and slices via reflection; preserves the runtime type. |
| [Sampling](/plugins/sampling) | Drop a fraction of emissions to keep volume and cost under control. `FixedRate` (per-emission Bernoulli draw), `FixedRatePerLevel` (per-level rate), and `Burst` (rate cap per rolling window). Composes with itself for "1% kept, capped at 100/sec" patterns. |
| [Format Strings](/plugins/fmtlog) | Opt the logger into `fmt.Sprintf`-style format strings: `log.Info("user %d", id)` resolves to `"user 1234"` before downstream hooks see it. |
| [Datadog APM Trace Injector](/plugins/datadogtrace) | Inject the active Datadog APM trace and span IDs (`dd.trace_id`, `dd.span_id`) into every log entry that carries a context, enabling Datadog's log/trace correlation. Tracer-agnostic; bring your own `dd-trace-go` (v1 or v2). |
| [OpenTelemetry Trace Injector](/plugins/oteltrace) | Inject the active OTel `trace_id` and `span_id` (and optional `trace_flags`) into every log entry that carries a context. Use with non-OTel transports for log/trace correlation; the OTel pipeline does this itself when shipping via `transports/otellog`. |
