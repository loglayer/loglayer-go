// Package loghttp provides HTTP middleware that derives a per-request logger
// from a base loglayer.LogLayer and stores it in the request context, so
// downstream handlers can log with request-scoped fields automatically.
//
// Drop the middleware in once at server setup; every handler downstream gets
// a logger pre-populated with the request ID, method, and path.
//
//	mux := http.NewServeMux()
//	mux.HandleFunc("/users", handler)
//	http.ListenAndServe(":8080", loghttp.Middleware(log)(mux))
//
//	func handler(w http.ResponseWriter, r *http.Request) {
//	    log := loghttp.FromRequest(r)
//	    log.Info("doing work") // includes requestId/method/path
//	}
//
// Mirrors the role that hlog.NewHandler plays for zerolog.
package loghttp

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"go.loglayer.dev"
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

type config struct {
	requestIDHeader string
	requestIDGen    func() string
	fieldNames      FieldNames
	startLog        bool
	statusLevels    func(int) loglayer.LogLevel
	extraFields     func(*http.Request) loglayer.Fields
}

// Option configures the middleware.
type Option func(*config)

// WithRequestIDHeader sets the HTTP header read for an incoming request ID.
// Default: "X-Request-ID".
func WithRequestIDHeader(name string) Option {
	return func(c *config) { c.requestIDHeader = name }
}

// WithRequestIDGenerator sets the function called when no request ID header
// is present. Default: 8 random bytes hex-encoded.
func WithRequestIDGenerator(fn func() string) Option {
	return func(c *config) { c.requestIDGen = fn }
}

// WithFieldNames overrides the field keys used in the emitted logs.
// Fields left empty in names keep their defaults; this option cannot be used
// to disable a field, only to rename it. The middleware always emits all six.
func WithFieldNames(names FieldNames) Option {
	return func(c *config) {
		if names.RequestID != "" {
			c.fieldNames.RequestID = names.RequestID
		}
		if names.Method != "" {
			c.fieldNames.Method = names.Method
		}
		if names.Path != "" {
			c.fieldNames.Path = names.Path
		}
		if names.Status != "" {
			c.fieldNames.Status = names.Status
		}
		if names.DurationMs != "" {
			c.fieldNames.DurationMs = names.DurationMs
		}
		if names.Bytes != "" {
			c.fieldNames.Bytes = names.Bytes
		}
	}
}

// WithStartLog controls whether a "request started" line is emitted at the
// start of every request (in addition to "request completed" at the end).
// Default false to keep log volume low.
func WithStartLog(enabled bool) Option {
	return func(c *config) { c.startLog = enabled }
}

// WithStatusLevels overrides the function that picks a log level for the
// "request completed" line based on the response status code. Default:
//
//   - 5xx → LogLevelError
//   - 4xx → LogLevelWarn
//   - else → LogLevelInfo
func WithStatusLevels(fn func(status int) loglayer.LogLevel) Option {
	return func(c *config) { c.statusLevels = fn }
}

// WithExtraFields attaches additional fields to the per-request logger.
// Useful for tenant ID, user ID, trace ID, etc. extracted from the request.
func WithExtraFields(fn func(*http.Request) loglayer.Fields) Option {
	return func(c *config) { c.extraFields = fn }
}

func defaultConfig() *config {
	return &config{
		requestIDHeader: "X-Request-ID",
		requestIDGen:    randomID,
		fieldNames: FieldNames{
			RequestID:  "requestId",
			Method:     "method",
			Path:       "path",
			Status:     "status",
			DurationMs: "durationMs",
			Bytes:      "bytes",
		},
		statusLevels: defaultStatusLevels,
	}
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

func randomID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// Middleware returns an HTTP middleware that derives a per-request logger
// from log and stores it in the request context. The middleware emits a
// "request completed" log line at the end of every request, with status code,
// bytes written, and duration in metadata. The base log is never mutated.
func Middleware(log *loglayer.LogLayer, opts ...Option) func(http.Handler) http.Handler {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := r.Header.Get(cfg.requestIDHeader)
			if reqID == "" {
				reqID = cfg.requestIDGen()
			}

			extras := map[string]any(nil)
			if cfg.extraFields != nil {
				extras = cfg.extraFields(r)
			}
			fields := make(loglayer.Fields, 3+len(extras))
			fields[cfg.fieldNames.RequestID] = reqID
			fields[cfg.fieldNames.Method] = r.Method
			fields[cfg.fieldNames.Path] = r.URL.Path
			for k, v := range extras {
				fields[k] = v
			}

			reqLog := log.WithFields(fields)
			r = r.WithContext(loglayer.NewContext(r.Context(), reqLog))

			sw := wrapWriter(w)
			start := time.Now()

			if cfg.startLog {
				reqLog.Info("request started")
			}

			next.ServeHTTP(sw, r)

			status := sw.status
			if status == 0 {
				status = http.StatusOK
			}
			reqLog.Raw(loglayer.RawLogEntry{
				LogLevel: cfg.statusLevels(status),
				Messages: []any{"request completed"},
				Metadata: loglayer.Metadata{
					cfg.fieldNames.Status:     status,
					cfg.fieldNames.DurationMs: time.Since(start).Milliseconds(),
					cfg.fieldNames.Bytes:      sw.bytes,
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
