package central_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
	"go.loglayer.dev/transports/central"
	httptr "go.loglayer.dev/transports/http"
)

type capture struct {
	mu      sync.Mutex
	bodies  [][]byte
	headers []http.Header
	paths   []string
}

func (c *capture) handler(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	c.mu.Lock()
	c.bodies = append(c.bodies, body)
	c.headers = append(c.headers, r.Header.Clone())
	c.paths = append(c.paths, r.URL.Path)
	c.mu.Unlock()
	w.WriteHeader(http.StatusOK)
}

func (c *capture) snapshotBodies() [][]byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([][]byte, len(c.bodies))
	copy(out, c.bodies)
	return out
}

// decodeFirstBatch parses the first captured request body as a JSON array of
// entries. Fails the test if no body was captured or the body isn't valid JSON.
func decodeFirstBatch(t *testing.T, c *capture) []map[string]any {
	t.Helper()
	bodies := c.snapshotBodies()
	if len(bodies) == 0 {
		t.Fatal("no batches captured")
	}
	var arr []map[string]any
	if err := json.Unmarshal(bodies[0], &arr); err != nil {
		t.Fatalf("body is not JSON: %v: %q", err, bodies[0])
	}
	return arr
}

// newTestCentral spins up an httptest server, builds a central.Transport
// pointed at it, and wraps it in a loglayer logger. Defaults Service="svc"
// and a batch large enough that Close drains pending entries (callers that
// want immediate per-call flushing override HTTP.BatchSize/BatchInterval).
// The server is closed via t.Cleanup; callers still call tr.Close() before
// asserting on captured bodies (Close is what drains the worker).
//
// cfg.BaseURL is always overwritten with the test server's URL. Tests that
// need a different BaseURL (e.g. trailing-slash handling) or a non-default
// loglayer.Config (e.g. custom ErrorFieldName) set up manually instead.
func newTestCentral(t *testing.T, cfg central.Config) (*loglayer.LogLayer, *central.Transport, *capture) {
	t.Helper()
	cap := &capture{}
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	t.Cleanup(srv.Close)

	if cfg.Service == "" {
		cfg.Service = "svc"
	}
	cfg.BaseURL = srv.URL
	if cfg.HTTP.BatchSize == 0 {
		cfg.HTTP.BatchSize = 10
	}
	if cfg.HTTP.BatchInterval == 0 {
		cfg.HTTP.BatchInterval = time.Hour
	}

	tr := central.New(cfg)
	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})
	return log, tr, cap
}

func TestCentral_Constants(t *testing.T) {
	if central.DefaultPort != 9800 {
		t.Errorf("DefaultPort: got %d, want 9800", central.DefaultPort)
	}
	if central.DefaultBaseURL != "http://localhost:9800" {
		t.Errorf("DefaultBaseURL: got %q, want http://localhost:9800", central.DefaultBaseURL)
	}
}

func TestCentral_PanicsWithoutService(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic when Service missing")
		}
		err, ok := r.(error)
		if !ok || !errors.Is(err, central.ErrServiceRequired) {
			t.Errorf("panic value: got %v, want ErrServiceRequired", r)
		}
	}()
	_ = central.New(central.Config{})
}

func TestCentral_Build_ReturnsErrServiceRequired(t *testing.T) {
	_, err := central.Build(central.Config{})
	if !errors.Is(err, central.ErrServiceRequired) {
		t.Errorf("Build with missing Service: got %v, want ErrServiceRequired", err)
	}
}

// HTTP.URL and HTTP.Encoder are managed by the central transport itself;
// setting them via the embedded HTTP config is rejected so silent overrides
// don't surprise callers.
func TestCentral_Build_RejectsHTTPOverrides(t *testing.T) {
	noopEncoder := httptr.EncoderFunc(func(_ []httptr.Entry) ([]byte, string, error) {
		return nil, "", nil
	})
	cases := []struct {
		name string
		http httptr.Config
	}{
		{"URL set", httptr.Config{URL: "http://my-forwarder.internal/v1/logs"}},
		{"Encoder set", httptr.Config{Encoder: noopEncoder}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := central.Build(central.Config{Service: "svc", HTTP: c.http})
			if !errors.Is(err, central.ErrHTTPOverrideForbidden) {
				t.Errorf("got %v, want ErrHTTPOverrideForbidden", err)
			}
		})
	}
}

func TestCentral_PostsToBaseURLAPIPath(t *testing.T) {
	log, tr, cap := newTestCentral(t, central.Config{})
	log.Info("hello")
	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	cap.mu.Lock()
	defer cap.mu.Unlock()
	if len(cap.paths) != 1 {
		t.Fatalf("expected 1 request, got %d", len(cap.paths))
	}
	if cap.paths[0] != "/api/logs" {
		t.Errorf("path: got %q, want /api/logs", cap.paths[0])
	}
}

// Trailing slashes on BaseURL must be trimmed before /api/logs is appended,
// otherwise the URL ends up with a double slash that some intakes reject.
func TestCentral_TrimsTrailingSlashFromBaseURL(t *testing.T) {
	cap := &capture{}
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	t.Cleanup(srv.Close)

	tr := central.New(central.Config{
		Service: "svc",
		BaseURL: srv.URL + "/",
		HTTP:    httptr.Config{BatchSize: 10, BatchInterval: time.Hour},
	})
	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})
	log.Info("hello")
	_ = tr.Close()

	cap.mu.Lock()
	defer cap.mu.Unlock()
	if len(cap.paths) != 1 || cap.paths[0] != "/api/logs" {
		t.Errorf("trimmed path: got %v, want /api/logs", cap.paths)
	}
}

func TestCentral_PayloadShape(t *testing.T) {
	log, tr, cap := newTestCentral(t, central.Config{
		Service:    "test-service",
		InstanceID: "instance-1",
		Tags:       []string{"env:test", "team:platform"},
	})
	log = log.WithFields(loglayer.Fields{"requestId": "abc"})
	log.WithMetadata(loglayer.Metadata{"durationMs": 42}).Info("served", "request")
	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	cap.mu.Lock()
	contentType := cap.headers[0].Get("Content-Type")
	cap.mu.Unlock()
	if contentType != "application/json" {
		t.Errorf("Content-Type: got %q, want application/json", contentType)
	}

	arr := decodeFirstBatch(t, cap)
	if len(arr) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(arr))
	}
	obj := arr[0]

	if obj["service"] != "test-service" {
		t.Errorf("service: got %v", obj["service"])
	}
	if obj["message"] != "served request" {
		t.Errorf("message: got %v", obj["message"])
	}
	if obj["level"] != "info" {
		t.Errorf("level: got %v", obj["level"])
	}
	if obj["instanceId"] != "instance-1" {
		t.Errorf("instanceId: got %v", obj["instanceId"])
	}

	tags, ok := obj["tags"].([]any)
	if !ok {
		t.Fatalf("tags: got %T, want []any", obj["tags"])
	}
	if len(tags) != 2 || tags[0] != "env:test" || tags[1] != "team:platform" {
		t.Errorf("tags: got %v", tags)
	}

	ctx, ok := obj["context"].(map[string]any)
	if !ok {
		t.Fatalf("context: got %T, want map[string]any", obj["context"])
	}
	if ctx["requestId"] != "abc" {
		t.Errorf("context.requestId: got %v", ctx["requestId"])
	}

	meta, ok := obj["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("metadata: got %T, want map[string]any", obj["metadata"])
	}
	if meta["durationMs"] != float64(42) {
		t.Errorf("metadata.durationMs: got %v", meta["durationMs"])
	}

	ts, ok := obj["timestamp"].(string)
	if !ok || ts == "" {
		t.Errorf("timestamp: got %v", obj["timestamp"])
	}
	if _, err := time.Parse(time.RFC3339Nano, ts); err != nil {
		t.Errorf("timestamp not parseable as RFC3339: %v (%q)", err, ts)
	}
}

func TestCentral_LevelMapping(t *testing.T) {
	log, tr, cap := newTestCentral(t, central.Config{})

	log.Trace("t")
	log.Debug("d")
	log.Info("i")
	log.Warn("w")
	log.Error("e")
	log.Fatal("f")
	func() {
		defer func() { _ = recover() }()
		log.Panic("p")
	}()
	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	arr := decodeFirstBatch(t, cap)
	want := []string{"trace", "debug", "info", "warn", "error", "fatal", "panic"}
	if len(arr) != len(want) {
		t.Fatalf("expected %d entries, got %d", len(want), len(arr))
	}
	for i, w := range want {
		if arr[i]["level"] != w {
			t.Errorf("entry %d level: got %v, want %s", i, arr[i]["level"], w)
		}
	}
}

func TestCentral_OmitsEmptyOptionalFields(t *testing.T) {
	log, tr, cap := newTestCentral(t, central.Config{})
	log.Info("hello") // no fields, no metadata, no error
	_ = tr.Close()

	arr := decodeFirstBatch(t, cap)
	for _, key := range []string{"instanceId", "tags", "context", "metadata", "error", "groups"} {
		if _, present := arr[0][key]; present {
			t.Errorf("expected %q to be absent when not set, got %v", key, arr[0][key])
		}
	}
}

func TestCentral_ErrorSplitFromContext(t *testing.T) {
	log, tr, cap := newTestCentral(t, central.Config{})
	log = log.WithFields(loglayer.Fields{"requestId": "abc"})
	log.WithError(errors.New("boom")).Error("failed")
	_ = tr.Close()

	obj := decodeFirstBatch(t, cap)[0]

	errObj, ok := obj["error"].(map[string]any)
	if !ok {
		t.Fatalf("error: got %T, want map[string]any", obj["error"])
	}
	if errObj["message"] != "boom" {
		t.Errorf("error.message: got %v", errObj["message"])
	}

	ctx, ok := obj["context"].(map[string]any)
	if !ok {
		t.Fatalf("context: got %T, want map[string]any", obj["context"])
	}
	if ctx["requestId"] != "abc" {
		t.Errorf("context.requestId: got %v", ctx["requestId"])
	}
	if _, leaked := ctx["err"]; leaked {
		t.Errorf("context should not contain err key, got %v", ctx)
	}
}

// When the only thing in Data is the error map, "context" should be omitted
// (not emitted as an empty object).
func TestCentral_ErrorOnly_NoContext(t *testing.T) {
	log, tr, cap := newTestCentral(t, central.Config{})
	log.WithError(errors.New("oops")).Error("failed")
	_ = tr.Close()

	obj := decodeFirstBatch(t, cap)[0]
	if _, ok := obj["context"]; ok {
		t.Errorf("expected context to be absent, got %v", obj["context"])
	}
	if _, ok := obj["error"]; !ok {
		t.Errorf("expected error to be present, got %v", obj)
	}
}

// When the loglayer core is configured with a non-default ErrorFieldName,
// the central transport reads it from Schema and lifts the error out of
// Data correctly.
func TestCentral_CustomErrorFieldName(t *testing.T) {
	cap := &capture{}
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	t.Cleanup(srv.Close)

	tr := central.New(central.Config{
		Service: "svc",
		BaseURL: srv.URL,
		HTTP:    httptr.Config{BatchSize: 10, BatchInterval: time.Hour},
	})
	log := loglayer.New(loglayer.Config{
		Transport:        tr,
		ErrorFieldName:   "error_obj",
		DisableFatalExit: true,
	})
	log.WithError(errors.New("boom")).Error("failed")
	_ = tr.Close()

	obj := decodeFirstBatch(t, cap)[0]
	if _, ok := obj["error"]; !ok {
		t.Errorf("expected error to be present, got %v", obj)
	}
	if ctx, ok := obj["context"].(map[string]any); ok {
		if _, leaked := ctx["error_obj"]; leaked {
			t.Errorf("context should not contain custom error key, got %v", ctx)
		}
	}
}

func TestCentral_GroupsInPayload(t *testing.T) {
	log, tr, cap := newTestCentral(t, central.Config{})
	log.WithGroup("payments", "critical").Info("charged")
	log.Info("plain")
	_ = tr.Close()

	arr := decodeFirstBatch(t, cap)
	if len(arr) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(arr))
	}

	groups, ok := arr[0]["groups"].([]any)
	if !ok {
		t.Fatalf("groups: got %T (%v), want []any", arr[0]["groups"], arr[0]["groups"])
	}
	if len(groups) != 2 || groups[0] != "payments" || groups[1] != "critical" {
		t.Errorf("groups: got %v", groups)
	}

	if _, present := arr[1]["groups"]; present {
		t.Errorf("entry without WithGroup should omit groups, got %v", arr[1]["groups"])
	}
}

func TestCentral_NonMapMetadataPassesThrough(t *testing.T) {
	log, tr, cap := newTestCentral(t, central.Config{})

	type ev struct {
		Op string `json:"op"`
	}
	log.WithMetadata(ev{Op: "load"}).Info("did")
	_ = tr.Close()

	obj := decodeFirstBatch(t, cap)[0]
	meta, ok := obj["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("metadata: got %T (%v), want map[string]any", obj["metadata"], obj["metadata"])
	}
	if meta["op"] != "load" {
		t.Errorf("metadata.op: got %v", meta["op"])
	}
}

// BaseConfig.ID must propagate through the central → http wrap so callers
// can find the transport by ID via loglayer.RemoveTransport / GetLoggerInstance.
func TestCentral_BaseConfigPropagated(t *testing.T) {
	tr := central.New(central.Config{
		BaseConfig: transport.BaseConfig{ID: "central-test"},
		Service:    "svc",
	})
	defer tr.Close()
	if got := tr.ID(); got != "central-test" {
		t.Errorf("ID: got %q, want central-test", got)
	}
}
