---
title: Common Pitfalls
description: "Failure modes that bite first-time LogLayer users — and how to avoid them."
---

# Common Pitfalls

Most of these are Go-idiom subtleties that don't surface until you're staring at a log line that should be there but isn't. They're documented one at a time elsewhere, but seeing them collected makes the patterns easier to recognize.

## `WithFields` returns a new logger; the receiver is unchanged

```go
log := loglayer.New(loglayer.Config{...})

log.WithFields(loglayer.Fields{"requestId": rid})  // ❌ result discarded
log.Info("served")                                 // requestId is NOT here
```

`WithFields`, `WithoutFields`, `Child`, `WithPrefix`, `WithCtx` (on `*LogLayer`), and `WithGroup` (on `*LogLayer`) all return a *new* logger with the change applied. The receiver is untouched, matching how `zerolog`, `zap`, `slog`, and `logrus` handle the same pattern.

```go
log = log.WithFields(loglayer.Fields{"requestId": rid})  // ✅ assign result
log.Info("served")                                       // requestId is here
```

The same trap exists when handing the result to a function:

```go
// ❌ requestId never reaches the handler — `WithFields` result is dropped.
go runHandler(log)

// ✅ pass the derived logger.
go runHandler(log.WithFields(loglayer.Fields{"requestId": rid}))
```

## Mutating a `Fields` or `Metadata` map after binding

```go
m := loglayer.Metadata{"k": "v"}
log.WithMetadata(m).Info("first")
m["k"] = "v2"                        // ❌ may bleed into the first log
log.WithMetadata(m).Info("second")
```

LogLayer doesn't deep-copy maps you pass it. A transport that holds onto the map (the testing transport does, by design; some async transports may) will see your post-emission mutation. **Treat the map as read-only after you pass it in**, or build a fresh one per call.

## Forgetting to assign the persistent context

```go
ctx := r.Context()
log.WithCtx(ctx)             // ❌ no assignment, ctx not bound
log.Info("served")           // transports/plugins see no context
```

`(*LogLayer).WithCtx` returns a new logger; without assignment, nothing is bound. The builder-level `(*LogBuilder).WithCtx(ctx).Info(...)` form does NOT have this problem because the builder is single-use, but you can only attach for one emission at a time.

```go
log = log.WithCtx(r.Context())   // ✅ persistent on logger
log.Info("served")               // r.Context() is bound
log.Info("downstream done")      // still r.Context()
```

The `loghttp` middleware does this binding automatically — see [Go Context](/logging-api/go-context).

## Fatal exit behavior depends on the transport

`Config.DisableFatalExit: true` opts out of `os.Exit(1)` after a fatal log — usually. Two transport-shape concerns:

1. **`transports/phuslu` always exits on fatal**, regardless of `DisableFatalExit`, because the underlying `phuslu/log` library calls `os.Exit` from its fatal dispatch path. Pick another transport if you need fatal-without-exit.
2. **`transports/zap`, `slog`, `zerolog`, `logrus`, `charmlog`** each have library-specific machinery to neutralize the underlying logger's exit. They all defer the decision to LogLayer's `DisableFatalExit`. If a future library bump silently re-enables exit, the [livetests](/transports/) catch it; per-page docs describe the mechanism each transport uses.

See the [level-mapping table](/transports/#level-mapping-across-transports) for the complete picture.

## Mute toggles are not safe to flip during emission

```go
log.MuteFields()
go log.Info(...)   // ⚠️ racy with the toggle
```

`MuteFields`, `UnmuteFields`, `MuteMetadata`, `UnmuteMetadata` are backed by `atomic.Bool`, so concurrent reads from the dispatch path are safe. But the *visible behavior* of toggling mid-emission can interleave: some entries see the pre-toggle state, others the post. If you need a clean cutover, route through a feature flag or a level toggle instead.

## Group routing precedence has eight rules

The full precedence list lives on the [Groups page](/logging-api/groups#routing-precedence). Two rules people miss:

1. **Per-group `Level` filters the whole group, not individual transports.** If your `billing` group lists three transports with `Level: Error`, then `Info` is dropped from all three, not just from the noisy one. Split into two groups (one per "policy") if you need per-transport thresholds.
2. **An undefined group name in a tag is treated as no tag.** If every tag on an entry refers to a group that doesn't exist in `Config.Groups`, the entry falls through to `UngroupedRouting`. Compare to a *defined-but-`Disabled`* group, which is "explicitly off" and does not fall through.

## `MetadataFieldName` only applies to non-map metadata

```go
log.WithMetadata(loglayer.Metadata{"k": "v"}).Info(...)
// → flattens to root: {"k": "v", ...}

log.WithMetadata(MyStruct{ID: 1}).Info(...)
// → nests: {"metadata": {"id": 1}, ...}
```

Configuring `MetadataFieldName: "user"` only changes the *non-map* path. Map metadata always flattens to root attributes (this is the whole point of using `loglayer.Metadata` for ad-hoc bags). Use the `Fields` API for keyed data that should always nest under a specific name.

## "I picked the OTel transport but my logs are empty"

Two common reasons:

1. **No `LoggerProvider` wired up.** When `Config.Logger` and `Config.LoggerProvider` are both nil, `transports/otellog` falls back to `global.GetLoggerProvider`, which returns OTel's no-op provider unless you've called `global.SetLoggerProvider` somewhere in your startup. Construction succeeds; emission silently drops. Either set the global provider or pass `Config.LoggerProvider` explicitly.
2. **No SDK installed at all.** `transports/otellog` binds against `go.opentelemetry.io/otel/log` (the API). To actually export records you also need `go.opentelemetry.io/otel/sdk/log` plus an exporter (OTLP, stdout, etc.). The transport doesn't pull in the SDK by default — that's the user's choice.

See [the otellog transport guide](/transports/otellog) for full SDK setup.

## "My oteltrace plugin emits nothing on the second service"

Almost always: **trace context isn't propagating across the service boundary.** The plugin only reads what's already on `ctx`. Service B's handler needs an OTel propagator (e.g. [`otelhttp`](https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp)) installed to extract the `traceparent` header from the incoming request and seed `r.Context()` with the resulting span. Without propagation, B has no span on its context and the plugin emits nothing.

See [oteltrace](/plugins/oteltrace) for the full setup.
