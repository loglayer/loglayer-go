---
"go.loglayer.dev": minor
"transports/cli": minor
---

DX improvements from hmn-cli migration feedback.

`loglayer`:

- **`Config.Level` initial threshold**: set the minimum level at construction via `Config.Level`, applied exactly like `SetLevel`. Zero means "no override": every level stays enabled (the previous behavior). Composes with `Disabled`. See [Level](/configuration#level).
- **`WithStdlibContext` alias**: `WithContext` is now also reachable as `WithStdlibContext` on both `*LogLayer` and `*LogBuilder`, for discoverability when searching for "context". `WithContext` remains canonical. See [Go Context](/logging-api/go-context).

`transports/cli`:

- **Per-stream TTY detection in `ColorAuto`**: info / debug / trace lines follow stdout's TTY status; warn / error / fatal / panic lines follow stderr's. Piping stdout (e.g. `cli ... | less`) no longer strips color from severity lines that are still attached to a terminal. Resolution is pinned at construction. See [CLI Transport](/transports/cli#color-auto-always-never).
- **`Config.MessageFn` full-line takeover**: a callback that replaces the message plus the logfmt / table body with a single user-controlled string. The level prefix, its color, and the user prefix still apply; an empty return falls back to normal rendering. See [MessageFn](/transports/cli#messagefn).
