# LogLayer Go - Implementation Plan

## Goals

Port the core of the TypeScript `loglayer` library to Go, preserving the API shape as closely
as the language allows. **First version excludes:** plugins, field managers, log level managers.
Groups support is also deferred (complexity/value tradeoff for v1).

---

## Directory Structure

Single Go module (`go.loglayer.dev/loglayer`) with sub-packages per component.
GitHub repo: `go.loglayer.dev/loglayer`
Vanity redirect at `go.loglayer.dev` points Go toolchain to the GitHub repo.
This mirrors the TypeScript monorepo's logical layout and gives each future feature its
own package with a clean import path, without the overhead of separate `go.mod` files
at this stage.

```
loglayer-golang/
├── go.mod                          # module go.loglayer.dev/loglayer
├── go.sum
├── docs/                           # Documentation (markdown, examples)
│
│   ── Core (v1: implemented) ──────────────────────────────────────────
│
├── loglayer.go                     # LogLayer struct + all public methods
├── builder.go                      # LogBuilder fluent chain
├── types.go                        # Public types, interfaces, constants
├── level.go                        # LogLevel, priority, enable/disable logic
├── loglayer_test.go
│
│   ── Transport layer ──────────────────────────────────────────────────
│
├── transport/                      # Transport interface + BaseTransport
│   ├── transport.go
│   └── transport_test.go
│
├── transports/                     # Built-in transport implementations
│   ├── console/                    # v1: ConsoleTransport
│   │   ├── console.go
│   │   └── console_test.go
│   ├── structured/                 # v1: StructuredTransport
│   │   ├── structured.go
│   │   └── structured_test.go
│   └── testing/                    # v1: TestTransport + TestLoggingLibrary
│       ├── testing.go
│       └── testing_test.go
│
│   ── Plugin system (future) ───────────────────────────────────────────
│
├── plugin/                         # IPlugin interface + PluginManager
│   └── .gitkeep
├── plugins/                        # Official plugin implementations
│   └── .gitkeep                    #   e.g. plugins/redact, plugins/filter
│
│   ── Field managers (future) ──────────────────────────────────────────
│
├── fieldmanager/                   # IFieldManager interface
│   └── .gitkeep
├── fieldmanagers/                  # Alternative field manager implementations
│   └── .gitkeep                    #   e.g. fieldmanagers/linked, /isolated
│
│   ── Log level managers (future) ──────────────────────────────────────
│
├── loglevelmanager/                # ILogLevelManager interface
│   └── .gitkeep
├── loglevelmanagers/               # Alternative log level manager implementations
│   └── .gitkeep                    #   e.g. loglevelmanagers/global, /linked
│
│   ── Framework integrations (future) ──────────────────────────────────
│
└── integrations/                   # Middleware / framework adapters
    └── .gitkeep                    #   e.g. integrations/echo, /gin, /fiber
```

### Import paths (examples)

```go
import "go.loglayer.dev/loglayer"                          // core
import "go.loglayer.dev/loglayer/transport"                // transport interface
import "go.loglayer.dev/loglayer/transports/console"       // console transport
import "go.loglayer.dev/loglayer/transports/structured"    // structured transport
import "go.loglayer.dev/loglayer/transports/testing"       // test helpers
```

### Why single module vs workspace?

Using separate `go.mod` per component (workspace) would allow independent versioning
but adds significant overhead for a project at this stage. Starting with a single module
is idiomatic and sufficient — if independent versioning becomes necessary (e.g. a transport
with heavy external deps) a sub-module can be split out later without breaking callers.

The root package declaration is `package loglayer` so callers write:
```go
log := loglayer.New(loglayer.Config{...})
```

---

## Core Types (types.go)

```go
// Log levels as constants (int ordering enables priority comparison)
type LogLevel int

const (
    LogLevelTrace LogLevel = 10
    LogLevelDebug LogLevel = 20
    LogLevelInfo  LogLevel = 30
    LogLevelWarn  LogLevel = 40
    LogLevelError LogLevel = 50
    LogLevelFatal LogLevel = 60
)

// String names used in transport output
func (l LogLevel) String() string { ... }

// Data types
type Context  = map[string]any   // persistent per-logger
type Metadata = map[string]any   // per-log
type Data     = map[string]any   // assembled final object

// Error serializer function type
type ErrorSerializer func(err error) map[string]any

// Transport params passed to shipToLogger
type TransportParams struct {
    LogLevel LogLevel
    Messages []any
    Data     Data   // assembled context + metadata + error
    HasData  bool
    Metadata Metadata
    Err      error
    Context  Context
}

// Config for LogLayer constructor
type Config struct {
    Transport         Transport   // required (single)
    Transports        []Transport // required (multiple) - one of these must be set
    Prefix            string
    Enabled           *bool       // defaults to true
    ErrorSerializer   ErrorSerializer
    ErrorFieldName    string      // default: "err"
    CopyMsgOnOnlyError bool
    ErrorFieldInMetadata bool
    ContextFieldName  string
    MetadataFieldName string
    MuteContext       bool
    MuteMetadata      bool
}

// ErrorOnly options
type ErrorOnlyOpts struct {
    LogLevel LogLevel
    CopyMsg  *bool
}

// RawLogEntry for the Raw() method
type RawLogEntry struct {
    LogLevel LogLevel
    Messages []any
    Metadata Metadata
    Err      error
    Context  Context
}
```

---

## Transport Interface (transport.go)

```go
type Transport interface {
    ID() string
    Enabled() bool
    SendToLogger(params TransportParams)
    ShipToLogger(params TransportParams)
    GetLoggerInstance() any
}

// BaseTransport provides default SendToLogger + level filtering.
// Concrete transports embed BaseTransport and implement ShipToLogger.
type BaseTransport struct {
    id      string
    enabled bool
    level   LogLevel  // minimum level, default Trace
}

func (b *BaseTransport) ID() string        { return b.id }
func (b *BaseTransport) Enabled() bool     { return b.enabled }
func (b *BaseTransport) SetEnabled(v bool) { b.enabled = v }
func (b *BaseTransport) SendToLogger(params TransportParams) {
    if !b.enabled { return }
    if params.LogLevel < b.level { return }
    b.ShipToLogger(params)  // calls concrete implementation via interface
}
```

Note: Go doesn't support virtual dispatch through embedding cleanly.
`BaseTransport` will hold a reference to the outer `Transport` interface so
`SendToLogger` can call `ShipToLogger` on the concrete type. Alternatively,
`SendToLogger` is implemented on each concrete transport (DRY via a shared helper).

**Decision:** Use a helper function `defaultSendToLogger(t Transport, params)` that
checks `Enabled()` + level, then calls `t.ShipToLogger(params)`.
Each concrete transport's `SendToLogger` calls this helper.

---

## Built-in Transports

### ConsoleTransport (transports/console.go)

Config:
```go
type ConsoleTransportConfig struct {
    ID               string
    Enabled          *bool
    Level            LogLevel
    AppendObjectData bool       // append vs prepend data object
    MessageField     string     // if set: structured output with msg in this field
    DateField        string
    LevelField       string
    DateFn           func() string
    LevelFn          func(LogLevel) string
    Stringify        bool       // JSON.Marshal the output object
    MessageFn        func(TransportParams) string
}
```

Output strategy:
- If `MessageField` set: build object with msg/date/level + data, print as single arg
- If `DateField` or `LevelField` only: print original messages + extra object
- Otherwise: prepend (or append) data object to messages

Console methods used: `fmt.Fprintln` writing to `os.Stderr` (error/fatal/warn) or
`os.Stdout` (info/debug/trace), using log-level-appropriate prefix.

**Simpler option:** use `log/slog` as the underlying console writer or just `fmt`.
We'll use `fmt` directly to match TS behavior (no extra timestamp from `log` package).

### StructuredTransport (transports/structured.go)

Config:
```go
type StructuredTransportConfig struct {
    ID           string
    Enabled      *bool
    Level        LogLevel
    MessageField string     // default "msg"
    DateField    string     // default "time"
    LevelField   string     // default "level"
    DateFn       func() string
    LevelFn      func(LogLevel) string
    Stringify    bool
    MessageFn    func(TransportParams) string
    Writer       io.Writer  // default os.Stdout; allows test injection
}
```

Always outputs a structured object — messages joined with space into MessageField.

### TestTransport + TestLoggingLibrary (transports/testing.go)

```go
type LogLine struct {
    Level    LogLevel
    Data     []any
}

type TestLoggingLibrary struct {
    mu    sync.Mutex
    Lines []LogLine
}

func (l *TestLoggingLibrary) Log(level LogLevel, args ...any)
func (l *TestLoggingLibrary) GetLastLine() *LogLine
func (l *TestLoggingLibrary) PopLine() *LogLine
func (l *TestLoggingLibrary) ClearLines()

type TestTransport struct {
    BaseTransport
    Library *TestLoggingLibrary
}
```

---

## LogLayer Core (loglayer.go)

```go
type LogLayer struct {
    config           Config
    context          Context
    transports       []Transport
    transportMap     map[string]Transport   // id -> transport
    singleTransport  Transport              // nil if multiple
    enabled          bool
    levelStatus      map[LogLevel]bool
    prefix           string
    muteContext      bool
    muteMetadata     bool
}

func New(config Config) *LogLayer { ... }
func (l *LogLayer) Child() *LogLayer { ... }
func (l *LogLayer) WithPrefix(prefix string) *LogLayer { ... }

// Context
func (l *LogLayer) WithContext(ctx Context) *LogLayer { ... }
func (l *LogLayer) ClearContext(keys ...string) *LogLayer { ... }
func (l *LogLayer) GetContext() Context { ... }
func (l *LogLayer) MuteContext() *LogLayer { ... }
func (l *LogLayer) UnmuteContext() *LogLayer { ... }

// Level control
func (l *LogLayer) SetLevel(level LogLevel) *LogLayer { ... }
func (l *LogLayer) IsLevelEnabled(level LogLevel) bool { ... }
func (l *LogLayer) EnableLogging() *LogLayer { ... }
func (l *LogLayer) DisableLogging() *LogLayer { ... }
func (l *LogLayer) EnableLevel(level LogLevel) *LogLayer { ... }
func (l *LogLayer) DisableLevel(level LogLevel) *LogLayer { ... }

// Transport management
func (l *LogLayer) AddTransport(t ...Transport) *LogLayer { ... }
func (l *LogLayer) RemoveTransport(id string) bool { ... }
func (l *LogLayer) WithFreshTransports(t ...Transport) *LogLayer { ... }
func (l *LogLayer) GetLoggerInstance(id string) any { ... }

// Logging
func (l *LogLayer) Info(messages ...any) { ... }
func (l *LogLayer) Warn(messages ...any) { ... }
func (l *LogLayer) Error(messages ...any) { ... }
func (l *LogLayer) Debug(messages ...any) { ... }
func (l *LogLayer) Trace(messages ...any) { ... }
func (l *LogLayer) Fatal(messages ...any) { ... }

// Builder entry points
func (l *LogLayer) WithMetadata(metadata Metadata) *LogBuilder { ... }
func (l *LogLayer) WithError(err error) *LogBuilder { ... }

// Special
func (l *LogLayer) ErrorOnly(err error, opts ...ErrorOnlyOpts) { ... }
func (l *LogLayer) MetadataOnly(metadata Metadata, level ...LogLevel) { ... }
func (l *LogLayer) Raw(entry RawLogEntry) { ... }

// Mute metadata
func (l *LogLayer) MuteMetadata() *LogLayer { ... }
func (l *LogLayer) UnmuteMetadata() *LogLayer { ... }
```

---

## LogBuilder (builder.go)

Fluent chain, holds accumulated state until a log level method is called.

```go
type LogBuilder struct {
    layer    *LogLayer
    metadata Metadata
    err      error
}

func (b *LogBuilder) WithMetadata(metadata Metadata) *LogBuilder { ... }
func (b *LogBuilder) WithError(err error) *LogBuilder { ... }
func (b *LogBuilder) Info(messages ...any) { b.layer.formatLog(...) }
func (b *LogBuilder) Warn(messages ...any) { ... }
func (b *LogBuilder) Error(messages ...any) { ... }
func (b *LogBuilder) Debug(messages ...any) { ... }
func (b *LogBuilder) Trace(messages ...any) { ... }
func (b *LogBuilder) Fatal(messages ...any) { ... }
```

---

## Internal Log Flow

```
Info("msg") / WithMetadata({}).Info("msg")
  ↓
isLevelEnabled check → return if false
  ↓
applyPrefix(messages)
  ↓
formatLog(logLevel, messages, metadata, err, context)
  ↓
assembleData():
  - formatContext() → respect contextFieldName + muteContext
  - formatMetadata() → respect metadataFieldName + muteMetadata
  - merge into Data map
  - attach error under errorFieldName (handle errorFieldInMetadata)
  ↓
dispatchToTransports(logLevel, messages, data, hasData, err, metadata, ctx):
  - for each enabled transport:
      transport.SendToLogger(TransportParams{...})
```

---

## Level Management (level.go)

No external manager — embedded directly in LogLayer for v1 simplicity.

```go
// Initial state: all levels enabled
// SetLevel(warn) → enables warn/error/fatal, disables trace/debug/info
// EnableLevel / DisableLevel → toggle individual levels
// DisableLogging / EnableLogging → disable/enable all
```

---

## Error Handling

- `ErrorSerializer` in config: `func(error) map[string]any`
  - Default: `{"message": err.Error()}` (since Go errors don't serialize to objects automatically)
- `ErrorFieldName` default: `"err"`
- `CopyMsgOnOnlyError`: use `err.Error()` as message when calling `ErrorOnly`
- `ErrorFieldInMetadata`: nest error inside MetadataFieldName

---

## Key Go-specific Decisions

| TypeScript | Go equivalent |
|---|---|
| `any` metadata/context | `map[string]any` |
| Union types / type guards | interfaces + type assertions |
| Optional chaining `?.` | nil checks |
| `Promise<void>` for async lazy | Not supported in v1 (no lazy evaluation) |
| Variadic spread `...messages` | `...any` |
| Class inheritance | Struct embedding + interfaces |
| `Symbol.dispose` | Not needed for v1 |
| Multiple transports via `Promise.all` | Sequential (goroutines optional later) |

**Lazy evaluation:** Deferred to a future version. Go's value semantics make this
less common, and the complexity (goroutines, error channels) is high for v1.

---

## Testing Strategy

Each file gets a `_test.go` sibling. Tests use `TestLoggingLibrary` + `TestTransport`
for capturing log output without I/O.

Test coverage targets:
- Core log methods (info/warn/error/debug/trace/fatal)
- withMetadata + withError builder chains
- Context: set, append, clear, mute/unmute
- Level control: setLevel, enableLevel, disableLevel, enable/disableLogging
- Error serialization (default + custom)
- errorOnly, metadataOnly, raw
- Prefix prepending
- child() inheritance and isolation
- Transport management: add, remove, replace
- ConsoleTransport output format variants
- StructuredTransport output format variants
- Multiple transports dispatch

---

## Out of Scope (v1)

- Plugin system
- Field managers (LinkedFieldManager, IsolatedFieldManager)
- Log level managers (GlobalLogLevelManager, LinkedLogLevelManager)
- Lazy / async lazy evaluation
- Group routing system
- `LOGLAYER_GROUPS` env var
- Mixins
- 30+ community transports (those are separate packages/repos)
