---
title: Redact Plugin
description: "Replace values for sensitive keys, value patterns, or struct fields before they reach a transport."
---

# Redact Plugin

<ModuleBadges path="plugins/redact" />

`plugins/redact` replaces sensitive values in metadata and persistent fields before any transport sees them. Useful for keeping secrets, PII, and credentials out of log output without rewriting every call site.

```sh
go get go.loglayer.dev/plugins/redact
```

Dependency-free. Pure Go (only `regexp` from stdlib; the walker uses reflection from the standard library).

## Basic Usage

```go
import (
    "go.loglayer.dev"
    "go.loglayer.dev/plugins/redact"
    "go.loglayer.dev/transports/structured"
)

log := loglayer.New(loglayer.Config{
    Transport:         structured.New(structured.Config{}),
    MetadataFieldName: "metadata",
})

log.AddPlugin(redact.New(redact.Config{
    Keys: []string{"password", "apiKey", "ssn"},
}))

log.WithMetadata(loglayer.Metadata{
    "userId":   42,
    "password": "hunter2",
}).Info("login")
```

```json
{"level":"info","time":"...","msg":"login","metadata":{"password":"[REDACTED]","userId":42}}
```

## Type Preservation

Whatever runtime type you pass in comes back out, with sensitive values replaced:

- `map[string]any` → `map[string]any`
- `MyStruct` → `MyStruct` (with redacted fields)
- `*MyStruct` → `*MyStruct`
- `[]MyStruct` → `[]MyStruct`

Transports that type-switch on `params.Metadata` continue to see the original type.

## Config

```go
type Config struct {
    ID       string            // plugin ID; default "redact"
    Keys     []string          // key names to redact
    Patterns []*regexp.Regexp  // string-value patterns to redact
    Censor   any               // replacement value; default "[REDACTED]"
}
```

### `Keys`

Match by key name. The match is exact and case-sensitive.

Walks at any depth in nested values: `map[string]any`, `[]any`, struct fields, pointers, typed maps, typed slices. So `Keys: []string{"password"}` will redact all of:

- `metadata["password"]`
- `metadata["user"]["password"]`
- `metadata["users"][i]["password"]`
- `myStruct.Password` (when the field has `json:"password"` or is named `Password` and that name is in your Keys)

For struct fields, `Keys` matches the **JSON tag name** if present, otherwise the Go field name. So a field declared as `Password string` matches `Keys: []string{"Password"}`, and a field declared as `Password string \`json:"password"\`` matches `Keys: []string{"password"}`.

### `Patterns`

Regular expressions matched against **string values** (not keys). A value matching any pattern is replaced with `Censor`. Useful for value-shaped data that may appear under arbitrary keys.

```go
ssn := regexp.MustCompile(`^\d{3}-\d{2}-\d{4}$`)
cc  := regexp.MustCompile(`^\d{13,19}$`)

redact.New(redact.Config{
    Patterns: []*regexp.Regexp{ssn, cc},
})
```

Patterns walk the same shapes as `Keys`. Anchor your patterns (`^...$`) for full-string matches; otherwise a pattern like `\d{16}` will redact any string containing 16 consecutive digits.

### `Censor`

The value substituted in place of the original. Defaults to the string `"[REDACTED]"`.

The censor is applied based on the destination field's type:

| Destination | Behavior |
|---|---|
| `string` field | Censor stringified via `fmt.Sprintf` if non-string. |
| `any` / `interface{}` field | Censor stored as-is. |
| Other typed field (`int`, `time.Time`, custom struct, ...) | Field set to its zero value. |

The "other typed field → zero value" rule means a `Count int` field with `Keys: ["count"]` becomes `0`, not `"[REDACTED]"`. We can't safely substitute a string into an int field, so we err on the side of clearing.

### `ID`

Defaults to `"redact"`. Override when you register multiple redactors at once (e.g., one for PII keys with a `[PII]` censor, one for secrets with `[REDACTED]`):

```go
log.AddPlugin(redact.New(redact.Config{
    ID:     "redact-pii",
    Keys:   []string{"email", "phone"},
    Censor: "[PII]",
}))

log.AddPlugin(redact.New(redact.Config{
    ID:     "redact-secrets",
    Keys:   []string{"password", "apiKey", "token"},
    Censor: "[REDACTED]",
}))
```

## Where it Fires

The plugin implements three hooks:

- **`OnMetadataCalled`**: rewrites metadata when `WithMetadata` or `MetadataOnly` is called.
- **`OnFieldsCalled`**: rewrites fields when `WithFields` is called.
- **`OnBeforeDataOut`**: re-walks the assembled `Data` map (fields + the error subtree LogLayer builds from `WithError`) right before dispatch.

The first two scrub data the caller passes in. `OnBeforeDataOut` exists so a `Patterns`-style redactor also catches secrets that only surface in `err.Error()` (LogLayer places `WithError` errors into `Data` as `{"err": {"message": err.Error()}}`). Without the third hook a credit-card-shaped string baked into an error message would slip past redaction.

```go
log.WithError(errors.New("auth failed for card 4111111111111111")).Error("oops")
// {"err":{"message":"[REDACTED]"}, ...}
```

## Caller Input Preservation

The plugin produces a deep clone; your original metadata, fields map, struct, slice, or pointer is **not mutated**. Safe to pass the same value into multiple log calls without surprise.

```go
shared := map[string]any{"password": "p"}
log.WithMetadata(shared).Info("call 1")
log.WithMetadata(shared).Info("call 2")

shared["password"]  // still "p", not "[REDACTED]"
```

## Limitations

- **Unexported struct fields are skipped.** They aren't reachable through the structured logging output anyway, but the plugin won't help if you're relying on a renderer that uses `fmt.Sprintf("%+v", v)` to print private state.
- **Custom `MarshalJSON` may bypass redaction.** If your struct's `MarshalJSON` returns a hand-crafted byte slice that doesn't reflect the (now-redacted) field values, the rendered output could leak. The plugin redacts the struct, but `json.Marshal` will call your custom method on the cloned struct.
- **Map keys aren't matched for non-string-keyed maps.** `map[int]string` with `Keys: ["1"]` won't redact key `1`. The walker still recurses into the values.
