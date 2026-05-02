---
title: Migrating to v2
description: "Upgrade guide for loglayer-go v2: import paths bump to /v2, the prefix is now exposed on TransportParams.Prefix instead of being folded into Messages[0]."
---

# Migrating to v2

`loglayer-go` v2 ships one breaking change: **the loglayer core no longer mutates `Messages[0]` to fold the `WithPrefix` value into the message text.** The prefix now flows through `TransportParams.Prefix` and each transport decides how to render it.

This page is the upgrade checklist.

## Do I have to migrate?

Not immediately. v1.x continues to work; the v1 module path (`go.loglayer.dev`) keeps resolving to its last v1 tag and the auto-prepend behavior stays intact there. Future feature work and bug fixes ship at v2 (`go.loglayer.dev/v2`), so the migration is the path forward but it's not on a deadline.

You can migrate one module at a time: a project that uses several `loglayer-go` sub-modules can have v1 imports for some and v2 for others (Go treats `go.loglayer.dev` and `go.loglayer.dev/v2` as separate modules). The catch is that fields shared between modules (e.g. `loglayer.Config` from main) won't bridge between v1 and v2 — pick one main module per project.

## Why this change

`v1.x` folded the prefix into `Messages[0]` from the core so transports that didn't know about prefixes got the right behavior for free. The downside: transports that DID want to render the prefix differently (separate color, separate JSON field, structured forwarding to underlying loggers) couldn't, because by the time they saw the message it was already mangled. Pulling the prefix into a first-class field unblocks every smarter rendering, at the cost of a one-time import-path migration.

## Step 1: bump every import path to `/v2`

The main module and every sub-module are now versioned at `v2`. Update your `go.mod` requires and your source-file imports.

```sh
go get go.loglayer.dev/v2 \
       go.loglayer.dev/transports/cli/v2 \
       go.loglayer.dev/transports/zerolog/v2 \
       go.loglayer.dev/plugins/redact/v2
       # ... whichever sub-modules you import
```

In source files:

```diff
 import (
-    "go.loglayer.dev"
-    "go.loglayer.dev/transports/zerolog"
-    "go.loglayer.dev/plugins/redact"
+    "go.loglayer.dev/v2"
+    "go.loglayer.dev/transports/zerolog/v2"
+    "go.loglayer.dev/plugins/redact/v2"
 )
```

The package import name (`loglayer`, `zerolog`, `redact`) does not change; only the import path does.

## Step 2: most users are done

For users of the built-in transports who don't write custom transports, nothing else changes. Every built-in transport preserves the v1 user-visible output: `log.WithPrefix("[auth]").Info("hi")` still produces `"[auth] hi"` through every renderer / wrapper / network transport, just like it did in v1.

The exceptions to "nothing else changes":

- The **cli transport** opts into smart prefix rendering: the user prefix renders in dim grey while the level prefix and message body keep the level color. If you were using cli with `WithPrefix`, the rendered output is now visually layered. See the [cli transport doc](/transports/cli) for an example.
- The **blank transport** intentionally passes raw v2 params through to your `ShipToLogger` function. The prefix is on `params.Prefix`, not in `Messages[0]`; if you were extracting the prefix from `Messages[0]`, switch to reading `params.Prefix`.

## Step 3: custom transports

If you wrote a custom transport that reads `params.Messages[0]` and relied on the prefix being baked in, you have two paths:

### Path A: preserve v1 behavior (simplest)

Call `transport.JoinPrefixAndMessages` at the top of `SendToLogger`:

```go
import "go.loglayer.dev/v2/transport"

func (t *Transport) SendToLogger(p loglayer.TransportParams) {
    if !t.ShouldProcess(p.LogLevel) {
        return
    }
    p.Messages = transport.JoinPrefixAndMessages(p.Prefix, p.Messages)
    // ... your existing rendering logic, unchanged
}
```

The helper has fast-path early returns when the prefix is empty, when messages is empty, or when `messages[0]` isn't a string. Per-call cost on a no-prefix logger is one string compare.

### Path B: smart rendering

Read `params.Prefix` directly and render it however suits your transport:

- A renderer transport could color the prefix differently from the message body (see `transports/cli` for an example).
- A structured / JSON transport could emit the prefix as a separate top-level field instead of embedding it in `msg`.
- A wrapper transport could forward the prefix to the underlying logger's structured-field API (`zerolog.Event.Str("prefix", p.Prefix)`, `zap.Field`, etc.).

## Step 4: custom plugins

The dispatch-time plugin hook param structs (`BeforeDataOutParams`, `BeforeMessageOutParams`, `TransformLogLevelParams`, `ShouldSendParams`) gained a `Prefix string` field in v1.7.0; that part is unchanged in v2. The only difference: in v1, `params.Messages[0]` carried the prefix folded in; in v2 it doesn't. Plugins that read the message string directly should be aware.

The prefix is read-only from the plugin's perspective; hooks that return modified data / messages / level / send-decision can act on the prefix value but don't propagate a modified prefix back to downstream hooks.

## See also

- The full [release notes for v2](/whats-new) cover every package's bump and any other v2-only changes.
- [`creating-transports.md`](/transports/creating-transports#reading-params-prefix) documents the `params.Prefix` contract for transport authors.
- [`creating-plugins.md`](/plugins/creating-plugins#reading-params-prefix) documents it for plugin authors.
