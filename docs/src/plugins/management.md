---
title: Plugin Management
description: Add, remove, replace, inspect, and count plugins at runtime.
---

# Plugin Management

Plugins on a `*loglayer.LogLayer` can be added, removed, replaced, and inspected at runtime. All mutators are safe to call from any goroutine, including concurrently with emission.

For construction-time setup (`Config.Plugins`, ID semantics), see [Plugin Configuration](/plugins/configuration).

## Add

```go
log.AddPlugin(redact.New(...))
log.AddPlugin(redact.New(...), audit.New(...))   // multiple at once
log.AddPlugin(plugins...)                        // splat a slice
```

`AddPlugin` replaces by ID: re-adding the same ID updates the plugin in place. Plugins registered without an ID get an auto-generated one, so they can never be replaced by another no-ID plugin: each call adds a new entry.

## Remove, inspect, count

```go
log.RemovePlugin("redact")     // returns true if it was present
got, ok := log.GetPlugin("redact")
log.PluginCount()
```

## Worked example: replace and remove by ID

```go
log := loglayer.New(loglayer.Config{Transport: structured.New(structured.Config{})})

// Initial registration. The ID "audit" is the handle.
log.AddPlugin(loglayer.Plugin{
    ID: "audit",
    OnBeforeDataOut: func(p loglayer.BeforeDataOutParams) loglayer.Data {
        return loglayer.Data{"audited": true}
    },
})

// Hot-swap the plugin: same ID, new behavior. The previous "audit"
// plugin is discarded.
log.AddPlugin(loglayer.Plugin{
    ID: "audit",
    OnBeforeDataOut: func(p loglayer.BeforeDataOutParams) loglayer.Data {
        return loglayer.Data{"audited": true, "v": 2}
    },
})

// Tear it down at shutdown.
if log.RemovePlugin("audit") {
    fmt.Println("audit plugin removed")
}
```

Plugins added without an ID can't be retrieved or replaced this way: their auto-generated ID isn't surfaced anywhere. Use the ID-supplied form when management is part of the workflow.

## Concurrency

`AddPlugin` and `RemovePlugin` publish a new immutable plugin set via `atomic.Pointer`, so the dispatch hot path only loads a pointer. Concurrent mutators on the same logger serialize via an internal mutex. `Child()` clones the plugin set by reference; once either side mutates, the snapshots fork (copy-on-write).
