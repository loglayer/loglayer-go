---
title: Migration Guide
description: "Upgrade guides for loglayer-go: v2 (import paths /v2, TransportParams.Prefix) and v3 (import paths /v3, MetadataFieldName default)."
---

# Migration Guide

LogLayer for Go has shipped two major versions. Each upgrade is a short checklist; pick the section that matches where you are.

- **Migrating to v3** (from v2): import paths bump to `/v3`; metadata nests under `"metadata"` by default.
- **Migrating to v2** (from v1): import paths bump to `/v2`; the prefix moves to `TransportParams.Prefix`.

## Migrating to v3

`loglayer-go` v3 ships two changes: **every core import path bumps to `/v3`**, and **metadata now nests under the `"metadata"` key by default** instead of flattening at the root. One config field, `FlattenMetadata: true`, restores the v2 shape.

This section covers the core (`go.loglayer.dev/v3`); the import-path notes also apply to the core's sub-packages (`/v3/transport`, `/v3/utils/...`).

::: warning Transports are still on v2 in this release
The transports keep their `v2` paths until the follow-up release that bumps them to `v3`. Until then, code that pairs the v3 core with a `v2` transport path does not compile together. Migrate the core first, then the transports when their v3 bumps land.
:::

### Do I have to migrate?

Not immediately. v2.x continues to work; the v2 module path (`go.loglayer.dev/v2`) keeps resolving to its last v2 tag and the v2 metadata placement stays intact there. Future feature work and bug fixes ship at v3 (`go.loglayer.dev/v3`), so the migration is the path forward but it's not on a deadline.

You can migrate one module at a time: a project that uses several `loglayer-go` sub-modules can have v2 imports for some and v3 for others (Go treats `go.loglayer.dev/v2` and `go.loglayer.dev/v3` as separate modules). The catch is that fields shared between modules (e.g. `loglayer.Config` from main) won't bridge between v2 and v3; pick one core version per project.

### What changed

- **Import paths bump to `/v3`** for the core and its sub-packages: `go.loglayer.dev/v2` → `go.loglayer.dev/v3`, `go.loglayer.dev/v2/transport` → `go.loglayer.dev/v3/transport`, and so on. The package import names (`loglayer`, `transport`) do not change.
- **Metadata nests under `"metadata"` by default.** When `Config.MetadataFieldName` is empty, the core resolves it to `"metadata"`, so both map and struct metadata render under that single key uniformly across every transport. This closes the asymmetric v2 gap where renderers flattened map metadata at the root while wrappers nested non-map values under a hardcoded key. It applies to every transport, including third-party ones, because the resolved key ships on `TransportParams.Schema`.
- **`transports/structured` stays on v2 for this release.** The structured transport moves to `/v3` in a follow-up release, along with sanitized output and empty-message handling. The default nesting flip above applies to it today through the schema key.

### The one-line opt-out

Set `FlattenMetadata: true` to restore the v2 shape: map metadata merges at the root of the output, and non-map metadata follows each transport's historical placement. The field is ignored when `MetadataFieldName` is explicitly set; an explicit key always wins.

```go
loglayer.New(loglayer.Config{
    Transport:       structured.New(structured.Config{}),
    FlattenMetadata: true,
})
```

### Step 1: bump every core import path to `/v3`

Update your `go.mod` require for the core and your source-file imports.

```sh
go get go.loglayer.dev/v3
```

In source files:

```diff
 import (
-    "go.loglayer.dev/v2"
-    "go.loglayer.dev/v2/transport"
+    "go.loglayer.dev/v3"
+    "go.loglayer.dev/v3/transport"
 )
```

Then run `go mod tidy`. Transport and plugin sub-modules keep their current paths until each ships its own v3 bump; check each page in the [Transports overview](/transports/) and [Plugins overview](/plugins/) for the current path.

### Step 2: decide on metadata placement

If nothing in your pipeline depends on map metadata living at the root of the JSON output, you're done after the import bump.

If you do depend on root flattening (a JSON pipeline that parses map metadata at the root of structured / console / pretty output, or an alert rule keyed on a root field), set `FlattenMetadata: true` on the config as a stopgap while you migrate the pipeline, then remove it when the pipeline reads the `"metadata"` key.

### Step 3: check custom transports and plugins

- **Custom transports** that read `params.Schema.MetadataFieldName` keep working unchanged: the value is now `"metadata"` instead of `""` unless `FlattenMetadata` is set. Transports that never read the key are unaffected by the default flip.
- **Custom plugins** that inspect `params.Data` to locate metadata should read `params.Schema.MetadataFieldName` rather than assuming a placement. See [Creating Transports → Handling `any` Metadata](/transports/creating-transports#handling-any-metadata) for the placement policies.

### Known incompatibilities

Any consumer of the emitted JSON that parsed map metadata at the root now finds it under `"metadata"`. Concretely:

- Log pipelines and alert rules keyed on root-level fields that were previously metadata.
- Tests asserting on `testing.LogLine` shapes that assumed root flattening.
- Wrapper transports and downstream dashboards that read metadata attributes positionally (the JSON key they appear under changes, not the values).

All of these are addressed by `FlattenMetadata: true` (v2 shape) or by updating the consumer to read the `"metadata"` key.

## Migrating to v2

`loglayer-go` v2 ships two breaking changes: **every import path bumps to `/v2`**, and **the loglayer core no longer mutates `Messages[0]` to fold the `WithPrefix` value into the message text.** The prefix now flows through `TransportParams.Prefix` and each transport decides how to render it.

### Do I have to migrate?

Not immediately. v1.x continues to work; the v1 module path (`go.loglayer.dev`) keeps resolving to its last v1 tag and the auto-prepend behavior stays intact there. Future feature work and bug fixes ship at v2 (`go.loglayer.dev/v2`), so the migration is the path forward but it's not on a deadline.

You can migrate one module at a time: a project that uses several `loglayer-go` sub-modules can have v1 imports for some and v2 for others (Go treats `go.loglayer.dev` and `go.loglayer.dev/v2` as separate modules). The catch is that fields shared between modules (e.g. `loglayer.Config` from main) won't bridge between v1 and v2; pick one main module per project.

### Why this change

`v1.x` folded the prefix into `Messages[0]` from the core so transports that didn't know about prefixes got the right behavior for free. The downside: transports that DID want to render the prefix differently (separate color, separate JSON field, structured forwarding to underlying loggers) couldn't, because by the time they saw the message it was already mangled. Pulling the prefix into a first-class field unblocks every smarter rendering, at the cost of a one-time import-path migration.

The new contract also keeps the core out of the business of mutating caller-owned input: in v1, the prefix-prepend silently rewrote the user's `Messages` slice before any transport saw it; in v2, the core passes the slice through untouched and exposes the prefix on its own field.

### Step 1: bump every import path to `/v2`

The main module and every sub-module are now versioned at `v2`. Update your `go.mod` requires and your source-file imports.

```sh
# Run for each sub-module you import
go get go.loglayer.dev/v2
go get go.loglayer.dev/transports/cli/v2
go get go.loglayer.dev/transports/zerolog/v2
go get go.loglayer.dev/plugins/redact/v2
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

### Step 2: most users are done

For users of the built-in transports who don't write custom transports, nothing else changes. Every built-in transport preserves the v1 user-visible output: `log.WithPrefix("[auth]").Info("hi")` still produces `"[auth] hi"` through every renderer / wrapper / network transport, just like it did in v1.

The exceptions to "nothing else changes":

- The **cli transport** opts into smart prefix rendering: the user prefix renders in dim grey while the level prefix and message body keep the level color. If you were using cli with `WithPrefix`, the rendered output is now visually layered. See the [cli transport doc](/transports/cli) for an example.
- The **blank transport** hands `params` straight to your `ShipToLogger` function. If your callback was reading the prefix out of `Messages[0]`, read `params.Prefix` instead.
- If you assert on `testing.LogLine` in tests, the unmangled prefix is also available on `LogLine.Prefix` (new field in v2). Existing assertions on `Messages[0]` keep working because the testing transport calls `JoinPrefixAndMessages` internally.

### Step 3: custom transports

If you wrote a custom transport that reads `params.Messages[0]` and relied on the prefix being baked in, you have two paths:

#### Path A: preserve v1 behavior (simplest)

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

#### Path B: smart rendering

Read `params.Prefix` directly and render it however suits your transport:

- A renderer transport could color the prefix differently from the message body (see `transports/cli` for an example).
- A structured / JSON transport could emit the prefix as a separate top-level field instead of embedding it in `msg`.
- A wrapper transport could forward the prefix to the underlying logger's structured-field API (`zerolog.Event.Str("prefix", p.Prefix)`, `zap.Field`, etc.).

### Step 4: custom plugins

The dispatch-time plugin hook param structs (`BeforeDataOutParams`, `BeforeMessageOutParams`, `TransformLogLevelParams`, `ShouldSendParams`) gained a `Prefix string` field in v1.7.0; that part is unchanged in v2. The only difference: in v1, `params.Messages[0]` carried the prefix folded in; in v2 it doesn't. Plugins that read the message string directly should be aware.

The prefix is read-only from the plugin's perspective; hooks that return modified data / messages / level / send-decision can act on the prefix value but don't propagate a modified prefix back to downstream hooks.

## References

- [`MetadataFieldName`](/configuration#metadatafieldname) and [`FlattenMetadata`](/configuration#flattenmetadata) in the configuration reference.
- The [Metadata page](/logging-api/metadata) for the v3 nesting rule.
- [`creating-transports.md`](/transports/creating-transports#reading-params-prefix) documents the `params.Prefix` contract for transport authors.
- [`creating-plugins.md`](/plugins/creating-plugins#reading-params-prefix) documents it for plugin authors.
- The full [release notes](/whats-new) cover every package's bump and any other version-specific changes.
