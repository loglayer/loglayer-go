---
title: Testing Plugins
description: Helpers for testing custom LogLayer plugin implementations.
---

# Testing Plugins

`plugins/plugintest` is the standard testing helper for plugin authors. It wires your plugin into a fresh logger backed by the [`transports/testing`](/transports/testing) capture transport, so you drive scenarios through a real `*loglayer.LogLayer` and assert against captured `LogLine` entries.

## Installing the plugin

```go
import (
    "testing"

    "go.loglayer.dev"
    "go.loglayer.dev/plugins/plugintest"
)

func TestMyPlugin_AddsField(t *testing.T) {
    log, lib := plugintest.Install(t, myplugin.New(...))

    log.Info("served")

    line := lib.PopLine()
    if line.Data["my-field"] != "expected" {
        t.Errorf("my-field: got %v", line.Data["my-field"])
    }
}
```

`PopLine` returns the most recent entry and removes it; `Lines()` returns all captured. Both are `LogLine` structs with `LogLevel`, `Messages`, `Data`, `Metadata`, and `Err` fields. See [`transports/testing`](/transports/testing) for the full helper API.

## Verifying input-side hooks don't mutate input

`OnFieldsCalled` and `OnMetadataCalled` plugins must not mutate the caller's input. `plugintest.AssertNoMutation` deep-clones the input, runs the hook, and fails the test if the original differs from the snapshot afterward:

```go
plugintest.AssertNoMutation[any](t,
    redact.New(redact.Config{Keys: []string{"password"}}).OnMetadataCalled,
    map[string]any{"password": "hunter2", "user": "alice"},
)
```

## Verifying panic recovery

The framework recovers hook panics and forwards them to `Plugin.OnError` as `*loglayer.RecoveredPanicError`. Use `plugintest.AssertPanicRecovered` to verify both that the framework caught the panic and that your plugin's `Hook` field is set correctly:

```go
rpe := plugintest.AssertPanicRecovered(t,
    loglayer.Plugin{
        ID: "boom",
        OnBeforeDataOut: func(loglayer.BeforeDataOutParams) loglayer.Data {
            panic("kaboom")
        },
    },
    func(log *loglayer.LogLayer) { log.Info("trigger") },
)
// rpe.Hook == "OnBeforeDataOut"
// rpe.Value contains the original panic value (errors.Is works when it's an error)
```

`AssertPanicRecovered` overrides `Plugin.OnError` internally to capture, so leave it unset on the plugin you pass in.

## See Also

- [Creating Plugins](/plugins/creating-plugins), authoring a plugin and the hook reference.
