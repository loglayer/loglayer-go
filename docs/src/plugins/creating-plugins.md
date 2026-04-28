---
title: Creating Plugins
description: How to write a LogLayer plugin and which hook to reach for.
---

# Creating Plugins

A plugin is anything that satisfies the [`loglayer.Plugin`](https://pkg.go.dev/go.loglayer.dev#Plugin) interface, plus zero or more hook interfaces for the lifecycle points you want to participate in.

```go
type Plugin interface {
    ID() string
}
```

`ID()` is the only required method; everything else is opt-in via narrower interfaces. Return the empty string to let LogLayer auto-generate one at registration; supply your own ID when callers need to call `RemovePlugin` / `GetPlugin` or replace the plugin via `AddPlugin` later (those operations key off the ID).

For the registration API see [Plugin Configuration](/plugins/configuration) and [Plugin Management](/plugins/management).

## Two ways to write one

**For single-hook inline plugins** use one of the adapter constructors:

```go
import "go.loglayer.dev"

p := loglayer.NewMessageHook("prefix-msg", func(p loglayer.BeforeMessageOutParams) []any {
    if len(p.Messages) == 0 {
        return p.Messages
    }
    if s, ok := p.Messages[0].(string); ok {
        p.Messages[0] = "[svc] " + s
    }
    return p.Messages
})
log.AddPlugin(p)
```

The full set: `NewFieldsHook`, `NewMetadataHook`, `NewDataHook`, `NewMessageHook`, `NewLevelHook`, `NewSendGate`. There's also `NewPlugin(id)` for a no-op plugin (useful in tests for management semantics).

**For multi-hook plugins** declare a type implementing `Plugin` plus the relevant hook interfaces:

```go
package mything

import "go.loglayer.dev"

type Plugin struct {
    id  string
    cfg Config
}

func New(cfg Config) loglayer.Plugin {
    return &Plugin{id: "mything", cfg: cfg}
}

func (p *Plugin) ID() string { return p.id }

func (p *Plugin) OnFieldsCalled(f loglayer.Fields) loglayer.Fields { /* ... */ }
func (p *Plugin) OnBeforeDataOut(bp loglayer.BeforeDataOutParams) loglayer.Data { /* ... */ }
```

This is the canonical Go shape: an opaque interface from the consumer's point of view, satisfied by your concrete type. The built-in `plugins/redact` is the reference implementation.

## Hooks

Six hook interfaces fire during emission. Implement only the ones you need.

| Interface | Method | Fires when | Return |
|---|---|---|---|
| `FieldsHook` | `OnFieldsCalled(Fields) Fields` | `WithFields` is called | `Fields` to merge; nil drops the call |
| `MetadataHook` | `OnMetadataCalled(any) any` | `WithMetadata` / `MetadataOnly` | `any` metadata; nil drops it |
| `DataHook` | `OnBeforeDataOut(BeforeDataOutParams) Data` | per emission, after data assembly | `Data` to merge into the entry |
| `MessageHook` | `OnBeforeMessageOut(BeforeMessageOutParams) []any` | per emission, after data hooks | replacement messages slice; nil keeps |
| `LevelHook` | `TransformLogLevel(TransformLogLevelParams) (LogLevel, bool)` | per emission, after the above | `(level, ok)`; ok=false leaves unchanged |
| `SendGate` | `ShouldSend(ShouldSendParams) bool` | once per (entry, transport) | `false` to skip that transport |

`SendGate` is the only hook that sees the transport ID, so it's the place to gate dispatch per-transport.

A separate optional interface, `ErrorReporter`, lets a plugin observe recovered panics in its own hooks; see [Panic recovery](#panic-recovery).

### Lifecycle

```
WithFields(...)            → FieldsHook chain → fields merged onto logger
WithMetadata(...)          → MetadataHook chain → metadata stored

emission (Info / Warn / ...)
  ├─ assemble data (fields + error)
  ├─ DataHook chain                 (mutate the data map)
  ├─ MessageHook chain              (mutate the messages slice)
  ├─ LevelHook chain                (last ok=true wins)
  └─ for each transport:
       ├─ SendGate (false → skip this transport)
       └─ transport.SendToLogger(...)
```

Plugins run in the order they were added. `FieldsHook` and `MetadataHook` short-circuit on the first nil return; the dispatch-time hooks all run.

### Child loggers inherit plugins

`Child()` (and `WithFields`, `WithPrefix`) clones the plugin set by reference. Once either side mutates, the snapshots fork (copy-on-write). Adding a plugin to the child does not affect the parent.

## Hook reference

::: warning Nil-return semantics differ by hook
The two **input-side** hooks treat a nil return as "drop the input." The four **dispatch-time** hooks treat a nil return as "no transformation." This asymmetry is intentional but easy to misremember.

| Hook | Returning nil means |
|---|---|
| `FieldsHook` | Drop the WithFields call (receiver's existing fields preserved) |
| `MetadataHook` | Drop metadata for this entry |
| `DataHook` | Leave the assembled data unchanged |
| `MessageHook` | Leave the messages unchanged |
| `LevelHook` | (Returns `(_, false)` instead of nil) Leave the level unchanged |
| `SendGate` | (Not applicable: returns `bool`) |

The split: **input-side hooks** fire from `WithFields` / `WithMetadata`, where the user explicitly attached a value. Returning nil there is a meaningful "drop." **Output-side hooks** fire during dispatch, often from plugins that only want to transform sometimes. Returning nil there means "I don't have an opinion about this entry" rather than "drop everything."
:::

### `FieldsHook`

```go
type FieldsHook interface {
    OnFieldsCalled(fields Fields) Fields
}
```

Fires from `WithFields`. You receive the fields about to be merged onto the logger. Return the fields to actually merge; return `nil` to drop the call (the receiver's existing fields are preserved either way).

```go
loglayer.NewFieldsHook("uppercase-keys", func(fields loglayer.Fields) loglayer.Fields {
    out := make(loglayer.Fields, len(fields))
    for k, v := range fields {
        out["U_"+k] = v
    }
    return out
})
```

### `MetadataHook`

```go
type MetadataHook interface {
    OnMetadataCalled(metadata any) any
}
```

Fires from `WithMetadata` and `MetadataOnly`. The metadata can be any value (map, struct, scalar). If you only handle one shape, type-assert and pass through anything you don't understand:

```go
loglayer.NewMetadataHook("redact-password", func(metadata any) any {
    m, ok := metadata.(map[string]any)
    if !ok {
        return metadata
    }
    // Clone before mutating: m is the caller's map.
    out := make(map[string]any, len(m))
    for k, v := range m {
        out[k] = v
    }
    if _, has := out["password"]; has {
        out["password"] = "[REDACTED]"
    }
    return out
})
```

Return `nil` to drop the metadata for that entry.

::: warning Don't mutate caller inputs
The `metadata` you receive is the value the user passed to `WithMetadata`. If you mutate it in place, the user's variable changes too. They may pass the same map into multiple log calls and be surprised when the second call already has redacted values. Clone before you mutate. The same applies to `FieldsHook`.
:::

If your plugin needs to walk **structs and nested values** (not just top-level maps), see [Walking arbitrary inputs](#walking-arbitrary-inputs) below.

### `DataHook`

```go
type DataHook interface {
    OnBeforeDataOut(BeforeDataOutParams) Data
}
```

Fires per emission, after the data map is assembled (fields + error). Return a map of keys to merge into the entry's data. The returned map is **merged**, not substituted; nil leaves the data unchanged.

```go
loglayer.NewDataHook("tag", func(p loglayer.BeforeDataOutParams) loglayer.Data {
    return loglayer.Data{
        "service": "checkout",
        "version": buildVersion,
    }
})
```

### `MessageHook`

```go
type MessageHook interface {
    OnBeforeMessageOut(BeforeMessageOutParams) []any
}
```

Fires per emission, after `DataHook`. Return a replacement messages slice; nil leaves messages unchanged.

```go
loglayer.NewMessageHook("no-newlines", func(p loglayer.BeforeMessageOutParams) []any {
    out := make([]any, len(p.Messages))
    for i, m := range p.Messages {
        if s, ok := m.(string); ok {
            out[i] = strings.ReplaceAll(s, "\n", " ")
        } else {
            out[i] = m
        }
    }
    return out
})
```

### `LevelHook`

```go
type LevelHook interface {
    TransformLogLevel(TransformLogLevelParams) (LogLevel, bool)
}
```

Fires per emission, after the message hooks. Return `(level, true)` to override the entry's level; return `(_, false)` to leave it unchanged.

If multiple plugins return `ok=true`, the last one wins.

```go
loglayer.NewLevelHook("promote-on-error-key", func(p loglayer.TransformLogLevelParams) (loglayer.LogLevel, bool) {
    if _, hasErr := p.Data["err"]; hasErr && p.LogLevel < loglayer.LogLevelError {
        return loglayer.LogLevelError, true
    }
    return 0, false
})
```

`LevelHook` happens **after** the per-method level filter. An `Info` call that's filtered out by `SetLevel(LogLevelWarn)` doesn't reach this hook; you can't use it to "rescue" an entry that the core already dropped.

### `SendGate`

```go
type SendGate interface {
    ShouldSend(ShouldSendParams) bool
}
```

Fires once per (entry, transport) pair, just before dispatch. Return `false` to skip that transport.

```go
loglayer.NewSendGate("no-debug-to-shipping", func(p loglayer.ShouldSendParams) bool {
    if p.TransportID == "shipping" && p.LogLevel == loglayer.LogLevelDebug {
        return false
    }
    return true
})
```

If multiple plugins implement `SendGate`, the entry goes only when **every** plugin returns true.

## Per-call `context.Context`

All four dispatch-time hooks (`DataHook`, `MessageHook`, `LevelHook`, `SendGate`) receive a `Ctx context.Context` field on their params, populated from `WithCtx(ctx)`. The value is `nil` when the user didn't attach a context.

Use it to:

- Read trace IDs / span IDs and inject them as fields:

  ```go
  loglayer.NewDataHook("inject-trace", func(p loglayer.BeforeDataOutParams) loglayer.Data {
      if span := trace.SpanFromContext(p.Ctx); span.SpanContext().IsValid() {
          return loglayer.Data{
              "trace_id": span.SpanContext().TraceID().String(),
              "span_id":  span.SpanContext().SpanID().String(),
          }
      }
      return nil
  })
  ```

- Skip dispatch when the caller's context is cancelled:

  ```go
  loglayer.NewSendGate("ctx-gate", func(p loglayer.ShouldSendParams) bool {
      if p.Ctx == nil {
          return true
      }
      return p.Ctx.Err() == nil
  })
  ```

`MetadataHook` and `FieldsHook` do **not** receive a context. This is intentional, not an oversight: these hooks fire at builder time, when chain order is non-deterministic. A user can write `log.WithMetadata(m).WithCtx(ctx).Info(...)` (metadata first, ctx second) just as easily as `log.WithCtx(ctx).WithMetadata(m).Info(...)`. Threading ctx into the hook would mean it's `nil` half the time depending on call order, which is worse than not having it at all.

If you need context-aware behavior, use one of the dispatch-time hooks. They fire after every `With*` chain method has run, so the ctx they receive is the same one the transport will see.

## Walking arbitrary inputs

`MetadataHook.OnMetadataCalled` receives `any`. Real call sites pass maps, structs, pointers, slices, and scalars interchangeably. Any plugin that wants to "look inside" the value (redact, sanitize, rename, audit) faces the same problem: handle every shape uniformly without mutating the caller's input.

Three recipes apply, depending on what you want to handle and what you want the output to look like. The shared `utils/maputil` package gives you the primitives.

### Recipe 1: stay map-only

If your plugin only meaningfully operates on `map[string]any`, type-assert and pass through everything else:

```go
loglayer.NewMetadataHook("map-only", func(metadata any) any {
    m, ok := metadata.(map[string]any)
    if !ok {
        return metadata // structs, scalars, slices pass through unchanged
    }
    return cloneAndTransform(m)
})
```

Simple, predictable, no reflection. The downside: a struct with a `Password` field passes through untouched.

### Recipe 2: walk every shape (preserve type)

If your plugin needs to walk structs and nested values (recursively, honoring `json` tags), use [`maputil.Cloner`](https://pkg.go.dev/go.loglayer.dev/utils/maputil#Cloner). It produces a deep clone of any value with replacement predicates applied at any depth, preserving the runtime type.

```go
import "go.loglayer.dev/utils/maputil"

cloner := &maputil.Cloner{
    MatchKey:   func(k string) bool { return k == "password" || k == "apiKey" },
    MatchValue: func(s string) bool { return false },
    Censor:     "[REDACTED]",
}

loglayer.NewMetadataHook("redact", func(metadata any) any {
    return cloner.Clone(metadata)
})
```

`Cloner` handles maps (string-keyed), structs (json-tag aware), slices, arrays, pointers, and interface values. It skips unexported fields. Caller's input is never mutated.

The [`plugins/redact`](/plugins/redact) plugin is built on `Cloner`; [its source](https://github.com/loglayer/loglayer-go/blob/main/plugins/redact/redact.go) is the canonical reference for this pattern. It's also the canonical example of a multi-hook plugin (implements both `MetadataHook` and `FieldsHook`).

### Recipe 3: normalize to a map first

If the **shape** matters more than preserving the user's runtime type, use [`maputil.ToMap`](https://pkg.go.dev/go.loglayer.dev/utils/maputil#ToMap) to JSON-roundtrip the input, then walk the resulting map.

```go
import "go.loglayer.dev/utils/maputil"

loglayer.NewMetadataHook("normalize", func(metadata any) any {
    m := maputil.ToMap(metadata)
    if m == nil {
        return metadata
    }
    return walkMap(m)
})
```

The trade-off: the metadata reaches downstream plugins and transports as a `map[string]any`, not the user's struct. Anything that type-switches on `params.Metadata` will see a map. For most rendering paths this is invisible (they marshal to JSON anyway), but tests that compare to the original struct break.

### Performance: only clone if you mutate

The "don't mutate caller's input" rule means **mutating** plugins must clone. Read-only plugins (audit, metrics, sampling) should not. Both `Cloner` and `ToMap` always allocate a fresh value, even when nothing matches; if your plugin is going to return the input unchanged, return it unchanged and skip the clone.

```go
// ❌ unnecessary clone on every emission
loglayer.NewMetadataHook("naive", func(metadata any) any {
    return cloner.Clone(metadata) // always allocates, even if nothing redacts
})

// ✅ inspect first, clone only when there's work to do
loglayer.NewMetadataHook("smart", func(metadata any) any {
    if !containsSensitiveKeys(metadata) {
        return metadata
    }
    return cloner.Clone(metadata)
})
```

For pipelines with multiple mutating plugins, costs add up: each plugin gets the previous one's output and cloning it again means N deep walks per emission. Two mitigations:

- **Order matters.** Place cheap or filtering plugins (`SendGate`, level transforms) before expensive walking plugins so dropped entries never pay the clone cost.
- **Combine where possible.** If two plugins both redact, a single plugin with both rule sets does one walk instead of two. (The built-in `redact` plugin accepts multiple keys and patterns for exactly this reason.)

In practice most pipelines have zero or one mutating metadata plugin (typically `redact`), so the typical cost is one clone per emission. The hot path for read-only plugins is alloc-free.

## Panic recovery

Every hook call is wrapped in a deferred recover. If your hook panics, the framework swallows the panic, logging continues, and the entry treats the hook's contribution as if it returned the "no transformation" / "drop input" / "fail open" value for that hook:

| Hook | Behavior on panic |
|---|---|
| `FieldsHook` | Drops the input (nil return) |
| `MetadataHook` | Drops the input (nil return) |
| `DataHook` | No data merged (nil return) |
| `MessageHook` | Messages unchanged (nil return) |
| `LevelHook` | Level unchanged (`ok=false`) |
| `SendGate` | Entry sent to the transport (fails open) |

The framework writes a one-line description of the recovered panic to `os.Stderr` so the failure isn't silent. To observe the panic in your own code, implement [`ErrorReporter`](https://pkg.go.dev/go.loglayer.dev#ErrorReporter):

```go
type ErrorReporter interface {
    OnError(err error)
}
```

A plugin that wants to log recovered panics to its own observability stack defines `OnError` on its concrete type:

```go
type myPlugin struct{ /* ... */ }

func (p *myPlugin) ID() string { return "my-plugin" }
func (p *myPlugin) OnBeforeDataOut(...) loglayer.Data { /* may panic */ }
func (p *myPlugin) OnError(err error) {
    metrics.IncrPluginPanic("my-plugin")
    fmt.Fprintln(os.Stderr, "plugin error:", err)
}
```

When `ErrorReporter` is implemented, the framework calls it instead of writing to stderr. Either way, the panic never propagates to the caller's goroutine.

## Concurrency and performance

Hooks run on the dispatching goroutine. They may be called from any goroutine concurrently; the same plugin instance can fire on many emissions in parallel. Make any state your hook touches safe for concurrent reads/writes (use a mutex, atomics, or build the plugin from immutable config).

**Don't block in a hook**: it stalls the log call.

- Map lookups, string comparisons, simple type assertions: fine.
- Network or disk I/O: never. If you need to ship to an external system, enqueue to a channel and have a worker drain it.
- Reflection or JSON-encoding for every entry: usually too slow at high log volume; cache or precompute where you can.

The dispatcher pre-indexes hook membership at `AddPlugin` time, so having ten plugins where only one implements `DataHook` costs roughly the same as having one such plugin. You don't pay for hooks you don't implement.

## Convention: package shape

If you publish a plugin as a Go package, follow this shape:

```
yourpkg/
├── go.mod (if separate module)
├── plugin.go        // package yourpkg; Config + concrete plugin type + New(Config) loglayer.Plugin
├── plugin_test.go
└── README.md
```

The constructor signature `func New(Config) loglayer.Plugin` matches the [`plugins/redact`](/plugins/redact) reference plugin and the constructor pattern transports use. The returned value is your concrete type cast as `loglayer.Plugin`; consumers see the interface, you keep the implementation private.

## Testing

For testing a custom plugin, see [Testing Plugins](/plugins/testing-plugins). It covers `Install`, `AssertNoMutation` (verifies an input-side hook doesn't mutate caller-owned input), and `AssertPanicRecovered`.

## See also

- [Plugins overview](/plugins/): what hooks exist, when each fires, lifecycle and thread-safety semantics.
- [`plugins/redact`](/plugins/redact): built-in reference plugin built on `maputil.Cloner`.
