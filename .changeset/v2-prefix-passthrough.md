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

**Breaking: import paths bump to `/v2`.** The loglayer core no longer folds the `WithPrefix` value into `Messages[0]`; the prefix flows through `TransportParams.Prefix` and each transport renders it. Built-in transports preserve their prior user-visible output via the new `transport.JoinPrefixAndMessages` helper. The `cli` transport opts into smart rendering (dim-grey user prefix separate from level color).

See [Migrating to v2](https://go.loglayer.dev/migrating-to-v2) for the upgrade checklist.
