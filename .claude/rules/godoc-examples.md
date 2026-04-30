# godoc Examples and `doc.go`

Rules for godoc `Example*` functions (rendered on pkg.go.dev) and the package-level `doc.go` file. Distinct from the VitePress docs site, which `documentation.md` owns.

## Coverage

- **Every transport sub-module** has at least one `ExampleNew` (or `Example<Constructor>` if there is no `New`) showing wiring.
- **Every plugin sub-module** has at least one Example for its most common constructor.
- **Main module** has:
  - A package-level `Example` that exercises the fluent chain (`WithFields` + `WithMetadata` + level method).
  - One Example per non-trivial flow: plugin hook authoring (one per distinct hook signature, since `Fields`/`Metadata`/`Data`/`Message`/`Level`/`SendGate` differ enough to confuse readers), `Lazy`, `FromContext` middleware, `WithGroup` routing, `AddTransport` runtime mutation, `ErrorSerializer` customization.
- **Skip:** per-level methods (`ExampleLogLayer_Info` etc.) — the package-level `Example` already shows `.Info()`. Skip mechanical mutation siblings (`RemoveTransport` once `AddTransport` exists). Skip niche reference surface (`Source`, `Schema`, `ParseLogLevel`, `ActiveGroupsFromEnv`) — type-level GoDoc plus the docs site is enough.

## File shape

- Filename: `example_test.go` in the package directory.
- Package: external test package (`<name>_test`), so the Example imports the package by its public path.
- Each Example is one function; ~15-25 lines including imports.
- Sub-modules cannot import `internal/lltest` (it's package-private to the main module). Use `transports/testing` for in-memory capture in plugin Examples.

## Determinism strategy by transport class

| Class | Examples | `// Output:` |
|-------|----------|--------------|
| Pure transports with a deterministic-time hook (`DateFn`, `TimestampFn`) | `console`, `pretty`, `structured`, `lumberjack` | yes — set the hook to a fixed string |
| In-memory capture | `testing`, `blank` | yes — read back from the captured library or callback |
| Wrapper transports (underlying logger owns format/timestamp) | `zerolog`, `zap`, `slog`, `logrus`, `charmlog`, `phuslu` | no — set `Writer: io.Discard` and let the Example be construct-only |
| Network transports | `datadog`, `http`, `central`, `sentry`, `otellog` | no — construct only, `defer t.Close()` if the transport spawns a worker |

`// Output:` is what makes an Example genuinely useful (the test runner verifies it). Use it whenever feasible. Construction-only Examples still render on pkg.go.dev and still compile-check, just don't run.

## Self-containment

pkg.go.dev shows the Example function body, NOT file-scope helpers. Two consequences:

- **First-encounter Examples** (the package-level `Example`, every sub-module's `ExampleNew`) call `loglayer.New(loglayer.Config{...})` directly. Don't hide construction behind a helper; readers landing on the page need to see the full call shape.
- **Method-focused Examples** (e.g. `ExampleLogLayer_WithFields` showing what `WithFields` does on an already-constructed logger) may use a helper like `exampleLogger()` since construction isn't the point.

If an Example needs a transport whose output differs from the file's main helper (e.g. `ExampleLogLayer_AddTransport` needing per-transport ID prefixes), define a second small transport type in the same file (`tagTransport`). It won't render on pkg.go.dev, but the Example body is still readable because the type's name signals what it does.

## Comment hygiene

Each Example function gets a one-or-two-sentence comment leading with the conclusion. Skip:

- "Output goes to io.Discard so the godoc example doesn't pollute test stdout." Test plumbing isn't reader-relevant.
- "Production code leaves it unset." Inferred.
- Lists of generic use cases ("Use it for redaction, normalization, or augmentation"). Either the reader knows or doesn't; the constructor's GoDoc owns this.
- Sentences that restate the function signature.

Include:

- Caveats specific to the package (phuslu calling `os.Exit` on Fatal, Sentry needing `sentry.Init`).
- Behavior the Example asserts that isn't obvious from the body (e.g. "transports dispatch in registration order" for `AddTransport`).

## `// Output:` type-name hygiene

Don't expose stdlib-internal type names like `*errors.errorString`. Define a small package-owned error type in the test file so the rendered name is stable and example-owned:

```go
type queryErr struct{ msg string }

func (e *queryErr) Error() string { return e.msg }
```

The `// Output:` line then shows `*<package>_test.queryErr`, which is owned by this codebase.

## Mutation hygiene in plugin Examples

Plugin hooks that receive caller-owned data (`Fields`, `Metadata`, message slice) must return a copy in Examples, even when the hook's API allows in-place mutation. Examples are copy-paste material. A reader who copies a mutating hook into a real codebase risks data races against another goroutine still holding the same map reference.

```go
// ✅ models safe authoring
addEnv := loglayer.NewMetadataHook("add-env", func(in any) any {
    md, ok := in.(loglayer.Metadata)
    if !ok {
        return in
    }
    out := make(loglayer.Metadata, len(md)+1)
    maps.Copy(out, md)
    out["env"] = "prod"
    return out
})

// ❌ models unsafe authoring (even if technically allowed by the hook contract)
addEnv := loglayer.NewMetadataHook("add-env", func(in any) any {
    if md, ok := in.(loglayer.Metadata); ok {
        md["env"] = "prod"
        return md
    }
    return in
})
```

## `doc.go` for package overview

The main module's package-level GoDoc lives alone in `doc.go` (a file with only `package <name>` and the comment above it). Don't co-locate the package comment in a code-bearing file.

Required sections for the main module:

1. One-line synopsis.
2. Link to the docs site (`https://go.loglayer.dev`). pkg.go.dev autolinks URLs in package comments.
3. **Quickstart** code block showing the canonical fluent-chain wiring.
4. A mental-model section that names the distinct concepts a caller has to keep separate. For loglayer: **Three data shapes** (`Fields`, `Metadata`, `Context`), each with lifetime and use case.
5. **Choosing a constructor** when there's a `New` / `Build` / `NewMock` pair (panics vs returns error vs silent).
6. **Concurrency** statement summarizing what's safe to call from any goroutine, classified by method group (read-only / returns-new / atomic-mutate / setup-only). The detailed contract lives in `AGENTS.md`; the package doc gives the one-paragraph summary.
7. **Authoring** pointer for users implementing transports or plugins, linking to the docs site's creator pages.

Sub-module `doc.go` files are typically not needed; the per-package `// Package <name> ...` comment at the top of the main `.go` file is enough. Add `doc.go` only when the overview wants its own file (separate from the implementation).

## When you add a new transport, plugin, or hook

In addition to the docs-site updates listed in `documentation.md`:

1. Add `example_test.go` in the new sub-module with at least one Example (per the "Coverage" and "Determinism" rules above).
2. If the new feature introduces a non-trivial flow on the main module's surface (a new constructor, a new hook signature, a new runtime mutator), add a corresponding Example to the main module's `example_test.go`.
3. Run `go test -run='^Example' .` in the affected module(s) to verify any `// Output:` directives.
