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

Plugins implementing `FieldsHook` or `MetadataHook` must not mutate the caller's input. `plugintest.AssertNoMutation` deep-clones the input, runs the hook function, and fails the test if the original differs from the snapshot afterward.

If the plugin under test is a defined type, type-assert it to the relevant hook interface to extract the method:

```go
p := redact.New(redact.Config{Keys: []string{"password"}})
plugintest.AssertNoMutation[any](t,
    p.(loglayer.MetadataHook).OnMetadataCalled,
    map[string]any{"password": "hunter2", "user": "alice"},
)
```

## Verifying panic recovery

The framework recovers hook panics and forwards them to a plugin's `OnError` (when the plugin implements [`loglayer.ErrorReporter`](https://pkg.go.dev/go.loglayer.dev#ErrorReporter)). Use `plugintest.AssertPanicRecovered` to drive a panicking hook and assert the framework forwarded a `*loglayer.RecoveredPanicError`.

The helper takes a builder closure that receives a `captureFn`: thread it through to your plugin's `OnError` so the framework's recovery path delivers the panic to the helper's capture.

```go
rpe := plugintest.AssertPanicRecovered(t,
    func(captureFn func(error)) loglayer.Plugin {
        return myplugin.New(myplugin.Config{
            // ... configure a hook that panics ...
            OnError: captureFn,
        })
    },
    func(log *loglayer.LogLayer) { log.Info("trigger") },
)
// rpe.Hook == "OnBeforeDataOut" (or whichever hook panicked)
// rpe.Value contains the original panic value (errors.Is works when it's an error)
```

If your plugin doesn't expose an `OnError` config field, build the plugin around a custom type that implements `ErrorReporter` directly and set its capture closure inside the builder.

## See Also

- [Creating Plugins](/plugins/creating-plugins), authoring a plugin and the hook reference.
