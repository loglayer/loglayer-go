---
"go.loglayer.dev": major
"transports/blank": major
"transports/charmlog": major
"transports/cli": major
"transports/console": major
"transports/datadog": major
"transports/gcplogging": major
"transports/http": major
"transports/logrus": major
"transports/lumberjack": major
"transports/otellog": major
"transports/phuslu": major
"transports/pretty": major
"transports/sentry": major
"transports/slog": major
"transports/structured": major
"transports/testing": major
"transports/zap": major
"transports/zerolog": major
"plugins/datadogtrace": major
"plugins/fmtlog": major
"plugins/oteltrace": major
"plugins/plugintest": major
"plugins/redact": major
"plugins/sampling": major
"integrations/loghttp": major
"integrations/sloghandler": major
---

**Breaking: import paths bump to `/v2`.**

The loglayer core no longer mutates `Messages[0]` to fold the `WithPrefix` value into the message text. The prefix flows through `TransportParams.Prefix` (and the matching field on every dispatch-time plugin hook param struct). Each transport decides how to render the prefix:

- Most built-in transports call `transport.JoinPrefixAndMessages(params.Prefix, params.Messages)` at the top of `SendToLogger` to preserve the v1 user-visible output exactly.
- The cli transport opts into smart rendering: the level prefix and message body keep the level color (yellow / red), while the `WithPrefix` value gets its own dim-grey color, visually separating caller-context from urgency.
- The `blank` transport intentionally passes raw v2 params through to the user-supplied `ShipToLogger` function, so advanced users can decide their own rendering.

## Migration

Every consumer must update import paths to `/v2`:

```sh
go get go.loglayer.dev/v2 \
       go.loglayer.dev/transports/cli/v2 \
       go.loglayer.dev/transports/zerolog/v2 \
       # ... whichever sub-modules you import
```

In source files:

```diff
-import (
-    "go.loglayer.dev"
-    "go.loglayer.dev/transports/zerolog"
-)
+import (
+    "go.loglayer.dev/v2"
+    "go.loglayer.dev/transports/zerolog/v2"
+)
```

For most users no other changes are needed: the built-in transports preserve v1 user-visible output. Custom transports that consumed `params.Messages[0]` and assumed the prefix was already prepended must either:

1. Call `transport.JoinPrefixAndMessages(params.Prefix, params.Messages)` at the top of `SendToLogger` (legacy behavior preserved), or
2. Read `params.Prefix` directly and render it however you like (e.g. as a separate JSON field, in its own color, forwarded to the underlying logger's structured-field API).

The new `transport.JoinPrefixAndMessages` helper has fast-path early returns when the prefix is empty, when messages is empty, or when `messages[0]` isn't a string; the per-call cost for any logger that hasn't called `WithPrefix` is one string compare.

## Plugin authors

Plugin hooks (`OnBeforeDataOut`, `OnBeforeMessageOut`, `TransformLogLevel`, `ShouldSend`) gained a `Prefix string` field on their respective params structs (was added additively in v1.7.0). The field is read-only from the plugin's perspective. With v2, `params.Messages[0]` no longer carries the prefix — plugins that read messages directly should be aware that the prefix is now ONLY on `params.Prefix`.

## See also

The prior v1.7.0 release added `Prefix` to the param structs as an additive field; this release removes the auto-prepend.
