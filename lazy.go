package loglayer

// LazyEvalError is the placeholder substituted into a log entry when a
// [Lazy] callback panics.
const LazyEvalError = "[LazyEvalError]"

// LazyValue wraps a callback evaluated at log dispatch time. Construct
// with [Lazy] and store as a value in [Fields]. See the Lazy
// Evaluation docs for placement and timing.
type LazyValue struct {
	fn func() any
}

// Lazy wraps fn so it is invoked only at log emit time, and only when
// the level is enabled. Re-evaluated on every emission from a logger
// holding it; child loggers inherit the wrapper, not a resolved value.
func Lazy(fn func() any) *LazyValue {
	return &LazyValue{fn: fn}
}

// resolve runs the callback under a recover wrap. On panic it returns
// [LazyEvalError] so the rest of the entry still emits.
func (lv *LazyValue) resolve() (out any) {
	defer func() {
		if recover() != nil {
			out = LazyEvalError
		}
	}()
	return lv.fn()
}

// mapHasLazy reports whether m contains any [*LazyValue] entries.
// Called at attach time to set a precomputed flag so the dispatch hot
// path can skip resolution when no lazies are present.
func mapHasLazy(m map[string]any) bool {
	for _, v := range m {
		if _, ok := v.(*LazyValue); ok {
			return true
		}
	}
	return false
}

// resolveLazyFields returns a new [Fields] with every [*LazyValue]
// resolved. Callers gate this on a precomputed flag so the no-lazy
// hot path doesn't pay the allocation.
func resolveLazyFields(fields Fields) Fields {
	out := make(Fields, len(fields))
	for k, v := range fields {
		if lv, ok := v.(*LazyValue); ok {
			out[k] = lv.resolve()
			continue
		}
		out[k] = v
	}
	return out
}
