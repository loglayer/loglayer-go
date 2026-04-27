---
title: Groups
description: "Named routing rules for sending logs to specific transports based on tags."
---

# Groups

Groups are named routing rules that decide which transports receive which log entries. In a system with many subsystems (database, auth, payments) and many destinations (console, Datadog, Sentry), groups let you "listen to" specific categories without touching global level state.

The concept mirrors the [`withGroup` feature in the TypeScript loglayer](https://loglayer.dev/logging-api/groups.html).

## Configuration

Define groups when creating the logger:

```go
import (
    "go.loglayer.dev"
    "go.loglayer.dev/transports/structured"
    "go.loglayer.dev/transports/datadog"
    "go.loglayer.dev/transport"
)

log := loglayer.New(loglayer.Config{
    Transports: []loglayer.Transport{
        structured.New(structured.Config{BaseConfig: transport.BaseConfig{ID: "console"}}),
        datadog.New(datadog.Config{BaseConfig: transport.BaseConfig{ID: "datadog"}, APIKey: key}),
    },
    Groups: map[string]loglayer.LogGroup{
        "database": {
            Transports: []string{"datadog"},
            Level:      loglayer.LogLevelError,
        },
        "auth": {
            Transports: []string{"datadog"},
            Level:      loglayer.LogLevelWarn,
        },
    },
})
```

### `LogGroup`

| Field | Type | Default | Description |
|---|---|---|---|
| `Transports` | `[]string` | required | Transport IDs the group routes to. |
| `Level` | `LogLevel` | (no filter) | Minimum log level for this group; entries below it are dropped for the group's transports. |
| `Disabled` | `bool` | `false` | When `true`, this group's routing is suppressed (entries tagged only with disabled groups are dropped). |

### `Config` group fields

| Field | Type | Default | Description |
|---|---|---|---|
| `Groups` | `map[string]LogGroup` | nil | Named routing rules. When nil, every transport receives every entry (no routing). |
| `ActiveGroups` | `[]string` | nil | When non-empty, only the named groups are active. Nil/empty means "no filter — all defined groups active." |
| `UngroupedRouting` | `UngroupedRouting` | `{Mode: UngroupedToAll}` | Routes for entries with no group tag. |

## Routing Precedence

For each (entry, transport) pair the framework walks these checks top-to-bottom; the first one that fails drops the entry from that transport.

| # | Check | Drops the entry when |
|---|---|---|
| 1 | **`Groups` configured?** | If no `Groups` at all, the transport receives everything: rules below are skipped. |
| 2 | **Transport `IsEnabled()`** | Transport is disabled. |
| 3 | **Group `Disabled` flag** | The tagged group is disabled. |
| 4 | **`ActiveGroups` filter** | A non-empty filter is set and the group isn't in it. |
| 5 | **Group `Level`** | Entry's level is below the group's minimum. |
| 6 | **Transport membership** | The transport's ID isn't in the group's `Transports`. |
| 7 | **Ungrouped fall-back** | Tags reference only undefined groups, route via `UngroupedRouting`. Tags referencing a defined-but-blocked group do *not* fall back: they drop. |
| 8 | **Plugin `ShouldSend`** | Any plugin returns false for this (entry, transport) pair. |

Each subsequent section in this page configures one of the rules above. Read this table first; then the rest is detail.

## Per-Log Tagging

Tag a single entry with `WithGroup` on the builder chain:

```go
log.WithGroup("database").Error("connection lost")

// Combine with metadata and errors:
log.WithMetadata(loglayer.Metadata{"query": "SELECT *"}).WithGroup("database").Error("query failed")
log.WithError(err).WithGroup("database").Error("connection lost")

// Multiple groups: the entry routes to the union of both groups' transports.
log.WithGroup("database", "auth").Error("auth DB connection failed")
```

## Persistent Tagging (Child Loggers)

`WithGroup` on `*LogLayer` returns a child where every log is tagged:

```go
dbLog := log.WithGroup("database")
dbLog.Error("pool exhausted")  // routed via 'database'
dbLog.Info("connected")        // also routed via 'database'

// Pass to a library that accepts a logger:
db := newDBClient(log.WithGroup("database"))
```

Tags are additive across chained calls (deduplicated):

```go
authDB := log.WithGroup("auth").WithGroup("database")
authDB.Error("auth DB failure")  // routes to both 'auth' and 'database' transports
```

The parent logger is unchanged.

## Worked Example: Multi-Service Routing

A realistic setup makes the routing rules click. Suppose your service has three concerns and three places log entries should land:

- **`pretty`** (terminal): everything in development.
- **`structured-file`** (JSON to disk): everything in production for the local fluentd agent to ship.
- **`datadog`** (network): only `error` and `fatal` from the `billing` and `auth` paths, because Datadog log volume is metered.

Configure groups to express that policy:

```go
log := loglayer.New(loglayer.Config{
    Transports: []loglayer.Transport{
        pretty.New(pretty.Config{BaseConfig: transport.BaseConfig{ID: "pretty"}}),
        structured.New(structured.Config{BaseConfig: transport.BaseConfig{ID: "structured-file"}, Writer: file}),
        datadogtransport.New(datadogtransport.Config{
            BaseConfig: transport.BaseConfig{ID: "datadog"},
            APIKey:     os.Getenv("DD_API_KEY"),
        }),
    },
    Groups: map[string]loglayer.LogGroup{
        // "billing" entries go to all three transports, but Datadog
        // only receives error+fatal because of the per-group level.
        "billing": {
            Transports: []string{"pretty", "structured-file", "datadog"},
            Level:      loglayer.LogLevelError,
        },
        "auth": {
            Transports: []string{"pretty", "structured-file", "datadog"},
            Level:      loglayer.LogLevelError,
        },
        // Database concern: noisy, never ship to Datadog.
        "database": {
            Transports: []string{"pretty", "structured-file"},
        },
    },
    // Untagged entries (the dispatcher's own logs, ad-hoc Info calls in
    // main.go) go to pretty + structured-file but NOT Datadog.
    UngroupedRouting: loglayer.UngroupedRouting{
        Mode:       loglayer.UngroupedToTransports,
        Transports: []string{"pretty", "structured-file"},
    },
})

billingLog := log.WithGroup("billing")
authLog    := log.WithGroup("auth")
dbLog      := log.WithGroup("database")
```

What happens at emission time, for the configuration above:

| Call                                           | Routes to                                       | Why                                                                  |
|------------------------------------------------|-------------------------------------------------|----------------------------------------------------------------------|
| `billingLog.Error("invoice rejected")`         | `pretty`, `structured-file`, `datadog`          | `billing` group lists all three; `Error` passes the per-group level. |
| `billingLog.Info("invoice created")`           | dropped from all three                          | `billing` group's `Level: Error` filters the whole group, not just Datadog. |
| `dbLog.Debug("acquired conn")`                 | `pretty`, `structured-file`                     | `database` group lists those two; no Datadog. No per-group level set, so `Debug` passes. |
| `log.Info("server started")` (no `WithGroup`)  | `pretty`, `structured-file`                     | Untagged, so `UngroupedRouting` decides; Datadog excluded.           |
| `authLog.WithGroup("billing").Error("...")`    | `pretty`, `structured-file`, `datadog`          | Two groups; route to the union of their transports.                  |

::: warning Per-group level is per-*group*, not per-*transport*
Notice the `billingLog.Info(...)` row: the `Level: Error` on the `billing` group **drops the entry from every transport in that group**, including `pretty`. If you want `Info` to land on pretty/structured but only `Error+` on Datadog, the group level isn't the right tool. Split the work across two groups:
:::

```go
Groups: map[string]loglayer.LogGroup{
    "billing-all":    {Transports: []string{"pretty", "structured-file"}},
    "billing-remote": {Transports: []string{"datadog"}, Level: loglayer.LogLevelError},
}

billingLog := log.WithGroup("billing-all", "billing-remote")
billingLog.Info(...)   // → pretty, structured-file (billing-remote drops below Error)
billingLog.Error(...)  // → all three
```

Now the per-transport filtering is explicit, and a glance at the group definitions tells you where each level goes.

## Group Level Filtering

Each group has its own minimum log level; entries below it are dropped for that group's transports:

```go
log := loglayer.New(loglayer.Config{
    Transports: []loglayer.Transport{...},
    Groups: map[string]loglayer.LogGroup{
        "database": {Transports: []string{"datadog"}, Level: loglayer.LogLevelError},
    },
})

log.WithGroup("database").Info("query took 50ms")     // dropped (below error)
log.WithGroup("database").Error("connection lost")    // sent to datadog
```

This is independent of the logger's overall `SetLevel`. The logger-level filter runs first; if it passes, group-level filters apply per group.

## Ungrouped Routing

`UngroupedRouting` controls what happens to entries with **no** group tag (or whose tags are all undefined):

```go
// Default: all transports receive ungrouped entries (backward compatible).
loglayer.UngroupedRouting{Mode: loglayer.UngroupedToAll}

// Drop ungrouped entries entirely.
loglayer.UngroupedRouting{Mode: loglayer.UngroupedToNone}

// Route ungrouped entries only to specific transports.
loglayer.UngroupedRouting{
    Mode:       loglayer.UngroupedToTransports,
    Transports: []string{"console"},
}
```

::: tip
The default `UngroupedToAll` ensures full backward compatibility: adding `Groups` to an existing logger doesn't change anything for un-tagged log calls.
:::

## Active Groups Filter

`ActiveGroups` restricts routing to only the named groups. Entries tagged with other groups are dropped (unless every tag is undefined, in which case ungrouped rules apply):

```go
log := loglayer.New(loglayer.Config{
    Transports: []loglayer.Transport{...},
    Groups: map[string]loglayer.LogGroup{
        "database": {Transports: []string{"datadog"}},
        "auth":     {Transports: []string{"sentry"}},
        "payments": {Transports: []string{"datadog"}},
    },
    ActiveGroups: []string{"database"},  // only 'database' is active
})
```

### Driving from an environment variable

We don't read environment variables on your behalf, but `ActiveGroupsFromEnv` parses the standard comma-separated form for you to feed into `Config.ActiveGroups`:

```go
loglayer.New(loglayer.Config{
    Transports:   ...,
    Groups:       ...,
    ActiveGroups: loglayer.ActiveGroupsFromEnv("LOGLAYER_GROUPS"),
})
```

```sh
LOGLAYER_GROUPS=database,auth go run .
```

Useful for narrowing focus to a specific subsystem during debugging without code changes.

## Runtime Management

```go
// Add (or replace, by name) a group at runtime
log.AddGroup("inbox", loglayer.LogGroup{
    Transports: []string{"datadog"},
    Level:      loglayer.LogLevelError,
})

// Remove a group; returns true if it existed
log.RemoveGroup("inbox")

// Enable / disable a group without removing it
log.DisableGroup("database")
log.EnableGroup("database")

// Change a group's level
log.SetGroupLevel("database", loglayer.LogLevelDebug)

// Change the active-groups filter
log.SetActiveGroups("database", "auth")
log.ClearActiveGroups()                  // remove the filter

// Inspect current state
groups := log.GetGroups()                 // shallow copy
```

All runtime mutators are safe to call from any goroutine (atomic publish, mutex-serialized), matching the existing transport- and plugin-mutator contract.

## `Disabled` vs Undefined

Two cases that look similar but behave differently:

- **Disabled group**: `LogGroup.Disabled = true` is "explicitly off." Entries tagged only with disabled groups drop. They do **not** fall back to `UngroupedRouting`. Use it to silence a subsystem without removing the group config.
- **Undefined group**: a group name that isn't in `Config.Groups` (typo, or registered later). Entries tagged only with undefined groups fall back to `UngroupedRouting`, treated as if they had no tags.

This mirrors the TypeScript loglayer's behavior. The pragmatic effect: typos in code become harmless (graceful fall-back); explicit operator action stays load-bearing.

## Combining with `Raw`

`Raw` accepts a `Groups []string` that overrides the logger's assigned groups for that entry:

```go
log.Raw(loglayer.RawLogEntry{
    LogLevel: loglayer.LogLevelInfo,
    Messages: []any{"forwarded entry"},
    Groups:   []string{"forwarded"},
})
```
