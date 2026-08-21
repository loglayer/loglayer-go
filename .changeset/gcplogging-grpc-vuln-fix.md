---
"transports/gcplogging": patch
---

Bump `google.golang.org/grpc` from v1.79.3 to v1.82.1 (plus transitive
upgrades) to clear GO-2026-6061 (xDS RBAC authorization + HTTP/2
transport server vulnerabilities), which was reachable from
`SendToLogger` / `reportError`.
