| Type | Method | Scope | Purpose |
|------|--------|-------|---------|
| **Fields** | `WithFields()` | Persistent across all logs from the derived logger | Request IDs, user info, session data |
| **Metadata** | `WithMetadata()` | Single log entry only | Event-specific details, durations, counts |
| **Errors** | `WithError()` | Single log entry only | An `error` value, serialized for output |

Per-log metadata can never accidentally leak into future logs, errors are serialized consistently, and each type can be nested under a dedicated field via [configuration](/configuration). `WithFields` is keyed (`map[string]any`) because fields support keyed operations like merge, clear-by-key, and copy on `Child()`. `WithMetadata` accepts `any` because each entry is a one-shot payload — pass a struct, a map, or a scalar.
