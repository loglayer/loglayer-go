---
title: Plugin Configuration
description: Wire plugins into a LogLayer at construction time and pick the right plugin IDs.
---

# Plugin Configuration

Plugins are wired into a `*loglayer.LogLayer` at construction time via the `Config.Plugins` field. This page covers construction-time setup. For runtime mutation (add, remove, replace at runtime), see [Plugin Management](/plugins/management). For the hook reference and lifecycle, see [Creating Plugins](/plugins/creating-plugins).

## Register at construction time

```go
log := loglayer.New(loglayer.Config{
    Transport: structured.New(structured.Config{}),
    Plugins:   []loglayer.Plugin{redact.New(...), audit.New(...)},
})
```

Plugin order matters: hooks run in the order plugins were added, and each plugin sees the previous plugin's output.

## Plugin IDs

A plugin's `ID()` method may return the empty string; LogLayer assigns an auto-generated identifier at registration time so multiple no-ID plugins never collide. **Supply your own ID** when you intend to call `RemovePlugin` / `GetPlugin` later, or want a readable identifier in logs and tooling: those operations key off the string.

For inline plugins via the adapter constructors, the first argument is the ID:

```go
log.AddPlugin(loglayer.NewDataHook("audit", func(p loglayer.BeforeDataOutParams) loglayer.Data {
    return loglayer.Data{"audited": true}
}))
```

For plugins you set up once and never touch (the common case), leaving the ID empty is fine. The auto-generated ID is still unique within the logger; you just won't have a stable handle for management calls.

## See Also

- [Plugin Management](/plugins/management), runtime mutation of the plugin set.
- [Creating Plugins](/plugins/creating-plugins), authoring a plugin and the hook reference.
