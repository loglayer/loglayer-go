---
title: About LogLayer for Go
description: Learn more about LogLayer for Go and how it unifies your logging experience.
---

# Introduction

Most Go projects already have structured logging working. What they don't have, out of the box, is a layer above it: redaction that travels with the logger, where logs go, what they get rewritten to first, and how to split traffic by subsystem.

LogLayer is that layer. It sits on top of whichever logging library you already use (or one of its built-in transports) and gives you:

- **A plugin pipeline.** Rewrite, filter, or enrich entries globally, before any transport sees them. The built-in [`redact`](/plugins/redact) plugin walks structs, maps, slices, and pointers via reflection and replaces matched keys (json-tag aware) or value patterns at any depth, with the runtime type preserved. [`oteltrace`](/plugins/oteltrace) and [`datadogtrace`](/plugins/datadogtrace) inject trace IDs from the bound context. Six narrow hook interfaces if you need your own.
- **Multi-destination fan-out with per-transport level filters.** Send the same emission to several transports at once, each with its own minimum level: pretty in dev plus structured to a file plus batched HTTP to Datadog, from one logger.
- **Group routing.** Tag entries by subsystem (`db`, `auth`, ...) and route each tag to specific transports with its own minimum level. Toggle which groups are active at runtime via env var.
- **Two-way slog interop.** Wrap a `*log/slog.Logger` as a backend ([transports/slog](/transports/slog)), or install a [`slog.Handler`](/integrations/sloghandler) so every `slog.Info(...)` call (yours and your dependencies') flows through the loglayer pipeline.
- **First-class test capture.** A typed `LogLine` for each emission, so tests assert on level, message, fields, metadata, and context independently. No JSON parsing in tests.
- **Caller info, opt-in.** `Config.Source.Enabled` captures file/line/function for every emission and surfaces it under a configurable key (default `source`). JSON tags match the `log/slog` convention so structured output is interchangeable. The slog handler forwards `Record.PC` automatically, no toggle required.
- **Distinct types for persistent fields, per-call metadata, and errors.** The compiler catches misuse; the dispatch path serializes each consistently.
- **Bring-your-own logger.** Wrap a `*zerolog.Logger`, `*zap.Logger`, `*log/slog.Logger`, `*logrus.Logger`, `*charmbracelet/log.Logger`, or `*phuslu/log.Logger` you've already configured. The API in your call sites becomes uniform without a rewrite.
- **Runtime control.** Hot-swap transports, add or remove plugins, toggle levels, all live and concurrency-safe.

If your service is small and you only need "log to stdout in JSON," the stdlib is fine. The friction LogLayer fixes shows up later: when you add a second destination, redact a field across every log site, or want to wire in OpenTelemetry without rewriting how you log everywhere.

LogLayer adds about 40 ns and one allocation per emission on top of whatever your underlying logger costs. See [Benchmarks](/benchmarks) for the full picture.

```go
log.
    WithMetadata(loglayer.Metadata{"userId": "1234"}).
    WithError(errors.New("something went wrong")).
    Error("user action failed")
```

```json
{
  "msg": "user action failed",
  "userId": "1234",
  "err": { "message": "something went wrong" }
}
```

::: tip Coming from TypeScript?
This is the Go port of [`loglayer`](https://loglayer.dev) for TypeScript. The mental model and API shape map directly. See [For TypeScript Developers](/for-typescript-developers) for the full convention map and the deliberate Go-specific differences (`Fields` instead of `context`, threading guarantees, error handling, module layout).
:::

## Bring Your Own Logger

LogLayer is designed to sit on top of your logging library of choice (`zerolog`, `zap`, `slog`, `logrus`, `charmbracelet/log`, `phuslu/log`) or to run standalone with one of the built-in transports (pretty terminal, structured JSON, HTTP, console).

Start with the built-in pretty transport during development, then switch to the zerolog or zap transport later when you have a real production setup, without changing a single log call.

Learn more about logging [transports](/transports/).

## Consistent API

No need to remember different parameter orders or method names between logging libraries:

```go
// With LogLayer, same call shape regardless of backend
log.WithMetadata(loglayer.Metadata{"some": "data"}).Info("my message")

// Without LogLayer, every library wants something different
zerologLogger.Info().Interface("some", "data").Msg("my message")
zapLogger.Info("my message", zap.Any("some", "data"))
slog.Info("my message", "some", "data")
```

Start with [basic logging](/logging-api/basic-logging).

## Separation of Errors, Fields, and Metadata

LogLayer distinguishes three kinds of structured data, each with a clear scope:

<!--@include: ./logging-api/_partials/fields-vs-metadata.md-->

This separation provides several benefits:

- **Clarity**: each piece of data has a clear purpose and appropriate scope.
- **No pollution**: per-log metadata can never accidentally persist to future logs.
- **Flexible output**: configure where each type appears in the final log (root level or dedicated fields) via [configuration](/configuration).
- **Better debugging**: errors are serialized consistently via a configurable `ErrorSerializer`.

```go
log.
    WithFields(loglayer.Fields{"requestId": "abc-123"}). // persists
    WithMetadata(loglayer.Metadata{"duration": 150}).    // this log only
    WithError(errors.New("timeout")).                    // this log only
    Error("Request failed")
```

```json
{
  "msg": "Request failed",
  "requestId": "abc-123",
  "duration": 150,
  "err": { "message": "timeout" }
}
```

See the dedicated pages for [fields](/logging-api/fields), [metadata](/logging-api/metadata), and [error handling](/logging-api/error-handling).

## Powerful Plugin System

Plugins extend the emission pipeline so you can rewrite, filter, or enrich entries globally without touching call sites. The built-in [`plugins/redact`](/plugins/redact) plugin walks structs, maps, and slices via reflection, redacting matched keys at any nesting depth. The [`plugins/datadogtrace`](/plugins/datadogtrace) and [`plugins/oteltrace`](/plugins/oteltrace) plugins inject trace IDs from the bound context. See [Plugins](/plugins/) for the catalog and [Creating Plugins](/plugins/creating-plugins) to write your own.

## Multi-Transport Support

Send your logs to multiple destinations simultaneously:

```go
log := loglayer.New(loglayer.Config{
    Transports: []loglayer.Transport{
        pretty.New(pretty.Config{}),                           // dev console
        structured.New(structured.Config{Writer: jsonFile}),   // shipping
    },
})

log.Info("user signed in") // both transports receive it
```

See more about [multi-transport support](/transports/multiple-transports).

## Targeted Log Routing with Groups

In a large system with many subsystems, you often want certain logs to go to certain destinations. Groups let you tag logs by category and route them to specific transports with per-group log levels:

```go
log := loglayer.New(loglayer.Config{
    Transports: []loglayer.Transport{...},
    Routing: loglayer.RoutingConfig{
        Groups: map[string]loglayer.LogGroup{
            "database": {Transports: []string{"datadog"}, Level: loglayer.LogLevelError},
            "auth":     {Transports: []string{"datadog", "console"}, Level: loglayer.LogLevelWarn},
        },
    },
})

// Tag individual logs
log.WithGroup("database").Error("connection lost")

// Or create a dedicated logger for a subsystem
dbLogger := log.WithGroup("database")
dbLogger.Error("pool exhausted") // routed to datadog only
```

Narrow focus to a specific subsystem at runtime via an environment variable, no code changes:

```go
loglayer.New(loglayer.Config{
    Routing: loglayer.RoutingConfig{
        ActiveGroups: loglayer.ActiveGroupsFromEnv("LOGLAYER_GROUPS"),
    },
})
```

```sh
LOGLAYER_GROUPS=database,auth go run .
```

See more about [groups](/logging-api/groups).

## HTTP and Cloud Shipping

Send logs directly to any HTTP endpoint without a third-party logging library, with built-in batching, retries, and a pluggable encoder. The [HTTP transport](/transports/http) is the foundation; the [Datadog transport](/transports/datadog) is built on top of it for the Datadog Logs intake API.

## Easy Testing

Built-in mocks make testing painless:

```go
// Silent mock for tests that don't care about output
log := loglayer.NewMock()

// Capturing transport for tests that assert on what was logged
lib := &lltest.TestLoggingLibrary{}
log := loglayer.New(loglayer.Config{
    Transport: lltest.New(lltest.Config{Library: lib}),
})
log.WithMetadata(loglayer.Metadata{"k": "v"}).Info("msg")

line := lib.PopLine()
require.Equal(t, "msg", line.Messages[0])
```

See more about [testing](/logging-api/mocking).
