---
"go.loglayer.dev": major
---

**Metadata now nests by default.** `Config.MetadataFieldName` resolves to `"metadata"` when empty, so map and struct metadata render uniformly under that key across every transport. Restore the v2 root-flattening shape with `Config.FlattenMetadata: true`. The core module path moves from `go.loglayer.dev/v2` to `go.loglayer.dev/v3`. See [Migrating to v3](/migrating-to-v3).
