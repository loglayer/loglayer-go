# Changelog

All notable changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

The repo is currently a single Go module: `go.loglayer.dev/loglayer`. All packages
move together under one tag. If a transport later needs an independent release
cadence we may split it into its own module; see `AGENTS.md` for the policy.

## [Unreleased] (target: v0.1.0)

Initial release. Pre-1.0; the API may still shift before v1.

### Core

- `loglayer.LogLayer` with the six standard log levels (Trace, Debug, Info,
  Warn, Error, Fatal) and a fluent builder API: `WithMetadata`, `WithError`,
  `WithCtx`, chained into a level method (`Info`, `Warn`, etc.).
- Persistent fields via `WithFields(loglayer.Fields)` and `ClearFields(keys...)`.
  Both return a new logger; the receiver is unchanged. Matches the convention
  of zerolog, zap, slog, and logrus.
- `loglayer.Fields` and `loglayer.Metadata` type aliases for `map[string]any`.
  `WithMetadata` accepts `any`; transports decide how to render structs and
  other types.
- `Child()` and `WithPrefix(prefix)` for explicit cloning.
- `Raw(RawLogEntry)` for forwarding pre-assembled entries from another system.
- `MetadataOnly` and `ErrorOnly` shortcuts.
- `loglayer.New(Config)` panics on misconfiguration; `loglayer.Build(Config)`
  returns `loglayer.ErrNoTransport` instead.
- `Config.DisableFatalExit` opts out of `os.Exit(1)` after a Fatal log
  (default-on, matching Go convention).
- `loglayer.NewMock()` returns a silent `*LogLayer` for tests with
  `DisableFatalExit` enabled automatically.
- Go context integration: `WithCtx(ctx)` per-call, surfaced via
  `TransportParams.Ctx`. `loglayer.NewContext(ctx, log)` and
  `loglayer.FromContext(ctx)` for embedding/retrieving a logger inside a Go
  context (zerolog-style middleware pattern). `MustFromContext` panics on
  absence.

### Thread safety

Every method on `*LogLayer` is safe to call from any goroutine, including
concurrently with emission. There is no setup-only category. See
`AGENTS.md#thread-safety` for the per-method breakdown:

- Returns-new methods (`WithFields`, `ClearFields`, `Child`, `WithPrefix`)
  build a new logger; receiver untouched.
- Level mutators backed by `atomic.Uint32` bitmap (mirrors `zap.AtomicLevel`).
- Transport mutators backed by `atomic.Pointer[transportSet]`; concurrent
  mutators on the same logger serialize via an internal mutex.
- Mute toggles backed by `atomic.Bool` state.

### Transports

Renderers (self-contained):

- `transports/console`: plain `fmt.Println`-style output to stdout/stderr.
- `transports/pretty`: colorized terminal output with five themes
  (Moonlight, Sunlight, Neon, Nature, Pastel) and three view modes
  (inline, message-only, expanded). Pulls in `github.com/fatih/color`.
- `transports/structured`: one JSON object per log entry.
- `transports/testing`: in-memory capture for test assertions.
- `transports/blank`: delegates `SendToLogger` to a user-supplied function.
  For prototyping or one-off integrations.

Logger wrappers:

- `transports/zerolog`: wraps `github.com/rs/zerolog`. Routes fatal entries
  through `WithLevel` so the core decides whether to exit.
- `transports/zap`: wraps `go.uber.org/zap`. Custom `CheckWriteHook`
  neutralizes zap's fatal-exit so the core decides via `DisableFatalExit`.
- `transports/slog`: wraps the stdlib `*log/slog.Logger`. Forwards `WithCtx`
  to `slog.Logger.LogAttrs`.
- `transports/phuslu`: wraps `github.com/phuslu/log`. Note: phuslu cannot be
  prevented from calling `os.Exit` on Fatal; documented as a limitation.
- `transports/logrus`: wraps `github.com/sirupsen/logrus`. Builds an internal
  copy with no-op `ExitFunc` so the user's logger is never mutated.
- `transports/charmlog`: wraps `github.com/charmbracelet/log`. Uses
  `Log(level, ...)` to defer the exit decision to the core.

### Integrations

- `integrations/loghttp`: HTTP middleware that derives a per-request logger
  with `requestId`/`method`/`path` fields, stores it in the request context,
  and emits a "request completed" log with status, bytes, and duration.
  Functional options for request-ID header, request-ID generator, field
  names, status-based level escalation, optional start log, extra-fields
  hook. Wraps any `net/http`-compatible router (chi, gorilla, gin, echo,
  stdlib).

[Unreleased]: https://github.com/loglayer/loglayer-go/commits/main
