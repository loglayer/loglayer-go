package loghttp_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.loglayer.dev/loglayer"
	"go.loglayer.dev/loglayer/internal/transporttest"
	"go.loglayer.dev/loglayer/integrations/loghttp"
	"go.loglayer.dev/loglayer/transport"
	lltest "go.loglayer.dev/loglayer/transports/testing"
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
	handler := loghttp.Middleware(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	handler := loghttp.Middleware(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	runOne(t, handler, httptest.NewRequest(http.MethodGet, "/", nil))

	line := lib.PopLine()
	id, _ := line.Data["requestId"].(string)
	if id == "" {
		t.Errorf("expected generated requestId, got empty")
	}
	if len(id) != 16 {
		t.Errorf("expected 16-char hex requestId, got %q", id)
	}
}

func TestMiddleware_FromRequest(t *testing.T) {
	log, lib := setupLogger(t)

	var inner *loglayer.LogLayer
	handler := loghttp.Middleware(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	handler := loghttp.Middleware(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	handler := loghttp.Middleware(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	handler := loghttp.Middleware(log, loghttp.WithStartLog(true))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

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

func TestMiddleware_CustomFieldNames(t *testing.T) {
	log, lib := setupLogger(t)
	handler := loghttp.Middleware(log,
		loghttp.WithFieldNames(loghttp.FieldNames{
			RequestID: "trace_id",
			Status:    "http_status",
		}),
	)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

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
	handler := loghttp.Middleware(log,
		loghttp.WithExtraFields(func(r *http.Request) loglayer.Fields {
			return loglayer.Fields{"tenant": r.Header.Get("X-Tenant")}
		}),
	)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

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

	handler := loghttp.Middleware(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
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
	handler := loghttp.Middleware(log, loghttp.WithRequestIDHeader("X-Trace"))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

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

