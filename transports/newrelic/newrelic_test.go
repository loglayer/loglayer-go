package newrelic_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	httptr "go.loglayer.dev/transports/http/v2"
	"go.loglayer.dev/transports/newrelic/v2"
	"go.loglayer.dev/v2"
	"go.loglayer.dev/v2/transport"
)

func newCapture() *capture { return &capture{} }

type capture struct {
	mu      sync.Mutex
	bodies  [][]byte
	headers []http.Header
	hits    int
}

func (c *capture) handler(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	c.mu.Lock()
	c.bodies = append(c.bodies, body)
	c.headers = append(c.headers, r.Header.Clone())
	c.hits++
	c.mu.Unlock()
	w.WriteHeader(http.StatusAccepted)
}

func (c *capture) lastBody() []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.bodies) == 0 {
		return nil
	}
	return c.bodies[len(c.bodies)-1]
}

func (c *capture) lastHeaders() http.Header {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.headers) == 0 {
		return nil
	}
	return c.headers[len(c.headers)-1]
}

// testCfg returns a Config wired against httptest server with
// AllowInsecureURL (since httptest uses http://).
func testCfg(srv *httptest.Server, fields ...func(*newrelic.Config)) newrelic.Config {
	cfg := newrelic.Config{
		APIKey: "k",
		URL:    srv.URL,
		HTTP: httptr.Config{
			BatchSize:     10,
			BatchInterval: time.Hour,
		},
		AllowInsecureURL: true,
	}
	for _, f := range fields {
		f(&cfg)
	}
	return cfg
}

func TestNewRelic_BasicSend(t *testing.T) {
	cap := newCapture()
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	defer srv.Close()

	tr := newrelic.New(testCfg(srv))
	defer tr.Close()

	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})
	log.Info("hello from newrelic")

	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if cap.hits != 1 {
		t.Fatalf("expected 1 request, got %d", cap.hits)
	}

	hdrs := cap.lastHeaders()
	if got := hdrs.Get("Api-Key"); got != "k" {
		t.Errorf("Api-Key header: got %q, want %q", got, "k")
	}
	if got := hdrs.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type: got %q, want %q", got, "application/json")
	}

	// Parse NDJSON body
	body := cap.lastBody()
	lines := bytes.Split(body, []byte{'\n'})
	if len(lines) != 1 {
		t.Fatalf("expected 1 NDJSON line, got %d", len(lines))
	}
	var obj map[string]any
	if err := json.Unmarshal(lines[0], &obj); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if obj["message"] != "hello from newrelic" {
		t.Errorf("message: got %v", obj["message"])
	}
	if obj["loglevel"] != "info" {
		t.Errorf("loglevel: got %v", obj["loglevel"])
	}
	if _, hasTimestamp := obj["timestamp"]; !hasTimestamp {
		t.Error("missing timestamp field")
	}
}

func TestNewRelic_LevelMapping(t *testing.T) {
	cap := newCapture()
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	defer srv.Close()

	tr := newrelic.New(testCfg(srv))

	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})

	log.Trace("t")
	log.Debug("d")
	log.Info("i")
	log.Warn("w")
	log.Error("e")
	func() {
		defer func() { _ = recover() }()
		log.Fatal("f")
	}()
	func() {
		defer func() { _ = recover() }()
		log.Panic("p")
	}()
	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	body := cap.lastBody()
	lines := bytes.Split(body, []byte{'\n'})
	if len(lines) != 7 {
		t.Fatalf("expected 7 lines, got %d: %q", len(lines), body)
	}

	wantLevels := []string{"debug", "debug", "info", "warning", "error", "critical", "critical"}
	for i, want := range wantLevels {
		var obj map[string]any
		json.Unmarshal(lines[i], &obj)
		if obj["loglevel"] != want {
			t.Errorf("line %d loglevel: got %v, want %s", i, obj["loglevel"], want)
		}
	}
}

func TestNewRelic_HostnameInBody(t *testing.T) {
	cap := newCapture()
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	defer srv.Close()

	tr := newrelic.New(testCfg(srv, func(cfg *newrelic.Config) {
		cfg.Hostname = "prod-web-01"
	}))
	defer tr.Close()

	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})
	log.Info("hello")

	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	var obj map[string]any
	json.Unmarshal(cap.lastBody(), &obj)
	if obj["hostname"] != "prod-web-01" {
		t.Errorf("hostname: got %v, want prod-web-01", obj["hostname"])
	}
}

func TestNewRelic_FieldsAndMetadata(t *testing.T) {
	cap := newCapture()
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	defer srv.Close()

	tr := newrelic.New(testCfg(srv))
	defer tr.Close()

	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})
	log = log.WithFields(loglayer.Fields{"requestId": "abc-123"})
	log.WithMetadata(loglayer.Metadata{"durationMs": 42}).Info("served")

	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	var obj map[string]any
	json.Unmarshal(cap.lastBody(), &obj)
	if obj["requestId"] != "abc-123" {
		t.Errorf("requestId: got %v", obj["requestId"])
	}
	if obj["durationMs"] != float64(42) {
		t.Errorf("durationMs: got %v", obj["durationMs"])
	}
}

func TestNewRelic_LicenseKeyHeader(t *testing.T) {
	cap := newCapture()
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	defer srv.Close()

	tr := newrelic.New(testCfg(srv, func(cfg *newrelic.Config) {
		cfg.LicenseKey = "license-key"
	}))
	defer tr.Close()

	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})
	log.Info("hello")

	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	hdrs := cap.lastHeaders()
	if got := hdrs.Get("X-License-Key"); got != "license-key" {
		t.Errorf("X-License-Key: got %q, want license-key", got)
	}
}

func TestNewRelic_BatchingBySize(t *testing.T) {
	cap := newCapture()
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	defer srv.Close()

	tr := newrelic.New(testCfg(srv, func(cfg *newrelic.Config) {
		cfg.HTTP.BatchSize = 3
	}))
	defer tr.Close()

	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})
	log.Info("a")
	log.Info("b")
	log.Info("c")
	log.Info("d")
	log.Info("e")

	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// With batch size 3: first 3 flush on size, last 2 flush on Close
	if cap.hits < 2 {
		t.Fatalf("expected at least 2 batches, got %d", cap.hits)
	}

	cap.mu.Lock()
	defer cap.mu.Unlock()

	lines := bytes.Split(cap.bodies[0], []byte{'\n'})
	if len(lines) != 3 {
		t.Errorf("batch 0 lines: got %d, want 3", len(lines))
	}

	lastLines := bytes.Split(cap.bodies[cap.hits-1], []byte{'\n'})
	if len(lastLines) != 2 {
		t.Errorf("final batch lines: got %d, want 2", len(lastLines))
	}
}

func TestNewRelic_TimestampFormat(t *testing.T) {
	cap := newCapture()
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	defer srv.Close()

	tr := newrelic.New(testCfg(srv))
	defer tr.Close()

	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})
	log.Info("time check")

	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	var obj map[string]any
	json.Unmarshal(cap.lastBody(), &obj)
	ts, ok := obj["timestamp"].(string)
	if !ok {
		t.Fatalf("timestamp not a string: %v", obj["timestamp"])
	}
	_, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		t.Errorf("timestamp %q is not valid RFC3339: %v", ts, err)
	}
	if !strings.HasSuffix(ts, "Z") {
		t.Errorf("timestamp should end with Z (UTC): %q", ts)
	}
}

func TestNewRelic_LevelFiltering(t *testing.T) {
	cap := newCapture()
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	defer srv.Close()

	tr := newrelic.New(testCfg(srv, func(cfg *newrelic.Config) {
		cfg.BaseConfig = transport.BaseConfig{ID: "newrelic", Level: loglayer.LogLevelError}
	}))
	defer tr.Close()

	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})
	log.Info("dropped")
	log.Warn("dropped")
	log.Error("kept")

	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if cap.hits != 1 {
		t.Fatalf("expected 1 request (error only), got %d", cap.hits)
	}
	var obj map[string]any
	json.Unmarshal(cap.lastBody(), &obj)
	if obj["loglevel"] != "error" {
		t.Errorf("loglevel: got %v, want error", obj["loglevel"])
	}
}

func TestNewRelic_NonMapMetadataNestedUnderKey(t *testing.T) {
	cap := newCapture()
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	defer srv.Close()

	tr := newrelic.New(testCfg(srv))

	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})

	type ev struct {
		Op string `json:"op"`
	}
	log.WithMetadata(ev{Op: "load"}).Info("did")
	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	var obj map[string]any
	json.Unmarshal(cap.lastBody(), &obj)
	meta, ok := obj["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("expected nested metadata object, got %T: %v", obj["metadata"], obj["metadata"])
	}
	if meta["op"] != "load" {
		t.Errorf("metadata.op: got %v", meta["op"])
	}
}

func TestNewRelic_CustomHeaders(t *testing.T) {
	cap := newCapture()
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	defer srv.Close()

	tr := newrelic.New(testCfg(srv, func(cfg *newrelic.Config) {
		cfg.HTTP.Headers = map[string]string{"X-Custom-Header": "custom-value"}
	}))
	defer tr.Close()

	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})
	log.Info("hello")

	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	hdrs := cap.lastHeaders()
	if got := hdrs.Get("X-Custom-Header"); got != "custom-value" {
		t.Errorf("X-Custom-Header: got %q, want custom-value", got)
	}
	if got := hdrs.Get("Api-Key"); got != "k" {
		t.Errorf("Api-Key: got %q, want k", got)
	}
}

func TestNewRelic_ZoneURLs(t *testing.T) {
	if got := newrelic.IntakeZone("US").URL(); got != "https://log-api.newrelic.com/log/v1" {
		t.Errorf("US URL: got %q", got)
	}
	if got := newrelic.IntakeZone("EU").URL(); got != "https://log-api.eu.newrelic.com/log/v1" {
		t.Errorf("EU URL: got %q", got)
	}
	if got := newrelic.IntakeZone("").URL(); got != "https://log-api.newrelic.com/log/v1" {
		t.Errorf("empty zone (default US): got %q", got)
	}
}

func TestNewRelic_URLOverride(t *testing.T) {
	cap := newCapture()
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	defer srv.Close()

	tr := newrelic.New(testCfg(srv))
	defer tr.Close()

	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})
	log.Info("to override")

	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if cap.hits != 1 {
		t.Fatalf("expected 1 request to override URL, got %d", cap.hits)
	}
}

func TestNewRelic_ConfigStringRedactsKeys(t *testing.T) {
	cfg := newrelic.Config{
		APIKey:     "deadbeef-secret",
		LicenseKey: "license-secret",
		Hostname:   "myhost",
	}

	s := cfg.String()
	if strings.Contains(s, "deadbeef-secret") {
		t.Errorf("APIKey leaked through String(): %s", s)
	}
	if strings.Contains(s, "license-secret") {
		t.Errorf("LicenseKey leaked through String(): %s", s)
	}
	if !strings.Contains(s, "redacted") {
		t.Errorf("String() should mark values as redacted: %s", s)
	}
	if !strings.Contains(s, "myhost") {
		t.Errorf("hostname should be visible: %s", s)
	}
}

func TestNewRelic_ConfigKeysTaggedJSONIgnore(t *testing.T) {
	typ := reflect.TypeOf(newrelic.Config{})
	for _, name := range []string{"APIKey", "LicenseKey"} {
		field, ok := typ.FieldByName(name)
		if !ok {
			t.Fatalf("Config.%s field not found", name)
		}
		got := field.Tag.Get("json")
		if got != "-" {
			t.Errorf("Config.%s json tag: got %q, want \"-\"", name, got)
		}
	}
}

func TestNewRelic_Build_ErrAPIKeyRequired(t *testing.T) {
	_, err := newrelic.Build(newrelic.Config{})
	if !errors.Is(err, newrelic.ErrAPIKeyRequired) {
		t.Errorf("Build with empty APIKey: got %v, want ErrAPIKeyRequired", err)
	}
}

func TestNewRelic_New_PanicsWithoutAPIKey(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic when APIKey missing")
		}
		err, ok := r.(error)
		if !ok || !errors.Is(err, newrelic.ErrAPIKeyRequired) {
			t.Errorf("panic value: got %v, want ErrAPIKeyRequired", r)
		}
	}()
	_ = newrelic.New(newrelic.Config{})
}

func TestNewRelic_Build_ErrHTTPOverrideForbidden(t *testing.T) {
	_, err := newrelic.Build(newrelic.Config{
		APIKey: "k",
		HTTP: httptr.Config{
			URL: "https://example.com",
		},
	})
	if !errors.Is(err, newrelic.ErrHTTPOverrideForbidden) {
		t.Errorf("Build with HTTP.URL set: got %v, want ErrHTTPOverrideForbidden", err)
	}

	enc := httptr.EncoderFunc(func(_ []httptr.Entry) ([]byte, string, error) {
		return nil, "", nil
	})
	_, err = newrelic.Build(newrelic.Config{
		APIKey: "k",
		HTTP: httptr.Config{
			Encoder: enc,
		},
	})
	if !errors.Is(err, newrelic.ErrHTTPOverrideForbidden) {
		t.Errorf("Build with HTTP.Encoder set: got %v, want ErrHTTPOverrideForbidden", err)
	}
}

func TestNewRelic_Build_RejectsInsecureURL(t *testing.T) {
	_, err := newrelic.Build(newrelic.Config{
		APIKey: "k",
		URL:    "http://example.com",
	})
	if !errors.Is(err, newrelic.ErrInsecureURL) {
		t.Errorf("Build with http URL: got %v, want ErrInsecureURL", err)
	}
}

func TestNewRelic_Build_AllowsInsecureURLWithOptIn(t *testing.T) {
	tr, err := newrelic.Build(newrelic.Config{
		APIKey:           "k",
		URL:              "http://example.com",
		AllowInsecureURL: true,
	})
	if err != nil {
		t.Fatalf("AllowInsecureURL=true should pass: %v", err)
	}
	_ = tr.Close()
}

func TestNewRelic_ConfigString(t *testing.T) {
	cfg := newrelic.Config{
		APIKey:     "secret-api-key",
		LicenseKey: "secret-license-key",
		Hostname:   "web-prod-1",
	}

	s := fmt.Sprintf("%v", cfg)
	if strings.Contains(s, "secret-api-key") {
		t.Errorf("APIKey leaked via %%v: %s", s)
	}
	if strings.Contains(s, "secret-license-key") {
		t.Errorf("LicenseKey leaked via %%v: %s", s)
	}
}
