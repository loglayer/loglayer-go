---
title: Creating Plugins
description: How to write a LogLayer plugin and which hook to reach for.
---

# Creating Plugins

A plugin is a [`loglayer.Plugin`](https://pkg.go.dev/go.loglayer.dev#Plugin) struct: an `ID` plus one or more hook function fields. Construct one inline and register it with `log.AddPlugin(...)`, or have a constructor function in your own package that returns a `loglayer.Plugin`.

```go
package mything

import "go.loglayer.dev"

func New(prefix string) loglayer.Plugin {
    return loglayer.Plugin{
        ID: "mything",
        OnBeforeMessageOut: func(p loglayer.BeforeMessageOutParams) []any {
            // prepend prefix to the first string message
            if len(p.Messages) == 0 {
                return p.Messages
            }
            if s, ok := p.Messages[0].(string); ok {
                p.Messages[0] = prefix + " " + s
            }
            return p.Messages
        },
    }
}
```

::: tip Single-hook shortcuts
For the common cases where a plugin only implements one hook, three named constructors avoid the struct-literal boilerplate:

```go
loglayer.MetadataPlugin("redact", func(m any) any { ... })
loglayer.FieldsPlugin("rename", func(f loglayer.Fields) loglayer.Fields { ... })
loglayer.LevelPlugin("promote", func(p loglayer.TransformLogLevelParams) (loglayer.LogLevel, bool) { ... })
```

These are sugar over `loglayer.Plugin{...}` — use the struct literal directly when you need multiple hooks.
:::

::: warning Plugin is consumed by value
The `Plugin` struct is copied at `AddPlugin` time. Mutating its function fields *after* registration has no effect on the registered behavior — to update a plugin, build a new `Plugin` and `AddPlugin` it again (the existing one is replaced because IDs match).
:::

For the registration API (`AddPlugin`, `RemovePlugin`, etc.) see [Plugin Management](/plugins/#plugin-management) on the overview page.

## Hooks

Six hooks fire during emission. Implement only the ones you need.

| Hook | Fires when | Return |
|---|---|---|
| `OnFieldsCalled` | `WithFields` is called | `Fields` to merge; nil drops the call |
| `OnMetadataCalled` | `WithMetadata` / `MetadataOnly` | `any` metadata; nil drops it |
| `OnBeforeDataOut` | per emission, after data assembly | `Data` to merge into the entry |
| `OnBeforeMessageOut` | per emission, after data hooks | replacement messages slice; nil keeps |
| `TransformLogLevel` | per emission, after the above | `(level, ok)`; ok=false leaves unchanged |
| `ShouldSend` | once per (entry, transport) | `false` to skip that transport |

`ShouldSend` is the only hook that sees the transport ID, so it's the place to gate dispatch per-transport.

### Lifecycle

```
WithFields(...)            → OnFieldsCalled chain → fields merged onto logger
WithMetadata(...)          → OnMetadataCalled chain → metadata stored

emission (Info / Warn / ...)
  ├─ assemble data (fields + error)
  ├─ OnBeforeDataOut chain          (mutate the data map)
  ├─ OnBeforeMessageOut chain       (mutate the messages slice)
  ├─ TransformLogLevel chain        (last ok=true wins)
  └─ for each transport:
       ├─ ShouldSend (false → skip this transport)
       └─ transport.SendToLogger(...)
```

Plugins run in the order they were added. `OnFieldsCalled` and `OnMetadataCalled` short-circuit on the first nil return; the dispatch-time hooks all run.

### Child loggers inherit plugins

`Child()` (and `WithFields`, `WithPrefix`) clones the plugin set by reference. Once either side mutates, the snapshots fork — copy-on-write. Adding a plugin to the child does not affect the parent.

## Picking the right hook

| You want to…                                    | Use                  |
|-------------------------------------------------|----------------------|
| Redact / rewrite per-call metadata              | `OnMetadataCalled`   |
| Redact / rewrite logger fields                  | `OnFieldsCalled`     |
| Add or rewrite assembled output keys            | `OnBeforeDataOut`    |
| Rewrite the message text                        | `OnBeforeMessageOut` |
| Promote / demote the level for some entries     | `TransformLogLevel`  |
| Drop entries from one specific transport        | `ShouldSend`         |
| Drop entries from the logger entirely           | `ShouldSend` returning false for every transport, or a level filter |

## Hook reference

::: warning Nil-return semantics differ by hook
The two **input-side** hooks treat a nil return as "drop the input." The four **dispatch-time** hooks treat a nil return as "no transformation." This asymmetry is intentional but easy to misremember.

| Hook | Returning nil means |
|---|---|
| `OnFieldsCalled` | Drop the WithFields call (receiver's existing fields preserved) |
| `OnMetadataCalled` | Drop metadata for this entry |
| `OnBeforeDataOut` | Leave the assembled data unchanged |
| `OnBeforeMessageOut` | Leave the messages unchanged |
| `TransformLogLevel` | (Returns `(_, false)` instead of nil) Leave the level unchanged |
| `ShouldSend` | (Not applicable — returns `bool`) |

The split: **input-side hooks** fire from `WithFields` / `WithMetadata`, where the user explicitly attached a value. Returning nil there is a meaningful "drop." **Output-side hooks** fire during dispatch, often from plugins that only want to transform sometimes. Returning nil there means "I don't have an opinion about this entry" rather than "drop everything."
:::

### `OnFieldsCalled(fields Fields) Fields`

Fires from `WithFields`. You receive the fields about to be merged onto the logger. Return the fields to actually merge; return `nil` to drop the call (the receiver's existing fields are preserved either way).

```go
loglayer.Plugin{
    ID: "uppercase-keys",
    OnFieldsCalled: func(fields loglayer.Fields) loglayer.Fields {
        out := make(loglayer.Fields, len(fields))
        for k, v := range fields {
            out["U_"+k] = v
        }
        return out
    },
}
```

### `OnMetadataCalled(metadata any) any`

Fires from `WithMetadata` and `MetadataOnly`. The metadata can be any value (map, struct, scalar). If you only handle one shape, type-assert and pass through anything you don't understand:

```go
loglayer.Plugin{
    ID: "redact-password",
    OnMetadataCalled: func(metadata any) any {
        m, ok := metadata.(map[string]any)
        if !ok {
            return metadata
        }
        // Clone before mutating — m is the caller's map.
        out := make(map[string]any, len(m))
        for k, v := range m {
            out[k] = v
        }
        if _, has := out["password"]; has {
            out["password"] = "[REDACTED]"
        }
        return out
    },
}
```

Return `nil` to drop the metadata for that entry.

::: warning Don't mutate caller inputs
The `metadata` you receive is the value the user passed to `WithMetadata`. If you mutate it in place, the user's variable changes too — they may pass the same map into multiple log calls and be surprised when the second call already has redacted values. Clone before you mutate. The same applies to `OnFieldsCalled`.
:::

If your plugin needs to walk **structs and nested values** (not just top-level maps), see [Walking arbitrary inputs](#walking-arbitrary-inputs) below.

### `OnBeforeDataOut(BeforeDataOutParams) Data`

Fires per emission, after the data map is assembled (fields + error). Return a map of keys to merge into the entry's data. The returned map is **merged**, not substituted; nil leaves the data unchanged.

```go
loglayer.Plugin{
    ID: "tag",
    OnBeforeDataOut: func(p loglayer.BeforeDataOutParams) loglayer.Data {
        return loglayer.Data{
            "service": "checkout",
            "version": buildVersion,
        }
    },
}
```

### `OnBeforeMessageOut(BeforeMessageOutParams) []any`

Fires per emission, after `OnBeforeDataOut`. Return a replacement messages slice; nil leaves messages unchanged.

```go
loglayer.Plugin{
    ID: "no-newlines",
    OnBeforeMessageOut: func(p loglayer.BeforeMessageOutParams) []any {
        out := make([]any, len(p.Messages))
        for i, m := range p.Messages {
            if s, ok := m.(string); ok {
                out[i] = strings.ReplaceAll(s, "\n", " ")
            } else {
                out[i] = m
            }
        }
        return out
    },
}
```

### `TransformLogLevel(TransformLogLevelParams) (LogLevel, bool)`

Fires per emission, after the message hooks. Return `(level, true)` to override the entry's level; return `(_, false)` to leave it unchanged.

If multiple plugins return `ok=true`, the last one wins.

```go
loglayer.Plugin{
    ID: "promote-on-error-key",
    TransformLogLevel: func(p loglayer.TransformLogLevelParams) (loglayer.LogLevel, bool) {
        if _, hasErr := p.Data["err"]; hasErr && p.LogLevel < loglayer.LogLevelError {
            return loglayer.LogLevelError, true
        }
        return 0, false
    },
}
```

`TransformLogLevel` happens **after** the per-method level filter. An `Info` call that's filtered out by `SetLevel(LogLevelWarn)` doesn't reach this hook; you can't use it to "rescue" an entry that the core already dropped.

### `ShouldSend(ShouldSendParams) bool`

Fires once per (entry, transport) pair, just before dispatch. Return `false` to skip that transport.

```go
loglayer.Plugin{
    ID: "no-debug-to-shipping",
    ShouldSend: func(p loglayer.ShouldSendParams) bool {
        if p.TransportID == "shipping" && p.LogLevel == loglayer.LogLevelDebug {
            return false
        }
        return true
    },
}
```

If multiple plugins define `ShouldSend`, the entry goes only when **every** plugin returns true.

## Per-call `context.Context`

All four dispatch-time hooks (`OnBeforeDataOut`, `OnBeforeMessageOut`, `TransformLogLevel`, `ShouldSend`) receive a `Ctx context.Context` field on their params, populated from `WithCtx(ctx)`. The value is `nil` when the user didn't attach a context.

Use it to:

- Read trace IDs / span IDs and inject them as fields:

  ```go
  OnBeforeDataOut: func(p loglayer.BeforeDataOutParams) loglayer.Data {
      if span := trace.SpanFromContext(p.Ctx); span.SpanContext().IsValid() {
          return loglayer.Data{
              "trace_id": span.SpanContext().TraceID().String(),
              "span_id":  span.SpanContext().SpanID().String(),
          }
      }
      return nil
  }
  ```

- Skip dispatch when the caller's context is cancelled:

  ```go
  ShouldSend: func(p loglayer.ShouldSendParams) bool {
      if p.Ctx == nil {
          return true
      }
      return p.Ctx.Err() == nil
  }
  ```

`OnMetadataCalled` and `OnFieldsCalled` do **not** receive a context — they fire from the builder phase, before the `WithCtx` call has been chained. If you need context-aware metadata mutation, do it from `OnBeforeDataOut` instead.

## Walking arbitrary inputs

`OnMetadataCalled` receives `any`. Real call sites pass maps, structs, pointers, slices, and scalars interchangeably. Any plugin that wants to "look inside" the value (redact, sanitize, rename, audit) faces the same problem: handle every shape uniformly without mutating the caller's input.

Three recipes apply, depending on what you want to handle and what you want the output to look like. The shared `utils/maputil` package gives you the primitives.

### Recipe 1: stay map-only

If your plugin only meaningfully operates on `map[string]any`, type-assert and pass through everything else:

```go
OnMetadataCalled: func(metadata any) any {
    m, ok := metadata.(map[string]any)
    if !ok {
        return metadata // structs, scalars, slices pass through unchanged
    }
    return cloneAndTransform(m)
}
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

OnMetadataCalled: func(metadata any) any {
    return cloner.Clone(metadata)
}
```

`Cloner` handles maps (string-keyed), structs (json-tag aware), slices, arrays, pointers, and interface values. It skips unexported fields. Caller's input is never mutated.

The [`plugins/redact`](/plugins/redact) plugin is built on `Cloner`; [its source](https://github.com/loglayer/loglayer-go/blob/main/plugins/redact/redact.go) is the canonical reference for this pattern.

### Recipe 3: normalize to a map first

If the **shape** matters more than preserving the user's runtime type, use [`maputil.ToMap`](https://pkg.go.dev/go.loglayer.dev/utils/maputil#ToMap) to JSON-roundtrip the input, then walk the resulting map.

```go
import "go.loglayer.dev/utils/maputil"

OnMetadataCalled: func(metadata any) any {
    m := maputil.ToMap(metadata)
    if m == nil {
        return metadata
    }
    return walkMap(m)
}
```

The trade-off: the metadata reaches downstream plugins and transports as a `map[string]any`, not the user's struct. Anything that type-switches on `params.Metadata` will see a map. For most rendering paths this is invisible (they marshal to JSON anyway), but tests that compare to the original struct break.

## Concurrency and performance

Hooks run on the dispatching goroutine. They may be called from any goroutine concurrently — the same plugin instance can fire on many emissions in parallel. Make any state your hook touches safe for concurrent reads/writes (use a mutex, atomics, or build the plugin from immutable config).

**Don't block in a hook**: it stalls the log call.

- Map lookups, string comparisons, simple type assertions: fine.
- Network or disk I/O: never. If you need to ship to an external system, enqueue to a channel and have a worker drain it.
- Reflection or JSON-encoding for every entry: usually too slow at high log volume; cache or precompute where you can.

The dispatcher pre-indexes hook membership at `AddPlugin` time, so having ten plugins where only one implements `OnBeforeDataOut` costs roughly the same as having one such plugin. You don't pay for hooks you don't implement.

## Convention: package shape

If you publish a plugin as a Go package, follow this shape:

```
yourpkg/
├── go.mod (if separate module)
├── plugin.go        // package yourpkg; exposes Config + New(Config) loglayer.Plugin
├── plugin_test.go
└── README.md
```

The constructor signature `func New(Config) loglayer.Plugin` matches the [`plugins/redact`](/plugins/redact) reference plugin and the constructor pattern transports use.

## See also

- [Plugins overview](/plugins/) — what hooks exist, when each fires, lifecycle and thread-safety semantics.
- [`plugins/redact`](/plugins/redact) — built-in reference plugin built on `maputil.Cloner`.
