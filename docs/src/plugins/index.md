---
title: Plugins
description: Hook into the LogLayer pipeline to transform metadata, fields, data, messages, log level, or per-transport dispatch.
---

# Plugins

Plugins let you hook into LogLayer's pipeline to transform what gets logged or to selectively veto dispatch. They run on every `*loglayer.LogLayer`; add them once at construction and they apply to every emission.

A plugin is a [`loglayer.Plugin`](https://pkg.go.dev/go.loglayer.dev#Plugin) struct with one or more hook function fields populated. Nil fields are skipped.

## Quick Example

```go
log := loglayer.New(loglayer.Config{
    Transport: structured.New(structured.Config{}),
})

log.AddPlugin(loglayer.Plugin{
    ID: "request-tag",
    OnBeforeDataOut: func(p loglayer.BeforeDataOutParams) loglayer.Data {
        // Add a static field to every log entry.
        return loglayer.Data{"app": "checkout-api"}
    },
})

log.Info("served")
// {"level":"info","msg":"served","app":"checkout-api","time":"..."}
```

## Hooks

Six hooks fire during emission. Implement only the ones you need.

| Hook                 | Fires when                          | Return                                   |
|----------------------|-------------------------------------|------------------------------------------|
| `OnFieldsCalled`     | `WithFields` is called              | `Fields` to merge; nil drops the call    |
| `OnMetadataCalled`   | `WithMetadata` / `MetadataOnly`     | `any` metadata; nil drops it             |
| `OnBeforeDataOut`    | per emission, after data assembly   | `Data` to merge into the entry           |
| `OnBeforeMessageOut` | per emission, after data hooks      | replacement messages slice; nil keeps    |
| `TransformLogLevel`  | per emission, after the above       | `(level, ok)`; ok=false leaves unchanged |
| `ShouldSend`         | once per (entry, transport)         | `false` to skip that transport           |

Lifecycle:

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

Plugins run in the order they were added.

## Ordering and Replacement

`AddPlugin` registers a plugin. If a plugin with the same `ID` already exists, it is **replaced** (matching the `AddTransport` convention). Adding the same plugin twice with the same ID is therefore a way to update its config in place.

```go
log.AddPlugin(loglayer.Plugin{ID: "redact", ...})  // first add
log.AddPlugin(loglayer.Plugin{ID: "redact", ...})  // replaces, doesn't duplicate
log.PluginCount()                                  // 1
```

`RemovePlugin(id)` removes a plugin and returns whether it was present.

```go
log.RemovePlugin("redact")  // returns true
log.RemovePlugin("ghost")   // returns false
```

`GetPlugin(id)` returns a copy of the registered plugin (and an `ok bool`).

## Per-transport selective dispatch

`ShouldSend` is the only hook that sees the transport ID, so it's the right place to gate dispatch per-transport. A typical use: send debug entries to console but not to a shipping transport.

```go
log.AddPlugin(loglayer.Plugin{
    ID: "drop-debug-from-shipping",
    ShouldSend: func(p loglayer.ShouldSendParams) bool {
        if p.TransportID == "shipping" && p.LogLevel == loglayer.LogLevelDebug {
            return false
        }
        return true
    },
})
```

Multiple `ShouldSend` plugins all have to allow the entry; any one returning false vetoes that transport.

## Thread Safety

`AddPlugin`, `RemovePlugin`, `GetPlugin`, and `PluginCount` are safe to call from any goroutine, including concurrently with emission. The plugin set is published atomically; concurrent mutators on the same logger serialize via an internal mutex.

Hooks themselves run on the dispatching goroutine. **Don't block in a hook**: it stalls the log call. If you need to do I/O or slow work in a plugin, do it asynchronously (e.g., enqueue to a channel that a worker drains) and return promptly.

## Child Loggers Inherit Plugins

`Child()` (and `WithFields`, `WithPrefix`) clones the plugin list. Adding a plugin to the child does not affect the parent.

```go
parent.AddPlugin(loglayer.Plugin{ID: "shared", ...})
child := parent.Child()
child.AddPlugin(loglayer.Plugin{ID: "child-only", ...})

parent.PluginCount()  // 1
child.PluginCount()   // 2
```

## First-party plugins

- [`plugins/redact`](/plugins/redact) — replace values for a configured set of keys before metadata or fields reach a transport.

## See also

- [Creating Plugins](/plugins/creating-plugins) — design patterns and the full API reference for each hook.
