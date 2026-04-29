package loghttp_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.loglayer.dev"
	"go.loglayer.dev/integrations/loghttp"
	"go.loglayer.dev/transport"
	"go.loglayer.dev/transport/transporttest"
	lltest "go.loglayer.dev/transports/testing"
)

func setupLogger(t *testing.T) (*loglayer.LogLayer, *lltest.TestLoggingLibrary) {
	t.Helper()
	lib := &lltest.TestLoggingLibrary{}
	tr := lltest.New(lltest.Config{
		BaseConfig: transport.BaseConfig{ID: "test"},
		Library:    lib,
	})
	log := loglayer.New(loglayer.Config{
		Transport:        tr,
		DisableFatalExit: true,
	})
	return log, lib
}

func runOne(t *testing.T, h http.Handler, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func TestMiddleware_RequestCompletedLog(t *testing.T) {
	log, lib := setupLogger(t)
	handler := loghttp.Middleware(log, loghttp.Config{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/foo", nil)
	req.Header.Set("X-Request-ID", "fixed-id")
	runOne(t, handler, req)

	if lib.Len() != 1 {
		t.Fatalf("expected 1 log line, got %d", lib.Len())
	}
	line := lib.PopLine()
	if !transporttest.MessageContains(line.Messages, "request completed") {
		t.Errorf("message: got %v", line.Messages)
	}
	if line.Level != loglayer.LogLevelInfo {
		t.Errorf("level: got %s, want info", line.Level)
	}
	if line.Data["requestId"] != "fixed-id" {
		t.Errorf("requestId: got %v", line.Data["requestId"])
	}
	if line.Data["method"] != http.MethodGet {
		t.Errorf("method: got %v", line.Data["method"])
	}
	if line.Data["path"] != "/foo" {
		t.Errorf("path: got %v", line.Data["path"])
	}
	meta, _ := line.Metadata.(loglayer.Metadata)
	if meta == nil {
		t.Fatalf("expected metadata, got %T", line.Metadata)
	}
	if meta["status"] != http.StatusOK {
		t.Errorf("status: got %v", meta["status"])
	}
	if _, ok := meta["durationMs"]; !ok {
		t.Errorf("durationMs missing from metadata")
	}
	if meta["bytes"] != 2 {
		t.Errorf("bytes: got %v want 2", meta["bytes"])
	}
}

func TestMiddleware_GeneratesRequestIDWhenAbsent(t *testing.T) {
	log, lib := setupLogger(t)
	handler := loghttp.Middleware(log, loghttp.Config{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	runOne(t, handler, httptest.NewRequest(http.MethodGet, "/", nil))

	line := lib.PopLine()
	id, _ := line.Data["requestId"].(string)
	if id == "" {
		t.Errorf("expected generated requestId, got empty")
	}
	if len(id) != 12 {
		t.Errorf("expected 12-char hex requestId, got %q", id)
	}
}

func TestMiddleware_FromRequest(t *testing.T) {
	log, lib := setupLogger(t)

	var inner *loglayer.LogLayer
	handler := loghttp.Middleware(log, loghttp.Config{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inner = loghttp.FromRequest(r)
		inner.Info("inside handler")
	}))

	runOne(t, handler, httptest.NewRequest(http.MethodGet, "/handler-test", nil))

	if inner == nil {
		t.Fatalf("FromRequest returned nil")
	}
	lines := lib.Lines()
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines (handler + completed), got %d", len(lines))
	}
	if !transporttest.MessageContains(lines[0].Messages, "inside handler") {
		t.Errorf("expected 'inside handler' first, got %v", lines[0].Messages)
	}
	if lines[0].Data["path"] != "/handler-test" {
		t.Errorf("inside-handler log should carry path field, got %v", lines[0].Data)
	}
}

func TestMiddleware_StatusCapture(t *testing.T) {
	log, lib := setupLogger(t)
	handler := loghttp.Middleware(log, loghttp.Config{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	runOne(t, handler, httptest.NewRequest(http.MethodGet, "/missing", nil))

	line := lib.PopLine()
	if line.Level != loglayer.LogLevelWarn {
		t.Errorf("4xx should escalate to Warn, got %s", line.Level)
	}
	meta, _ := line.Metadata.(loglayer.Metadata)
	if meta["status"] != http.StatusNotFound {
		t.Errorf("status: got %v", meta["status"])
	}
}

func TestMiddleware_ServerErrorEscalatesToError(t *testing.T) {
	log, lib := setupLogger(t)
	handler := loghttp.Middleware(log, loghttp.Config{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	runOne(t, handler, httptest.NewRequest(http.MethodPost, "/boom", nil))

	line := lib.PopLine()
	if line.Level != loglayer.LogLevelError {
		t.Errorf("5xx should escalate to Error, got %s", line.Level)
	}
}

func TestMiddleware_StartLogToggle(t *testing.T) {
	log, lib := setupLogger(t)
	handler := loghttp.Middleware(log, loghttp.Config{StartLog: true})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	runOne(t, handler, httptest.NewRequest(http.MethodGet, "/", nil))

	if lib.Len() != 2 {
		t.Fatalf("expected 2 log lines (start + end), got %d", lib.Len())
	}
	lines := lib.Lines()
	if !transporttest.MessageContains(lines[0].Messages, "request started") {
		t.Errorf("first line should be 'request started', got %v", lines[0].Messages)
	}
	if !transporttest.MessageContains(lines[1].Messages, "request completed") {
		t.Errorf("second line should be 'request completed', got %v", lines[1].Messages)
	}
}

// ShouldStartLog overrides the StartLog bool. Allows sampling or
// per-request decisions (e.g. only log starts for a debug header).
func TestMiddleware_ShouldStartLog(t *testing.T) {
	log, lib := setupLogger(t)
	handler := loghttp.Middleware(log, loghttp.Config{
		StartLog: false, // would normally suppress starts
		ShouldStartLog: func(r *http.Request) bool {
			return r.Header.Get("X-Debug") == "1"
		},
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	// Request without the header: only the completed line.
	runOne(t, handler, httptest.NewRequest(http.MethodGet, "/", nil))
	if lib.Len() != 1 {
		t.Errorf("no-debug-header request should emit 1 line, got %d", lib.Len())
	}
	lib.ClearLines()

	// Request with the header: start + completed.
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Debug", "1")
	runOne(t, handler, r)
	if lib.Len() != 2 {
		t.Errorf("debug-header request should emit 2 lines, got %d", lib.Len())
	}
}

func TestMiddleware_CustomFieldNames(t *testing.T) {
	log, lib := setupLogger(t)
	handler := loghttp.Middleware(log, loghttp.Config{
		FieldNames: loghttp.FieldNames{
			RequestID: "trace_id",
			Status:    "http_status",
		},
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "abc")
	runOne(t, handler, req)

	line := lib.PopLine()
	if line.Data["trace_id"] != "abc" {
		t.Errorf("trace_id: got %v", line.Data["trace_id"])
	}
	if _, present := line.Data["requestId"]; present {
		t.Errorf("default requestId key should not appear when overridden")
	}
	meta, _ := line.Metadata.(loglayer.Metadata)
	if meta["http_status"] != http.StatusOK {
		t.Errorf("http_status: got %v", meta["http_status"])
	}
}

func TestMiddleware_ExtraFields(t *testing.T) {
	log, lib := setupLogger(t)
	handler := loghttp.Middleware(log, loghttp.Config{
		ExtraFields: func(r *http.Request) loglayer.Fields {
			return loglayer.Fields{"tenant": r.Header.Get("X-Tenant")}
		},
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Tenant", "acme")
	runOne(t, handler, req)

	line := lib.PopLine()
	if line.Data["tenant"] != "acme" {
		t.Errorf("tenant: got %v", line.Data["tenant"])
	}
}

func TestMiddleware_BaseLoggerNotMutated(t *testing.T) {
	log, _ := setupLogger(t)
	beforeFields := log.GetFields()

	handler := loghttp.Middleware(log, loghttp.Config{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	for i := 0; i < 5; i++ {
		runOne(t, handler, httptest.NewRequest(http.MethodGet, "/", nil))
	}

	afterFields := log.GetFields()
	if len(beforeFields) != len(afterFields) {
		t.Errorf("base logger fields mutated: before=%v after=%v", beforeFields, afterFields)
	}
}

func TestMiddleware_CustomRequestIDHeader(t *testing.T) {
	log, lib := setupLogger(t)
	handler := loghttp.Middleware(log, loghttp.Config{RequestIDHeader: "X-Trace"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Trace", "trace-xyz")
	runOne(t, handler, req)

	line := lib.PopLine()
	if line.Data["requestId"] != "trace-xyz" {
		t.Errorf("requestId from custom header: got %v", line.Data["requestId"])
	}
}

func TestMiddleware_MustFromRequestPanicsWithoutMiddleware(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected panic when MustFromRequest called outside middleware")
		}
	}()
	loghttp.MustFromRequest(httptest.NewRequest(http.MethodGet, "/", nil))
}

// The middleware binds r.Context() to the per-request logger so handlers
// don't need to chain WithContext on every emission. Plugins that read
// TransportParams.Ctx (trace injectors, cancellation gates) see the
// request context for free.
func TestMiddleware_BindsRequestContextToLogger(t *testing.T) {
	type ctxKey struct{}
	log, lib := setupLogger(t)

	handler := loghttp.Middleware(log, loghttp.Config{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Plain Info call — no WithContext chain.
		loghttp.FromRequest(r).Info("inside handler")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(context.WithValue(req.Context(), ctxKey{}, "request-marker"))
	runOne(t, handler, req)

	lines := lib.Lines()
	if len(lines) < 1 {
		t.Fatalf("expected at least one captured line, got %d", len(lines))
	}
	for i, line := range lines {
		if line.Ctx == nil {
			t.Errorf("line %d: ctx not bound by middleware", i)
			continue
		}
		if got := line.Ctx.Value(ctxKey{}); got != "request-marker" {
			t.Errorf("line %d: ctx value got %v, want request-marker", i, got)
		}
	}
}

// When a handler panics, the middleware logs a "request panicked" entry
// at Error level and re-panics so any outer recovery middleware (or
// http.Server's connection-close behavior) still acts. Without this,
// the request-completed log would be lost and the failure would have no
// log trail.
func TestMiddleware_PanicEmitsLogAndRePanics(t *testing.T) {
	log, lib := setupLogger(t)
	handler := loghttp.Middleware(log, loghttp.Config{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))

	defer func() {
		rcv := recover()
		if rcv == nil {
			t.Fatal("middleware should re-panic so outer recovery can act")
		}
		if rcv != "boom" {
			t.Errorf("re-panicked value: got %v, want %q", rcv, "boom")
		}

		// The "request panicked" log should have emitted before the
		// re-panic.
		lines := lib.Lines()
		if len(lines) != 1 {
			t.Fatalf("expected 1 log line, got %d", len(lines))
		}
		if !transporttest.MessageContains(lines[0].Messages, "request panicked") {
			t.Errorf("message: got %v, want 'request panicked'", lines[0].Messages)
		}
		if lines[0].Level != loglayer.LogLevelError {
			t.Errorf("level: got %v, want Error", lines[0].Level)
		}
		// Metadata should carry the panic value, status (500 since the
		// handler never wrote), duration, and bytes.
		m, ok := lines[0].Metadata.(loglayer.Metadata)
		if !ok {
			t.Fatalf("metadata: got %T, want loglayer.Metadata", lines[0].Metadata)
		}
		if m["panic"] != "boom" {
			t.Errorf("metadata.panic: got %v", m["panic"])
		}
		if m["status"] != http.StatusInternalServerError {
			t.Errorf("metadata.status: got %v, want 500", m["status"])
		}
	}()

	runOne(t, handler, httptest.NewRequest(http.MethodGet, "/", nil))
}

// A handler that panics with a value carrying ANSI/CRLF must not
// smuggle those control sequences into the "request panicked" log.
// The middleware sanitizes the formatted panic value before storing
// it in the metadata.
func TestMiddleware_PanicValueIsSanitized(t *testing.T) {
	log, lib := setupLogger(t)
	handler := loghttp.Middleware(log, loghttp.Config{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("evil\r\n[FORGED] line\x1b[31m red")
	}))

	defer func() {
		if rcv := recover(); rcv == nil {
			t.Fatal("middleware should re-panic")
		}
		lines := lib.Lines()
		if len(lines) != 1 {
			t.Fatalf("expected 1 log line, got %d", len(lines))
		}
		m, ok := lines[0].Metadata.(loglayer.Metadata)
		if !ok {
			t.Fatalf("metadata: %T", lines[0].Metadata)
		}
		gotPanic, _ := m["panic"].(string)
		if strings.ContainsAny(gotPanic, "\r\n\x1b") {
			t.Errorf("panic value still contains control chars: %q", gotPanic)
		}
		// Visible content survives the sanitize.
		if !strings.Contains(gotPanic, "evil") || !strings.Contains(gotPanic, "[FORGED] line") {
			t.Errorf("printable content lost during sanitize: %q", gotPanic)
		}
	}()

	runOne(t, handler, httptest.NewRequest(http.MethodGet, "/", nil))
}

// Untrusted HTTP headers and paths that contain CR/LF/ESC could
// otherwise forge log lines or smuggle ANSI escape sequences. The
// middleware strips ASCII control characters before storing them as
// fields.
func TestMiddleware_SanitizesUntrustedInput(t *testing.T) {
	log, lib := setupLogger(t)
	handler := loghttp.Middleware(log, loghttp.Config{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	// X-Request-ID with a CRLF + faked log line.
	r.Header.Set("X-Request-ID", "real-id\r\n[ERROR] forged")
	// Path with an ESC + ANSI red.
	r.URL.Path = "/api\x1b[31m/users"

	runOne(t, handler, r)

	line := lib.PopLine()
	gotID, _ := line.Data["requestId"].(string)
	if strings.ContainsAny(gotID, "\r\n\x1b") {
		t.Errorf("requestId still contains control chars: %q", gotID)
	}
	if gotID != "real-id[ERROR] forged" {
		// The sanitizer drops control chars but leaves printable
		// content; ensure that's what we see.
		t.Errorf("requestId after sanitize: got %q, want %q", gotID, "real-id[ERROR] forged")
	}
	gotPath, _ := line.Data["path"].(string)
	if strings.ContainsAny(gotPath, "\r\n\x1b") {
		t.Errorf("path still contains control chars: %q", gotPath)
	}
}
