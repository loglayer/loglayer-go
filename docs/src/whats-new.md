---
title: What's New
description: User-visible changes to LogLayer for Go.
---

# What's New

Changes are recorded here once they ship. Until v0.1.0 lands, this page describes the API as it stands today.

## v0.1.0 (unreleased)

Initial release of LogLayer for Go: a transport-agnostic structured logging facade with a fluent API for messages, fields, metadata, and errors.

### Core

- Six log levels: Trace, Debug, Info, Warn, Error, Fatal.
- Fluent builder API: `WithMetadata`, `WithError`, `WithCtx`, chained into a level method (`Info`, `Warn`, etc.).
- `loglayer.Fields` (a `map[string]any` alias) for persistent, keyed data attached to a logger via `WithFields`. Returns a new logger on each call, matching the convention used by zerolog, zap, slog, and logrus.
- `ClearFields(keys ...string)` returns a new logger with those keys removed (or all fields when called with no arguments).
- `loglayer.Metadata` (a `map[string]any` alias) for the common shape passed to `WithMetadata`. `WithMetadata` accepts any value; transports decide how to serialize structs and other types.
- `Child()` and `WithPrefix(prefix)` for explicit cloning.
- `Raw(RawLogEntry)` for bypassing the builder when forwarding pre-assembled entries from another system.
- `MetadataOnly` and `ErrorOnly` shortcuts.

### Constructors

- `loglayer.New(Config) *LogLayer` panics on misconfiguration; `loglayer.Build(Config) (*LogLayer, error)` returns `loglayer.ErrNoTransport` instead.
- `Config.DisableFatalExit` opts out of the default `os.Exit(1)` after a Fatal log (Go convention).
- `loglayer.NewMock()` returns a silent `*LogLayer` for tests, with `DisableFatalExit` enabled automatically.

### Go context integration

- `WithCtx(ctx)` attaches a `context.Context` to a single log call; surfaced to transports via `TransportParams.Ctx`.
- `loglayer.NewContext(ctx, log)` and `loglayer.FromContext(ctx)` embed and retrieve a logger from a Go context (zerolog-style middleware pattern). `MustFromContext` panics on absence.

### Thread safety

Every method on `*LogLayer` is safe to call from any goroutine, including concurrently with emission. `WithFields`, `ClearFields`, `Child`, and `WithPrefix` return new loggers. Level mutators are backed by an atomic bitmap, transport mutators by an atomic pointer with an internal mutex on the slow path, mute toggles by atomic bools. Designed to support runtime patterns like SIGUSR1-driven debug toggling and hot-reloading transport lists without coordination.

### Transports

Renderers (self-contained):

- `transports/console`: plain `fmt.Println`-style output to stdout/stderr.
- `transports/pretty`: colorized terminal output with five themes (Moonlight default, Sunlight, Neon, Nature, Pastel) and three view modes (inline, message-only, expanded). Pulls in `github.com/fatih/color`.
- `transports/structured`: one JSON object per log entry. Recommended for production.
- `transports/testing`: in-memory capture into a typed `LogLine` for test assertions.
- `transports/blank`: delegates `SendToLogger` to a user-supplied function. For prototyping or one-off integrations.

Logger wrappers:

- `transports/zerolog`: wraps `github.com/rs/zerolog`. Routes fatal entries through `WithLevel` so the core decides whether to exit.
- `transports/zap`: wraps `go.uber.org/zap`. Custom `CheckWriteHook` neutralizes zap's fatal-exit so the core decides via `DisableFatalExit`.
- `transports/slog`: wraps the stdlib `*log/slog.Logger`. Forwards `WithCtx` to `slog.Logger.LogAttrs`.
- `transports/phuslu`: wraps `github.com/phuslu/log`. **Always exits on Fatal** regardless of `DisableFatalExit`; phuslu calls `os.Exit` from any fatal dispatch path.
- `transports/logrus`: wraps `github.com/sirupsen/logrus`. Builds an internal copy with no-op `ExitFunc` so the user's logger is never mutated.
- `transports/charmlog`: wraps `github.com/charmbracelet/log`. Uses `Log(level, ...)` so the core controls the exit decision.

### Integrations

- `integrations/loghttp`: HTTP middleware that derives a per-request logger from a base logger, attaches `requestId`/`method`/`path`, stores it in the request context via `loglayer.NewContext`, and emits a "request completed" log line with status, bytes, and duration. One line at server setup; downstream handlers retrieve the logger via `loghttp.FromRequest(r)`. Wraps any `net/http`-compatible router (chi, gorilla, gin, echo, stdlib). Functional options for request-ID header, request-ID generator, field names, status-based level escalation, optional start log, and an extra-fields hook.
