// Package sloghandler exposes a [log/slog.Handler] backed by a
// [*loglayer.LogLayer], so any code emitting through slog (your own
// `slog.Info(...)` calls, or library code that accepts a `*slog.Logger`)
// flows through loglayer's plugin pipeline, fan-out, and groups.
//
// Mental model: slog has two halves. `*slog.Logger` is the frontend API
// (`slog.Info("msg", "k", "v")`); `slog.Handler` is the sink that actually
// emits the record. This package implements the sink and routes every
// record into a loglayer logger, so the rest of loglayer's setup (redact
// plugin, oteltrace, multi-transport fan-out, level filters) applies.
//
// Direction matters: `transports/slog` is the opposite. It lets loglayer
// emit through a slog logger as a backend. Use this package when slog is
// the frontend you're calling and loglayer is what you want behind it.
//
// Usage:
//
//	log := loglayer.New(loglayer.Config{Transport: structured.New(structured.Config{})})
//	log.AddPlugin(redact.New(redact.Config{Keys: []string{"password"}}))
//
//	slog.SetDefault(slog.New(sloghandler.New(log)))
//	slog.Info("user signed in", "userId", 42, "password", "hunter2")
//	// → flows through loglayer; redact plugin replaces the password.
//
// Mapping summary:
//
//   - slog levels Debug/Info/Warn/Error map onto the matching loglayer
//     levels. slog has no Fatal; values at or above slog.LevelError map
//     to LogLevelError so a raw slog level can never trigger
//     loglayer's Fatal exit. (You can still call log.Fatal directly.)
//   - Attrs added via `slog.With(...)` / `Handler.WithAttrs` accumulate
//     on the handler and become persistent fields on every emission
//     handled by it.
//   - Attrs added inline on a record (`slog.Info("msg", "k", "v")`)
//     become per-call fields for that emission.
//   - `Handler.WithGroup(name)` opens a nested map under name; subsequent
//     attrs land under it. Empty groups (no attrs added under them) are
//     dropped on emission, matching slog's documented behavior.
//   - The context passed to Handle is forwarded via Raw so dispatch-time
//     plugins (oteltrace, datadogtrace) see the request's context.
package sloghandler

import (
	"context"
	"log/slog"
	"slices"

	"go.loglayer.dev"
)

// Handler is a slog.Handler that emits records through a loglayer logger.
// Construct with [New]. Safe for concurrent use; WithAttrs and WithGroup
// return new Handler values without mutating the receiver.
type Handler struct {
	log *loglayer.LogLayer

	// groups is the slog group prefix path active at this handler's
	// position in the WithGroup chain. New attrs land under
	// fields[groups[0]][groups[1]]...
	groups []string

	// attrs is the accumulated WithAttrs payload, each annotated with the
	// group path that was active when it was added. Materialized into
	// nested fields at Handle time so the captured group context is
	// preserved across later WithGroup / WithAttrs calls.
	attrs []groupedAttrs
}

// groupedAttrs binds a batch of slog.Attrs to the group path that was
// active at the time WithAttrs was called.
type groupedAttrs struct {
	groups []string
	attrs  []slog.Attr
}

// New constructs an slog.Handler that emits through log.
//
// The returned handler's WithAttrs / WithGroup return derived handlers
// that share the same underlying loglayer logger.
func New(log *loglayer.LogLayer) *Handler {
	return &Handler{log: log}
}

// Enabled reports whether log calls at level should be emitted, by
// translating to the matching loglayer level and consulting the
// underlying logger's level state.
func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
	return h.log.IsLevelEnabled(slogToLoglayerLevel(level))
}

// Handle converts the record into a loglayer Raw entry and dispatches.
// Per slog.Handler contract, it should not retain r.
func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	level := slogToLoglayerLevel(r.Level)
	if !h.log.IsLevelEnabled(level) {
		return nil
	}

	// Fast path: no attrs from the slog side, so let Raw use the underlying
	// logger's existing fields, no map allocation here.
	hasAttrs := len(h.attrs) > 0 || r.NumAttrs() > 0
	var fields loglayer.Fields
	if hasAttrs {
		fields = mergeExisting(h.log.GetFields())
		for _, ga := range h.attrs {
			applyAttrsAtPath(fields, ga.groups, ga.attrs)
		}
		if r.NumAttrs() > 0 {
			target := mapAtPath(fields, h.groups)
			r.Attrs(func(a slog.Attr) bool {
				applyAttr(target, a)
				return true
			})
		}
	}

	h.log.Raw(loglayer.RawLogEntry{
		LogLevel: level,
		Messages: []any{r.Message},
		Fields:   fields,
		Ctx:      ctx,
		// slog.Record.PC is set when the caller used slog.HandlerOptions
		// {AddSource: true} (or any handler-aware emitter that filled it).
		// Forward it as a Source so the loglayer side renders source info
		// without re-walking the stack. Zero PC produces nil; loglayer's
		// Config.Source.Enabled still applies as a fallback inside Raw.
		Source: loglayer.SourceFromPC(r.PC),
	})
	return nil
}

// WithAttrs returns a derived handler whose subsequent records carry
// attrs as persistent fields, nested under the current group path.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	out := &Handler{
		log:    h.log,
		groups: h.groups,
		attrs:  make([]groupedAttrs, len(h.attrs), len(h.attrs)+1),
	}
	copy(out.attrs, h.attrs)
	out.attrs = append(out.attrs, groupedAttrs{
		groups: slices.Clone(h.groups),
		attrs:  slices.Clone(attrs),
	})
	return out
}

// WithGroup returns a derived handler that nests subsequent attrs under
// name. Per slog convention, an empty name is a no-op.
func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return &Handler{
		log:    h.log,
		groups: append(slices.Clone(h.groups), name),
		attrs:  h.attrs,
	}
}

// slogTraceLevel is the slog level at which we map back to LogLevelTrace.
// slog has no Trace, so the slog transport's toSlogLevel synthesizes one
// at LevelDebug-4 and this constant mirrors that choice so the round-trip
// (loglayer.Trace → slog.Level → loglayer.Trace) stays stable.
const slogTraceLevel = slog.LevelDebug - 4

// slogToLoglayerLevel maps slog levels to loglayer levels. slog levels
// are arbitrary ints, so we partition by named-neighbour ranges. Values
// at or above LevelError pin to LogLevelError so a regular
// slog.Error(...) emission can never trip loglayer's Fatal exit.
func slogToLoglayerLevel(l slog.Level) loglayer.LogLevel {
	switch {
	case l <= slogTraceLevel:
		return loglayer.LogLevelTrace
	case l <= slog.LevelDebug:
		return loglayer.LogLevelDebug
	case l < slog.LevelWarn:
		return loglayer.LogLevelInfo
	case l < slog.LevelError:
		return loglayer.LogLevelWarn
	default:
		return loglayer.LogLevelError
	}
}

// mergeExisting copies the logger's current persistent fields into a
// fresh map so we can extend it without mutating the underlying logger.
// GetFields already returns a copy; mergeExisting just gives us a typed
// loglayer.Fields back.
func mergeExisting(existing loglayer.Fields) loglayer.Fields {
	if len(existing) == 0 {
		return make(loglayer.Fields)
	}
	return existing
}

// applyAttrsAtPath walks (and creates) maps for each segment of path,
// then applies attrs to the leaf map.
func applyAttrsAtPath(fields loglayer.Fields, path []string, attrs []slog.Attr) {
	target := mapAtPath(fields, path)
	for _, a := range attrs {
		applyAttr(target, a)
	}
}

// mapAtPath returns the nested map at path, creating intermediate maps
// as needed. An empty path returns fields itself (typed as map[string]any
// so the caller can use a single shape for nested writes).
func mapAtPath(fields loglayer.Fields, path []string) map[string]any {
	var m map[string]any = fields
	for _, p := range path {
		next, ok := m[p].(map[string]any)
		if !ok {
			next = make(map[string]any)
			m[p] = next
		}
		m = next
	}
	return m
}

// applyAttr writes a single slog.Attr into target. Group attrs recurse:
// a Group with a non-empty key nests its members under the key; a Group
// with an empty key inlines its members at this level (slog convention).
// LogValuer attrs are resolved before encoding so the resolved value is
// what lands in the log.
func applyAttr(target map[string]any, a slog.Attr) {
	v := a.Value.Resolve()
	if v.Kind() == slog.KindGroup {
		members := v.Group()
		if len(members) == 0 {
			return // empty group: drop, per slog
		}
		if a.Key == "" {
			// inline
			for _, m := range members {
				applyAttr(target, m)
			}
			return
		}
		next, ok := target[a.Key].(map[string]any)
		if !ok {
			next = make(map[string]any, len(members))
			target[a.Key] = next
		}
		for _, m := range members {
			applyAttr(next, m)
		}
		return
	}
	if a.Key == "" {
		return // empty key, non-group: drop, per slog
	}
	target[a.Key] = valueOf(v)
}

// valueOf extracts the Go value from a resolved slog.Value. Native kinds
// preserve their typed Go value (string, int64, bool, time.Time, ...)
// so transports that special-case those types still work.
func valueOf(v slog.Value) any {
	switch v.Kind() {
	case slog.KindString:
		return v.String()
	case slog.KindInt64:
		return v.Int64()
	case slog.KindUint64:
		return v.Uint64()
	case slog.KindFloat64:
		return v.Float64()
	case slog.KindBool:
		return v.Bool()
	case slog.KindTime:
		return v.Time()
	case slog.KindDuration:
		return v.Duration()
	case slog.KindAny:
		return v.Any()
	default:
		return v.Any()
	}
}
