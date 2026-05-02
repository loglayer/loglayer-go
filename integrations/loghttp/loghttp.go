// Package loghttp provides HTTP middleware that derives a per-request logger
// from a base loglayer.LogLayer and stores it in the request context, so
// downstream handlers can log with request-scoped fields automatically.
//
// Drop the middleware in once at server setup; every handler downstream gets
// a logger pre-populated with the request ID, method, and path.
//
//	mux := http.NewServeMux()
//	mux.HandleFunc("/users", handler)
//	http.ListenAndServe(":8080", loghttp.Middleware(log, loghttp.Config{})(mux))
//
//	func handler(w http.ResponseWriter, r *http.Request) {
//	    log := loghttp.FromRequest(r)
//	    log.Info("doing work") // includes requestId/method/path
//	}
//
// Mirrors the role that hlog.NewHandler plays for zerolog.
package loghttp

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.loglayer.dev/v2"
	"go.loglayer.dev/v2/utils/idgen"
	"go.loglayer.dev/v2/utils/sanitize"
)

// FieldNames customizes the keys emitted by the middleware. Empty values fall
// back to the default for that field.
type FieldNames struct {
	RequestID  string // default "requestId"
	Method     string // default "method"
	Path       string // default "path"
	Status     string // default "status"
	DurationMs string // default "durationMs"
	Bytes      string // default "bytes"
}

// Config holds middleware configuration. All fields are optional; the
// zero-value Config gives you sensible defaults (X-Request-ID header,
// random fallback IDs, the standard six field names, no start-log, status
// → level mapping for 4xx/5xx).
type Config struct {
	// RequestIDHeader is the HTTP header read for an incoming request ID.
	// Defaults to "X-Request-ID".
	RequestIDHeader string

	// RequestIDGenerator is called when no incoming request-ID header is
	// present. Defaults to 12 hex chars from crypto/rand.
	RequestIDGenerator func() string

	// FieldNames overrides the keys used in the emitted logs. Empty
	// fields here fall back to defaults; the middleware always emits all
	// six.
	FieldNames FieldNames

	// StartLog emits a "request started" line at the beginning of every
	// request, in addition to the "request completed" line at the end.
	// Defaults to false to keep log volume low. Ignored when
	// ShouldStartLog is non-nil.
	StartLog bool

	// ShouldStartLog, when non-nil, is consulted per request to decide
	// whether to emit the "request started" line. Use it for sampling
	// (e.g. log 1% of requests) or conditional logging (e.g. only when
	// a debug header is set). Returning true emits the start line;
	// false skips it. The "request completed" line still emits
	// regardless. When ShouldStartLog is set, StartLog is ignored.
	ShouldStartLog func(r *http.Request) bool

	// StatusLevels picks a log level for the "request completed" line
	// based on the response status code. Defaults to:
	//
	//   - 5xx → LogLevelError
	//   - 4xx → LogLevelWarn
	//   - else → LogLevelInfo
	StatusLevels func(status int) loglayer.LogLevel

	// ExtraFields, if set, is called once per request to attach extra
	// fields to the per-request logger. Useful for tenant/user/trace IDs
	// extracted from the request.
	ExtraFields func(*http.Request) loglayer.Fields
}

func (c Config) withDefaults() Config {
	out := c
	if out.RequestIDHeader == "" {
		out.RequestIDHeader = "X-Request-ID"
	}
	if out.RequestIDGenerator == nil {
		out.RequestIDGenerator = func() string { return idgen.Random("") }
	}
	if out.FieldNames.RequestID == "" {
		out.FieldNames.RequestID = "requestId"
	}
	if out.FieldNames.Method == "" {
		out.FieldNames.Method = "method"
	}
	if out.FieldNames.Path == "" {
		out.FieldNames.Path = "path"
	}
	if out.FieldNames.Status == "" {
		out.FieldNames.Status = "status"
	}
	if out.FieldNames.DurationMs == "" {
		out.FieldNames.DurationMs = "durationMs"
	}
	if out.FieldNames.Bytes == "" {
		out.FieldNames.Bytes = "bytes"
	}
	if out.StatusLevels == nil {
		out.StatusLevels = defaultStatusLevels
	}
	return out
}

func defaultStatusLevels(status int) loglayer.LogLevel {
	switch {
	case status >= 500:
		return loglayer.LogLevelError
	case status >= 400:
		return loglayer.LogLevelWarn
	default:
		return loglayer.LogLevelInfo
	}
}

// Middleware returns an HTTP middleware that derives a per-request logger
// from log and stores it in the request context. Emits a "request completed"
// log line at the end of every request, with status code, bytes written,
// and duration in metadata. The base log is never mutated.
//
// Pass a zero-value Config (loglayer.Config{}) to take the defaults; only
// set the fields you want to override.
func Middleware(log *loglayer.LogLayer, cfg Config) func(http.Handler) http.Handler {
	c := cfg.withDefaults()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := r.Header.Get(c.RequestIDHeader)
			if reqID == "" {
				reqID = c.RequestIDGenerator()
			}

			extras := map[string]any(nil)
			if c.ExtraFields != nil {
				extras = c.ExtraFields(r)
			}
			fields := make(loglayer.Fields, 3+len(extras))
			// Sanitize attacker-controllable strings: an X-Request-ID
			// header (or method / path on a malformed request) carrying
			// CR/LF/ESC could otherwise forge new log lines or smuggle
			// ANSI escape sequences. extras are caller-controlled, so
			// the application owns sanitization there if it pulls from
			// untrusted sources.
			fields[c.FieldNames.RequestID] = sanitizeForLog(reqID)
			fields[c.FieldNames.Method] = sanitizeForLog(r.Method)
			fields[c.FieldNames.Path] = sanitizeForLog(r.URL.Path)
			for k, v := range extras {
				fields[k] = v
			}

			// Bind r.Context() to the per-request logger so handlers can
			// just call .Info(...) and any plugin reading TransportParams.Ctx
			// (e.g. trace-injectors) sees the request's context without the
			// caller having to chain WithContext on every emission.
			reqLog := log.WithFields(fields).WithContext(r.Context())
			r = r.WithContext(loglayer.NewContext(r.Context(), reqLog))

			sw := wrapWriter(w)
			start := time.Now()

			if shouldEmitStart(c, r) {
				reqLog.Info("request started")
			}

			// Defer panic-recovery so the request-{completed,panicked}
			// log emits even when the handler crashes. We re-panic so
			// any outer recovery middleware (chi.Recoverer, an APM
			// interceptor, etc.) still gets to act; without our recover,
			// the panic propagates and the log line is lost.
			func() {
				defer func() {
					if rcv := recover(); rcv != nil {
						status := sw.status
						if status == 0 {
							status = http.StatusInternalServerError
						}
						// Sanitize the formatted panic value: a custom
						// error type passed to panic() could carry
						// ANSI / CRLF / bidi controls in its String()
						// output, which would smuggle straight into the
						// log otherwise. Bound length so a huge panic
						// payload doesn't bloat the record.
						reqLog.WithMetadata(loglayer.Metadata{
							c.FieldNames.Status:     status,
							c.FieldNames.DurationMs: time.Since(start).Milliseconds(),
							c.FieldNames.Bytes:      sw.bytes,
							"panic":                 sanitizeForLog(fmt.Sprintf("%v", rcv)),
						}).Error("request panicked")
						panic(rcv)
					}
				}()
				next.ServeHTTP(sw, r)
			}()

			status := sw.status
			if status == 0 {
				status = http.StatusOK
			}
			reqLog.Raw(loglayer.RawLogEntry{
				LogLevel: c.StatusLevels(status),
				Messages: []any{"request completed"},
				Metadata: loglayer.Metadata{
					c.FieldNames.Status:     status,
					c.FieldNames.DurationMs: time.Since(start).Milliseconds(),
					c.FieldNames.Bytes:      sw.bytes,
				},
			})
		})
	}
}

// FromRequest returns the per-request logger stored in r's context, or nil if
// the middleware was not applied. Equivalent to loglayer.FromContext(r.Context()).
func FromRequest(r *http.Request) *loglayer.LogLayer {
	return loglayer.FromContext(r.Context())
}

// MustFromRequest is FromRequest but panics if no logger is attached. Use it
// in handlers that always sit behind the middleware.
func MustFromRequest(r *http.Request) *loglayer.LogLayer {
	return loglayer.MustFromContext(r.Context())
}

// responseWriter captures status code and bytes written. Other optional
// http.ResponseWriter interfaces (Flusher, Hijacker, Pusher) are reachable
// via http.NewResponseController, which calls Unwrap below.
type responseWriter struct {
	http.ResponseWriter
	status        int
	bytes         int
	headerWritten bool
}

func wrapWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w}
}

// shouldEmitStart consults ShouldStartLog when present, else falls back
// to the StartLog bool. Centralizes the precedence so both code paths
// stay consistent if more knobs land here.
func shouldEmitStart(c Config, r *http.Request) bool {
	if c.ShouldStartLog != nil {
		return c.ShouldStartLog(r)
	}
	return c.StartLog
}

// maxLoggedHeaderLen bounds sanitized header/path/method values that
// land in log fields. Real-world request IDs are ~36 bytes (UUID) and
// paths are typically under 1k; 4096 is generous enough that no legit
// caller hits it but small enough to stop a 1MB-long X-Request-ID from
// bloating a log record.
const maxLoggedHeaderLen = 4096

// sanitizeForLog bounds the value's length and strips control characters,
// so an attacker-controlled HTTP header can't forge log lines or smuggle
// ANSI escapes. Truncation goes through strings.Clone so the resulting
// string does not pin a megabyte-sized backing array via the slice
// header for the lifetime of the request-scoped logger.
func sanitizeForLog(s string) string {
	if len(s) > maxLoggedHeaderLen {
		s = strings.Clone(s[:maxLoggedHeaderLen])
	}
	return sanitize.Message(s)
}

func (w *responseWriter) WriteHeader(status int) {
	if !w.headerWritten {
		w.status = status
		w.headerWritten = true
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseWriter) Write(b []byte) (int, error) {
	if !w.headerWritten {
		w.status = http.StatusOK
		w.headerWritten = true
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

// Unwrap exposes the underlying ResponseWriter so http.NewResponseController
// can reach optional interfaces like Flusher and Hijacker (Go 1.20+).
func (w *responseWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
