### Renderers

Self-contained transports that format the entry and write it to an `io.Writer`. Pick one of these when you want LogLayer to do the rendering itself.

| Name                                  | Description                                                                |
|---------------------------------------|----------------------------------------------------------------------------|
| [Pretty](/transports/pretty)          | Colorized, theme-aware terminal output. **Recommended for local dev.**     |
| [Structured](/transports/structured)  | One JSON object per log entry. Recommended for production.                 |
| [Console](/transports/console)        | Plain `fmt.Println`-style output to stdout/stderr; minimal formatting.     |
| [Testing](/transports/testing)        | Captures entries in memory for tests.                                      |
| [Blank](/transports/blank)            | Delegates dispatch to a user-supplied function. For prototyping or one-off integrations. |

### Network

Transports that ship log entries to a remote endpoint over the network. Async + batched by default.

| Name                                   | Description                                                              |
|----------------------------------------|--------------------------------------------------------------------------|
| [HTTP](/transports/http)               | Generic batched HTTP POST to any endpoint. Pluggable Encoder.            |
| [Datadog](/transports/datadog)         | Datadog Logs HTTP intake. Site-aware URL, DD-API-KEY header, status mapping. |

### Supported Loggers

Transports that hand the entry off to an existing third-party logger you already configure. Pick one of these when you have an established logging stack and want LogLayer's API on top.

| Name                                | Description                                                |
|-------------------------------------|------------------------------------------------------------|
| [Zerolog](/transports/zerolog)      | Wraps a `*zerolog.Logger`                                  |
| [Zap](/transports/zap)              | Wraps a `*zap.Logger`                                      |
| [log/slog](/transports/slog)        | Wraps a stdlib `*slog.Logger`. Forwards `WithCtx` to handlers. |
| [phuslu/log](/transports/phuslu)    | High-performance zero-alloc JSON logger. Always exits on fatal. |
| [logrus](/transports/logrus)        | The classic structured logger                              |
| [charmbracelet/log](/transports/charmlog) | Pretty terminal-friendly logger from Charm           |
