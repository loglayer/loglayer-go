---
title: Thread Safety
description: Every method on *LogLayer is safe to call from any goroutine. The classification per method.
---

# Thread Safety

Every method on `*loglayer.LogLayer` is safe to call from any goroutine, including concurrently with emission. There is no setup-only category. You don't need to clone or lock before passing a logger to a goroutine.

The contract is verified by `concurrency_test.go` under `-race`, including a runtime-level-toggle test and a transport hot-swap test.

## Per-method classification

| Class | Methods | How safety is achieved |
|-------|---------|------------------------|
| **Emission** | `Info`, `Warn`, `Error`, `Debug`, `Fatal`, `WithMetadata`, `WithError`, `WithCtx` (on builder), `Raw`, `MetadataOnly`, `ErrorOnly` | Read-only on logger state. |
| **Returns-new** | `WithFields`, `WithoutFields`, `Child`, `WithPrefix`, `WithGroup` (on `*LogLayer`), `WithCtx` (on `*LogLayer`) | Build a new logger; receiver untouched. |
| **Read-only** | `GetFields`, `GetLoggerInstance`, `IsLevelEnabled` | No state change. |
| **Level mutators** | `SetLevel`, `EnableLevel`, `DisableLevel`, `EnableLogging`, `DisableLogging` | Backed by an `atomic.Uint32` bitmap. Mirrors `zap.AtomicLevel`. Designed for live runtime toggling (SIGUSR1, admin endpoints flipping debug on, etc.). |
| **Transport mutators** | `AddTransport`, `RemoveTransport`, `SetTransports` | Publish a new immutable transport set via `atomic.Pointer`. Concurrent mutators on the same logger serialize via an internal mutex (slow path); the dispatch hot path only loads the pointer. |
| **Plugin mutators** | `AddPlugin`, `RemovePlugin` | Same atomic-pointer pattern as transports, serialized by `pluginMu`. |
| **Group mutators** | `AddGroup`, `RemoveGroup`, `EnableGroup`, `DisableGroup`, `SetGroupLevel`, `SetActiveGroups`, `ClearActiveGroups` | Same atomic-pointer pattern, serialized by `groupMu`. |
| **Mute toggles** | `MuteFields`, `UnmuteFields`, `MuteMetadata`, `UnmuteMetadata` | Backed by `atomic.Bool`. Concurrent reads from the dispatch path are safe, but flipping mid-emission can interleave (some entries see pre-toggle, others post). Treat them as setup-time admin toggles. |

## What this means in practice

- **Pass loggers to goroutines freely.** No clone needed. The receiver is unchanged by every emission method.
- **Use `WithFields` to derive per-request loggers** in HTTP handlers (or the [`loghttp` middleware](/integrations/loghttp), which does this for you). The new logger is goroutine-local; the parent is untouched.
- **`Child()` is the no-additions form of `WithFields`**: it returns an independent clone with the parent's fields shallow-copied, useful when you want isolation without adding fields.
- **Mute toggles aren't races, but they're racy in *intent***. Set them at startup or via an admin endpoint; don't expect a clean cutover mid-traffic.

## Caller-owned input

Maps you pass to `WithFields` and `WithMetadata` are **not deep-copied**. Treat them as read-only after handing them off, or build a fresh one per call. See the inline warnings on the [Fields](/logging-api/fields) and [Metadata](/logging-api/metadata) pages.