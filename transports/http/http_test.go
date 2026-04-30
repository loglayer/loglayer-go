package httptransport_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
	httptr "go.loglayer.dev/transports/http"
)

// captureServer accumulates request bodies for assertion. It supports an
// optional response status to simulate server failures.
type captureServer struct {
	mu       sync.Mutex
	bodies   [][]byte
	headers  []http.Header
	respCode int
	hits     atomic.Int32
}

func newCaptureServer() *captureServer {
	return &captureServer{respCode: http.StatusAccepted}
}

func (s *captureServer) handler(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	s.mu.Lock()
	s.bodies = append(s.bodies, body)
	s.headers = append(s.headers, r.Header.Clone())
	s.mu.Unlock()
	s.hits.Add(1)
	w.WriteHeader(s.respCode)
}

func (s *captureServer) batches() [][]byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([][]byte, len(s.bodies))
	copy(out, s.bodies)
	return out
}

func (s *captureServer) lastHeaders() http.Header {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.headers) == 0 {
		return nil
	}
	return s.headers[len(s.headers)-1]
}

func newLogger(t *testing.T, cfg httptr.Config, srv *httptest.Server) (*loglayer.LogLayer, *httptr.Transport) {
	t.Helper()
	cfg.URL = srv.URL
	if cfg.BaseConfig.ID == "" {
		cfg.BaseConfig.ID = "http"
	}
	tr := httptr.New(cfg)
	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})
	return log, tr
}

func TestHTTP_BasicBatchOnClose(t *testing.T) {
	cap := newCaptureServer()
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	defer srv.Close()

	log, tr := newLogger(t, httptr.Config{
		BatchSize:     10,
		BatchInterval: time.Hour, // never time out during this test
	}, srv)

	for i := 0; i < 3; i++ {
		log.Info("hello", i)
	}
	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	batches := cap.batches()
	if len(batches) != 1 {
		t.Fatalf("expected 1 batch, got %d", len(batches))
	}
	var arr []map[string]any
	if err := json.Unmarshal(batches[0], &arr); err != nil {
		t.Fatalf("body is not valid JSON: %v: got %q", err, batches[0])
	}
	if len(arr) != 3 {
		t.Errorf("expected 3 entries in batch, got %d", len(arr))
	}
	if arr[0]["level"] != "info" {
		t.Errorf("entry level: got %v", arr[0]["level"])
	}
	if arr[0]["msg"] != "hello 0" {
		t.Errorf("entry msg: got %v", arr[0]["msg"])
	}
}

func TestHTTP_FlushOnBatchSize(t *testing.T) {
	cap := newCaptureServer()
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	defer srv.Close()

	log, tr := newLogger(t, httptr.Config{
		BatchSize:     2, // small batch to force two flushes
		BatchInterval: time.Hour,
	}, srv)

	log.Info("a")
	log.Info("b")
	log.Info("c")
	log.Info("d")
	log.Info("e")
	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	batches := cap.batches()
	if len(batches) < 2 {
		t.Fatalf("expected at least 2 batches, got %d", len(batches))
	}
}

func TestHTTP_FlushOnInterval(t *testing.T) {
	cap := newCaptureServer()
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	defer srv.Close()

	log, tr := newLogger(t, httptr.Config{
		BatchSize:     100,
		BatchInterval: 50 * time.Millisecond,
	}, srv)
	defer tr.Close()

	log.Info("first")

	// Wait long enough for the interval timer to fire.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && cap.hits.Load() == 0 {
		time.Sleep(10 * time.Millisecond)
	}
	if cap.hits.Load() == 0 {
		t.Fatal("expected interval-driven flush, got none")
	}
}

func TestHTTP_HeadersAndContentType(t *testing.T) {
	cap := newCaptureServer()
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	defer srv.Close()

	log, tr := newLogger(t, httptr.Config{
		BatchSize:     10,
		BatchInterval: time.Hour,
		Headers: map[string]string{
			"X-Trace": "abc",
		},
	}, srv)

	log.Info("hello")
	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	hdrs := cap.lastHeaders()
	if hdrs.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type: got %q", hdrs.Get("Content-Type"))
	}
	if hdrs.Get("X-Trace") != "abc" {
		t.Errorf("X-Trace: got %q", hdrs.Get("X-Trace"))
	}
}

func TestHTTP_OnError_HTTPStatus(t *testing.T) {
	cap := newCaptureServer()
	cap.respCode = http.StatusInternalServerError
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	defer srv.Close()

	var captured atomic.Pointer[error]
	log, tr := newLogger(t, httptr.Config{
		BatchSize:     1,
		BatchInterval: time.Hour,
		OnError: func(err error, entries []httptr.Entry) {
			captured.Store(&err)
		},
	}, srv)

	log.Info("trigger")
	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	got := captured.Load()
	if got == nil {
		t.Fatal("expected OnError to fire")
	}
	var httpErr *httptr.HTTPError
	if !errors.As(*got, &httpErr) {
		t.Fatalf("expected *HTTPError, got %T: %v", *got, *got)
	}
	if httpErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("status: got %d", httpErr.StatusCode)
	}
}

func TestHTTP_OnError_BufferFull(t *testing.T) {
	// Server that blocks long enough to back up the worker. The client uses
	// a short timeout so the worker eventually unblocks and Close can return.
	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-block
	}))

	dropped := atomic.Int32{}
	log, tr := newLogger(t, httptr.Config{
		BatchSize:     1,
		BatchInterval: time.Hour,
		BufferSize:    1,
		Client:        &http.Client{Timeout: 200 * time.Millisecond},
		OnError: func(err error, entries []httptr.Entry) {
			if errors.Is(err, httptr.ErrBufferFull) {
				dropped.Add(1)
			}
		},
	}, srv)

	// Send a flood; the server hangs so the worker stalls and the queue fills.
	for i := 0; i < 100; i++ {
		log.Info("flood", i)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && dropped.Load() == 0 {
		time.Sleep(10 * time.Millisecond)
	}
	if dropped.Load() == 0 {
		t.Errorf("expected at least one dropped entry; got %d", dropped.Load())
	}

	// Unblock the server, then tear down in order: close the transport first
	// (its short client timeout already returned), then the server.
	close(block)
	_ = tr.Close()
	srv.Close()
}

func TestHTTP_OnError_AfterClose(t *testing.T) {
	cap := newCaptureServer()
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	defer srv.Close()

	closedCalls := atomic.Int32{}
	log, tr := newLogger(t, httptr.Config{
		BatchSize:     10,
		BatchInterval: time.Hour,
		OnError: func(err error, entries []httptr.Entry) {
			if errors.Is(err, httptr.ErrClosed) {
				closedCalls.Add(1)
			}
		},
	}, srv)

	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	log.Info("dropped post-close")
	if closedCalls.Load() != 1 {
		t.Errorf("expected 1 ErrClosed callback, got %d", closedCalls.Load())
	}
}

func TestHTTP_LevelFiltering(t *testing.T) {
	cap := newCaptureServer()
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	defer srv.Close()

	log, tr := newLogger(t, httptr.Config{
		BaseConfig:    transport.BaseConfig{ID: "http", Level: loglayer.LogLevelError},
		BatchSize:     10,
		BatchInterval: time.Hour,
	}, srv)

	log.Info("dropped")
	log.Warn("dropped")
	log.Error("kept")
	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	batches := cap.batches()
	if len(batches) != 1 {
		t.Fatalf("expected 1 batch, got %d", len(batches))
	}
	var arr []map[string]any
	_ = json.Unmarshal(batches[0], &arr)
	if len(arr) != 1 {
		t.Fatalf("expected 1 entry after level filter, got %d", len(arr))
	}
	if arr[0]["level"] != "error" {
		t.Errorf("kept entry level: got %v", arr[0]["level"])
	}
}

func TestHTTP_FieldsAndMetadataInBody(t *testing.T) {
	cap := newCaptureServer()
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	defer srv.Close()

	log, tr := newLogger(t, httptr.Config{
		BatchSize:     10,
		BatchInterval: time.Hour,
	}, srv)

	log = log.WithFields(loglayer.Fields{"requestId": "abc"})
	log.WithMetadata(loglayer.Metadata{"durationMs": 42}).Info("served")
	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	batches := cap.batches()
	if len(batches) != 1 {
		t.Fatalf("expected 1 batch, got %d", len(batches))
	}
	var arr []map[string]any
	_ = json.Unmarshal(batches[0], &arr)
	entry := arr[0]
	if entry["requestId"] != "abc" {
		t.Errorf("requestId: got %v", entry["requestId"])
	}
	if entry["durationMs"] != float64(42) {
		t.Errorf("durationMs: got %v", entry["durationMs"])
	}
}

func TestHTTP_MetadataNonMapNestedUnderKey(t *testing.T) {
	cap := newCaptureServer()
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	defer srv.Close()

	log, tr := newLogger(t, httptr.Config{
		BatchSize:     10,
		BatchInterval: time.Hour,
	}, srv)

	type ev struct {
		Op string `json:"op"`
	}
	log.WithMetadata(ev{Op: "load"}).Info("did")
	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	batches := cap.batches()
	if len(batches) != 1 {
		t.Fatalf("expected 1 batch, got %d", len(batches))
	}
	var arr []map[string]any
	_ = json.Unmarshal(batches[0], &arr)
	meta, ok := arr[0]["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("expected nested metadata object, got %T: %v", arr[0]["metadata"], arr[0]["metadata"])
	}
	if meta["op"] != "load" {
		t.Errorf("metadata.op: got %v", meta["op"])
	}
}

func TestHTTP_CloseIsIdempotent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	_, tr := newLogger(t, httptr.Config{}, srv)
	if err := tr.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := tr.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestHTTP_CustomEncoder(t *testing.T) {
	cap := newCaptureServer()
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	defer srv.Close()

	log, tr := newLogger(t, httptr.Config{
		BatchSize:     10,
		BatchInterval: time.Hour,
		Encoder: httptr.EncoderFunc(func(entries []httptr.Entry) ([]byte, string, error) {
			lines := make([]string, len(entries))
			for i, e := range entries {
				lines[i] = e.Level.String() + ":" + transport.JoinMessages(e.Messages)
			}
			return []byte(strings.Join(lines, "\n")), "text/plain", nil
		}),
	}, srv)

	log.Info("hi")
	log.Warn("watch")
	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	hdrs := cap.lastHeaders()
	if hdrs.Get("Content-Type") != "text/plain" {
		t.Errorf("Content-Type: got %q", hdrs.Get("Content-Type"))
	}
	body := string(cap.batches()[0])
	if !strings.Contains(body, "info:hi") || !strings.Contains(body, "warn:watch") {
		t.Errorf("body: %q", body)
	}
}

func TestHTTP_New_PanicsWithoutURL(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic when URL missing")
		}
		err, ok := r.(error)
		if !ok || !errors.Is(err, httptr.ErrURLRequired) {
			t.Errorf("panic value: got %v, want ErrURLRequired", r)
		}
	}()
	_ = httptr.New(httptr.Config{})
}

func TestHTTP_Build_ReturnsErrURLRequired(t *testing.T) {
	_, err := httptr.Build(httptr.Config{})
	if !errors.Is(err, httptr.ErrURLRequired) {
		t.Errorf("Build with missing URL: got %v, want ErrURLRequired", err)
	}
}

// A panicking user Encoder must not kill the worker. The panic is surfaced
// via OnError, and a subsequent log call still drains through a working
// (non-panicking) Encoder.
func TestHTTP_PanickingEncoder_RecoveredAndReportedViaOnError(t *testing.T) {
	cap := newCaptureServer()
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	defer srv.Close()

	var encodeCalls atomic.Int32
	errs := make(chan error, 4)

	encoder := httptr.EncoderFunc(func(entries []httptr.Entry) ([]byte, string, error) {
		n := encodeCalls.Add(1)
		if n == 1 {
			panic("encoder boom")
		}
		return httptr.JSONArrayEncoder.Encode(entries)
	})

	log, tr := newLogger(t, httptr.Config{
		Encoder:       encoder,
		BatchSize:     1,
		BatchInterval: 50 * time.Millisecond,
		OnError: func(err error, _ []httptr.Entry) {
			errs <- err
		},
	}, srv)

	log.Info("first")  // panics inside Encoder
	log.Info("second") // worker still alive, succeeds

	select {
	case err := <-errs:
		if !strings.Contains(err.Error(), "panic during flush") {
			t.Errorf("expected panic-recovery error, got: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("OnError was not invoked within 2s")
	}

	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if got := len(cap.batches()); got != 1 {
		t.Errorf("expected exactly 1 successful batch (the second log) after recovered panic, got %d", got)
	}
}

// A panicking user OnError must not kill the worker either. We trigger an
// HTTP error (status 500), the user OnError handler panics, and a follow-up
// successful entry must still be delivered.
func TestHTTP_PanickingOnError_DoesNotKillWorker(t *testing.T) {
	cap := newCaptureServer()
	cap.respCode = http.StatusInternalServerError
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	defer srv.Close()

	var onErrCalls atomic.Int32
	log, tr := newLogger(t, httptr.Config{
		BatchSize:     1,
		BatchInterval: 50 * time.Millisecond,
		OnError: func(err error, _ []httptr.Entry) {
			onErrCalls.Add(1)
			panic("onError boom")
		},
	}, srv)

	log.Info("first")  // server 500 → OnError → panics → recovered
	log.Info("second") // worker still alive, server still 500, OnError panics again

	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Worker should have processed both entries (so OnError was invoked
	// twice). Without panic recovery, the worker would have died after
	// the first call and the second batch would never reach the server.
	if got := cap.hits.Load(); got < 2 {
		t.Errorf("expected at least 2 server hits after recovered OnError panics, got %d", got)
	}
}

// The default Client refuses cross-host redirects so configured credentials
// (Authorization, X-API-Key, etc.) cannot be forwarded to a redirected host.
// Two servers run on different ports; the first 302s to the second. A header
// the user supplied for the original host must surface as an OnError
// (cross-host redirect refused), and the second host must never see it.
func TestHTTP_DefaultClient_RefusesCrossHostRedirect(t *testing.T) {
	receivedAuth := make(chan string, 1)
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case receivedAuth <- r.Header.Get("X-API-Key"):
		default:
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer target.Close()

	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer redirector.Close()

	errs := make(chan error, 2)
	tr := httptr.New(httptr.Config{
		URL:           redirector.URL,
		Headers:       map[string]string{"X-API-Key": "secret-token"},
		BatchSize:     1,
		BatchInterval: 50 * time.Millisecond,
		OnError: func(err error, _ []httptr.Entry) {
			errs <- err
		},
	})
	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})
	log.Info("trigger send")

	select {
	case err := <-errs:
		if !strings.Contains(err.Error(), "cross-host redirect") {
			t.Errorf("expected cross-host redirect error, got: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("OnError was not invoked within 2s")
	}

	_ = tr.Close()

	// If the redirect had been followed, target's handler would have
	// captured the X-API-Key. It must not have been called at all.
	select {
	case got := <-receivedAuth:
		t.Errorf("X-API-Key leaked to redirected host: %q", got)
	default:
	}
}

// Entry.Groups round-trips from loglayer.TransportParams.Groups so wrapper
// transports (datadog, central, etc.) can ship groups in the wire payload.
func TestHTTP_EntryCarriesGroups(t *testing.T) {
	cap := newCaptureServer()
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	defer srv.Close()

	var captured [][]string
	var mu sync.Mutex
	encoder := httptr.EncoderFunc(func(entries []httptr.Entry) ([]byte, string, error) {
		mu.Lock()
		for _, e := range entries {
			captured = append(captured, append([]string(nil), e.Groups...))
		}
		mu.Unlock()
		return []byte("[]"), "application/json", nil
	})

	log, tr := newLogger(t, httptr.Config{
		BatchSize:     10,
		BatchInterval: time.Hour,
		Encoder:       encoder,
	}, srv)
	log.WithGroup("payments", "critical").Info("charged")
	log.Info("plain")
	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(captured) != 2 {
		t.Fatalf("expected 2 captured entries, got %d", len(captured))
	}
	want := []string{"payments", "critical"}
	if !slices.Equal(captured[0], want) {
		t.Errorf("entry 0 Groups: got %v, want %v", captured[0], want)
	}
	if len(captured[1]) != 0 {
		t.Errorf("entry 1 Groups: got %v, want nil", captured[1])
	}
}
