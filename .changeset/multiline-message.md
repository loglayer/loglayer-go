---
"go.loglayer.dev": minor
"transports/cli": minor
"transports/pretty": minor
"transports/console": minor
---

Add `loglayer.Multiline(lines ...any)` for authoring multi-line message
content that survives terminal-renderer sanitization. The wrapper is
messages-only in v1; field/metadata values are still sanitized to a
single line in terminal transports (JSON sinks serialize via
`MarshalJSON` to the joined string).

Also fixes a pre-existing bug in `transport.JoinPrefixAndMessages`
where a `WithPrefix` value was silently dropped when `Messages[0]`
was not a string (e.g. `log.WithPrefix("X").Info(42)` lost the
prefix). The prefix now folds in front of the `%v`-formatted first
message.

See https://go.loglayer.dev/logging-api/multiline.
