`loglayer.New(Config) *LogLayer` is the typical entry point: it panics on misconfiguration (matches Go convention for setup-time errors). For applications that prefer explicit error handling on missing or invalid config, use `loglayer.Build(Config) (*LogLayer, error)`, which returns `loglayer.ErrNoTransport`, `ErrTransportAndTransports`, or `ErrUngroupedTransportsWithoutMode` instead of panicking.

```go
// Panics on misconfiguration (typical setup).
log := loglayer.New(loglayer.Config{
    Transport: structured.New(structured.Config{}),
})

// Or explicit error handling.
log, err := loglayer.Build(loglayer.Config{
    Transport: structured.New(structured.Config{}),
})
```
