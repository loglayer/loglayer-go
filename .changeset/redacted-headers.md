---
"transports/http": minor
---

Add `Config.String()` that redacts `Headers` values so an accidental `log.Info(cfg)` or `fmt.Sprintf("%v", cfg)` can't leak credentials passed via `Authorization` / `X-API-Key` / similar headers. Header keys stay visible for debuggability. Mirrors the redaction shape already used by `transports/datadog`.

`defaultCheckRedirect` now compares hosts case-insensitively, so legitimate same-host redirects with mixed-case URLs (`Example.COM` → `example.com`) aren't refused. Cross-host refusal still applies; ports are still compared exactly.

New `Config.ShutdownTimeout` (default 5s) bounds how long `Close` waits for in-flight requests to finish during shutdown. When the timeout elapses, the worker's outbound HTTP requests are cancelled via context so `Close` can return even if the endpoint is wedged; previously a stuck endpoint could pin `Close` for up to the per-request `Client.Timeout` (30s default), and the parent `flushTransports`'s 5s timeout would leak the close goroutine. Outbound requests are now built via `http.NewRequestWithContext`.
