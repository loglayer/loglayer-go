---
title: Testing Transports
description: Helpers for testing custom LogLayer transport implementations.
---

# Testing Transports

`transport/transporttest` is the standard testing helper for transport authors. It exports `ParseJSONLine` for asserting on JSON output and `RunContract` for the wrapper-transport contract suite that every built-in wrapper passes.

## Direct buffer assertions

Drive entries through your transport via a real `*loglayer.LogLayer` and assert on whatever your transport actually produced (a buffer, a captured request, a wrapped logger's calls). The pattern mirrors the built-in transport tests:

```go
import (
    "bytes"
    "testing"

    "go.loglayer.dev"
    "go.loglayer.dev/transport"
    "go.loglayer.dev/transport/transporttest"
)

func TestMyTransport_Basic(t *testing.T) {
    buf := &bytes.Buffer{}
    tr := mytransport.New(mytransport.Config{
        BaseConfig: transport.BaseConfig{ID: "test"},
        Writer:     buf,
    })
    log := loglayer.New(loglayer.Config{
        Transport:        tr,
        DisableFatalExit: true,
    })

    log.WithFields(loglayer.Fields{"k": "v"}).Info("served")

    obj := transporttest.ParseJSONLine(t, buf)
    if obj["k"] != "v" {
        t.Errorf("k: got %v, want \"v\"", obj["k"])
    }
}
```

For wrapper transports (those that hand entries off to a third-party logger), assert on the wrapped logger's output rather than the transport's. The slog/zerolog/zap test files in `transports/` show this pattern.

## The wrapper contract suite

`transport/transporttest` ships a [`RunContract`](https://pkg.go.dev/go.loglayer.dev/transport/transporttest#RunContract) helper that drives 14 sub-tests against any wrapper-shaped transport (renders to a buffer in JSON-per-line). The same suite verifies every built-in wrapper. Wire it in with a `Factory` closure that builds a fresh `(*loglayer.LogLayer, *bytes.Buffer)` honoring per-test config overrides, plus an `Expectations` struct describing your wrapper's rendering quirks (message key, level rendering, fatal handling):

```go
func factory(opts transporttest.FactoryOpts) (*loglayer.LogLayer, *bytes.Buffer) {
    buf := &bytes.Buffer{}
    cfg := mytransport.Config{
        BaseConfig:        transport.BaseConfig{ID: "mywrap", Level: opts.Level},
        Logger:            buildBackend(buf),
        MetadataFieldName: opts.MetadataFieldName,
    }
    tr := mytransport.New(cfg)
    return loglayer.New(loglayer.Config{DisableFatalExit: true, Transport: tr}), buf
}

func TestMyWrapperContract(t *testing.T) {
    transporttest.RunContract(t, transporttest.ContractCase{
        Name:    "mywrap",
        Factory: factory,
        Expect: transporttest.Expectations{
            MessageKey: "msg",
            LevelKey:   "level",
            Levels: map[loglayer.LogLevel]string{
                loglayer.LogLevelDebug: "debug",
                loglayer.LogLevelInfo:  "info",
                loglayer.LogLevelWarn:  "warn",
                loglayer.LogLevelError: "error",
                loglayer.LogLevelFatal: "fatal",
            },
        },
    })
}
```

Set `Expectations.SkipFatal` for libraries that unconditionally call `os.Exit` on Fatal (the `phuslu` wrapper does this). Library-specific behavior (positive `WithCtx` forwarding for `slog`-style wrappers, etc.) goes in your own per-wrapper test functions next to the `RunContract` call.

Cover the level-filtering case, the `MetadataFieldName` non-map path, and `WithCtx` propagation when applicable. The existing wrapper-transport test files are good templates: same structure, same assertion shape.

## See Also

- [Creating Transports](/transports/creating-transports), implementing the `Transport` interface from scratch.
