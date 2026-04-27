---
title: Plugins
description: Hook into the LogLayer pipeline to transform metadata, fields, data, messages, log level, or per-transport dispatch.
---

# Plugins

Plugins extend LogLayer's emission pipeline. They run on every `*loglayer.LogLayer` they're registered on and apply to every emission until removed.

A plugin is a [`loglayer.Plugin`](https://pkg.go.dev/go.loglayer.dev#Plugin) struct: an `ID` plus one or more hook function fields (nil fields are skipped). For the hook reference, lifecycle diagram, and how to write one, see [Creating Plugins](/plugins/creating-plugins).

## Available plugins

<!--@include: ./_partials/plugin-list.md-->

## Plugin management

Register plugins at construction time:

```go
log := loglayer.New(loglayer.Config{
    Transport: structured.New(structured.Config{}),
    Plugins:   []loglayer.Plugin{redact.New(...), audit.New(...)},
})
```

Or add them later:

```go
log.AddPlugin(redact.New(...))
log.AddPlugin(redact.New(...), audit.New(...))   // multiple at once
log.AddPlugin(plugins...)                        // splat a slice
```

`AddPlugin` replaces by ID — re-adding the same ID updates the plugin in place.

Remove, inspect, count:

```go
log.RemovePlugin("redact")     // returns true if it was present
got, ok := log.GetPlugin("redact")
log.PluginCount()
```

An empty `ID` panics with `loglayer.ErrPluginNoID` (or `Build` returns it).

All four mutators are safe to call from any goroutine, including concurrently with emission. `Child()` clones the plugin set by reference; once either side mutates, the snapshots fork (copy-on-write).
