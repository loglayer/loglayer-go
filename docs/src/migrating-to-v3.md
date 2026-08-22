---
title: Migrating to v3
description: "Upgrade guide for loglayer-go v3: import paths bump to /v3, metadata now nests under \"metadata\" by default, FlattenMetadata restores the v2 shape."
---

# Migrating to v3

`loglayer-go` v3 ships two changes: **every core import path bumps to `/v3`**, and **metadata now nests under the `"metadata"` key by default** instead of flattening at the root. One config field, `FlattenMetadata: true`, restores the v2 shape.

This page is the upgrade checklist. It covers the core (`go.loglayer.dev/v3`); the import-path notes also apply to the core's sub-packages (`/v3/transport`, `/v3/utils/...`).

## Do I have to migrate?

Not immediately. v2.x continues to work; the v2 module path (`go.loglayer.dev/v2`) keeps resolving to its last v2 tag and the v2 metadata placement stays intact there. Future feature work and bug fixes ship at v3 (`go.loglayer.dev/v3`), so the migration is the path forward but it's not on a deadline.

You can migrate one module at a time: a project that uses several `loglayer-go` sub-modules can have v2 imports for some and v3 for others (Go treats `go.loglayer.dev/v2` and `go.loglayer.dev/v3` as separate modules). The catch is that fields shared between modules (e.g. `loglayer.Config` from main) won't bridge between v2 and v3; pick one core version per project.

## What changed

- **Import paths bump to `/v3`** for the core and its sub-packages: `go.loglayer.dev/v2` → `go.loglayer.dev/v3`, `go.loglayer.dev/v2/transport` → `go.loglayer.dev/v3/transport`, and so on. The package import names (`loglayer`, `transport`) do not change.
- **Metadata nests under `"metadata"` by default.** When `Config.MetadataFieldName` is empty, the core resolves it to `"metadata"`, so both map and struct metadata render under that single key uniformly across every transport. This closes the asymmetric v2 gap where renderers flattened map metadata at the root while wrappers nested non-map values under a hardcoded key. It applies to every transport, including third-party ones, because the resolved key ships on `TransportParams.Schema`.
- **`transports/structured` stays on v2 for this release.** The structured transport moves to `/v3` in a follow-up release, along with sanitized output and empty-message handling. The default nesting flip above applies to it today through the schema key.

## The one-line opt-out

Set `FlattenMetadata: true` to restore the v2 shape: map metadata merges at the root of the output, and non-map metadata follows each transport's historical placement. The field is ignored when `MetadataFieldName` is explicitly set; an explicit key always wins.

```go
loglayer.New(loglayer.Config{
    Transport:       structured.New(structured.Config{}),
    FlattenMetadata: true,
})
```

## Step 1: bump every core import path to `/v3`

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

## Step 2: decide on metadata placement

If nothing in your pipeline depends on map metadata living at the root of the JSON output, you're done after the import bump.

If you do depend on root flattening (a JSON pipeline that parses map metadata at the root of structured / console / pretty output, or an alert rule keyed on a root field), set `FlattenMetadata: true` on the config as a stopgap while you migrate the pipeline, then remove it when the pipeline reads the `"metadata"` key.

## Step 3: check custom transports and plugins

- **Custom transports** that read `params.Schema.MetadataFieldName` keep working unchanged: the value is now `"metadata"` instead of `""` unless `FlattenMetadata` is set. Transports that never read the key are unaffected by the default flip.
- **Custom plugins** that inspect `params.Data` to locate metadata should read `params.Schema.MetadataFieldName` rather than assuming a placement. See [Creating Transports → Handling `any` Metadata](/transports/creating-transports#handling-any-metadata) for the placement policies.

## Known incompatibilities

Any consumer of the emitted JSON that parsed map metadata at the root now finds it under `"metadata"`. Concretely:

- Log pipelines and alert rules keyed on root-level fields that were previously metadata.
- Tests asserting on `testing.LogLine` shapes that assumed root flattening.
- Wrapper transports and downstream dashboards that read metadata attributes positionally (the JSON key they appear under changes, not the values).

All of these are addressed by `FlattenMetadata: true` (v2 shape) or by updating the consumer to read the `"metadata"` key.

## See also

- [`MetadataFieldName`](/configuration#metadatafieldname) and [`FlattenMetadata`](/configuration#flattenmetadata) in the configuration reference.
- The [Metadata page](/logging-api/metadata) for the v3 nesting rule.
- The full [release notes](/whats-new) cover every package's bump and any other v3 changes.
