---
"go.loglayer.dev": minor
---

Expose `Prefix` on dispatch-time params.

`loglayer.TransportParams` and every dispatch-time plugin hook param struct (`BeforeDataOutParams`, `BeforeMessageOutParams`, `TransformLogLevelParams`, `ShouldSendParams`) now carry `Prefix string`, populated from the `WithPrefix` value of the emitting logger. Transports and plugins can render or react to the prefix independently from the message text.

The legacy auto-prepend (the core writes `prefix + " "` into `Messages[0]` before dispatch) is preserved unchanged for backwards compatibility, so every existing transport keeps its v1 user-visible output. A future major version will remove the auto-prepend; transports that opt into reading `params.Prefix` today should either ignore the duplicate in `Messages[0]` (legacy behavior) or strip it before their own rendering (smart rendering).

Documented on `creating-transports.md` (Reading `params.Prefix`) and `creating-plugins.md` (Reading `params.Prefix`).

`internal/lltest.LogLine` gained a matching `Prefix` field so test fixtures can assert on the new signal.
