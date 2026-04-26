package loglayer

import "context"

// loggerKey is the unexported type used as the context.Context key when
// storing a *LogLayer via NewContext. The unexported type prevents collisions
// with keys defined in other packages.
type loggerKey struct{}

// NewContext returns a copy of parent in which the given LogLayer is attached.
// Use this in middleware (HTTP, gRPC, etc.) to make a request-scoped logger
// available to downstream code without threading it through every function
// signature. Recover the logger with FromContext.
//
// If log is nil, parent is returned unchanged.
func NewContext(parent context.Context, log *LogLayer) context.Context {
	if log == nil {
		return parent
	}
	return context.WithValue(parent, loggerKey{}, log)
}

// FromContext returns the LogLayer attached to ctx by NewContext, or nil if
// no logger was attached. Pair with MustFromContext if your code expects the
// logger to always be present (e.g. handlers behind middleware that always
// sets one).
func FromContext(ctx context.Context) *LogLayer {
	if ctx == nil {
		return nil
	}
	log, _ := ctx.Value(loggerKey{}).(*LogLayer)
	return log
}

// MustFromContext is like FromContext but panics if no logger is attached.
// Use it in handler code where middleware is guaranteed to have called
// NewContext; the panic surfaces a misconfiguration immediately rather than
// silently dropping logs.
func MustFromContext(ctx context.Context) *LogLayer {
	log := FromContext(ctx)
	if log == nil {
		panic("loglayer: no LogLayer attached to context (use NewContext in middleware)")
	}
	return log
}
