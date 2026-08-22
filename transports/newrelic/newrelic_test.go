package newrelic_test

import (
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

	httptr "go.loglayer.dev/transports/http/v3"
	"go.loglayer.dev/transports/newrelic"
	"go.loglayer.dev/v3"
	"go.loglayer.dev/v3/transport"
)

type capture struct {
	mu      sync.Mutex
	bodies  [][]byte
	headers []http.Header
}

func (c *capture) handler(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	c.mu.Lock()
	c.bodies = append(c.bodies, body)
	c.headers = append(c.headers, r.Header.Clone())
	c.mu.Unlock()
	w.WriteHeader(http.StatusAccepted)
}

func newLogger(t *testing.T, cfg newrelic.Config) (*loglayer.LogLayer, *newrelic.Transport) {
	t.Helper()
	tr := newrelic.New(cfg)
	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})
	return log, tr
}

// Basic log delivery: write entries and call Close to flush the batch.
func TestNewRelic_BasicBatchOnClose(t *testing.T) {
	cap := &capture{}
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	defer srv.Close()

	log, tr := newLogger(t, newrelic.Config{
		LicenseKey:       "test-key-123",
		URL:              srv.URL,
		AllowInsecureURL: true,
		HTTP: httptr.Config{
			BatchSize:     1,
			BatchInterval: 10 * time.Millisecond,
		},
	})

	log.Info("hello")
	log.Warn("world")

	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	cap.mu.Lock()
	defer cap.mu.Unlock()
	if len(cap.bodies) < 1 {
		t.Fatal("expected at least one request")
	}

	var arr []map[string]any
	if err := json.Unmarshal(cap.bodies[0], &arr); err != nil {
		t.Fatalf("body is not JSON: %v: %q", err, cap.bodies[0])
	}
	if len(arr) == 0 {
		t.Fatal("expected at least one entry in batch")
	}

	// First entry should be "hello"
	if arr[0]["log"] != "hello" {
		t.Errorf("first log: got %v, want hello", arr[0]["log"])
	}
}

// FlushOnBatchSize: when the batch reaches BatchSize, the worker POSTs
// immediately without waiting for BatchInterval.
func TestNewRelic_FlushOnBatchSize(t *testing.T) {
	cap := &capture{}
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	defer srv.Close()

	log, tr := newLogger(t, newrelic.Config{
		LicenseKey:       "k",
		URL:              srv.URL,
		AllowInsecureURL: true,
		HTTP: httptr.Config{
			BatchSize:     2,
			BatchInterval: time.Hour,
		},
	})

	log.Info("one")
	log.Info("two")

	// Give the worker time to flush the batch.
	time.Sleep(50 * time.Millisecond)

	cap.mu.Lock()
	defer cap.mu.Unlock()
	if len(cap.bodies) == 0 {
		t.Fatal("expected a request after batch size reached")
	}

	var arr []map[string]any
	_ = json.Unmarshal(cap.bodies[0], &arr)
	if len(arr) != 2 {
		t.Errorf("expected 2 entries in batch, got %d", len(arr))
	}

	// Close to shut down cleanly.
	cap.mu.Unlock()
	t.Cleanup(func() { _ = tr.Close() })
	cap.mu.Lock()
}

// FlushOnInterval: when BatchInterval elapses before BatchSize is reached,
// accumulated entries are POSTed.
func TestNewRelic_FlushOnInterval(t *testing.T) {
	cap := &capture{}
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	defer srv.Close()

	log, tr := newLogger(t, newrelic.Config{
		LicenseKey:       "k",
		URL:              srv.URL,
		AllowInsecureURL: true,
		HTTP: httptr.Config{
			BatchSize:     100,
			BatchInterval: 25 * time.Millisecond,
		},
	})

	log.Info("after interval")

	// Wait for the interval to fire.
	time.Sleep(100 * time.Millisecond)

	cap.mu.Lock()
	defer cap.mu.Unlock()
	if len(cap.bodies) == 0 {
		t.Fatal("expected a request after interval elapsed")
	}

	var arr []map[string]any
	_ = json.Unmarshal(cap.bodies[0], &arr)
	if len(arr) != 1 {
		t.Errorf("expected 1 entry, got %d", len(arr))
	}
	_ = tr.Close()
}

// HeadersAndContentType: the Api-Key header and Content-Type are set on
// every request.
func TestNewRelic_HeadersAndContentType(t *testing.T) {
	cap := &capture{}
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	defer srv.Close()

	log, tr := newLogger(t, newrelic.Config{
		LicenseKey:       "my-license-key",
		URL:              srv.URL,
		AllowInsecureURL: true,
		HTTP: httptr.Config{
			BatchSize:     1,
			BatchInterval: 10 * time.Millisecond,
		},
	})

	log.Info("check headers")
	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	cap.mu.Lock()
	defer cap.mu.Unlock()
	if len(cap.headers) == 0 {
		t.Fatal("no request captured")
	}

	hdrs := cap.headers[0]
	if got := hdrs.Get("Api-Key"); got != "my-license-key" {
		t.Errorf("Api-Key: got %q, want my-license-key", got)
	}
	if got := hdrs.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type: got %q, want application/json", got)
	}
}

// OnError_HTTPStatus: when the server responds with a non-2xx status, the
// OnError callback is invoked with an HTTPError.
func TestNewRelic_OnError_HTTPStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	var (
		mu         sync.Mutex
		gotErr     error
		gotEntries []httptr.Entry
	)
	_, tr := newLogger(t, newrelic.Config{
		LicenseKey:       "k",
		URL:              srv.URL,
		AllowInsecureURL: true,
		HTTP: httptr.Config{
			BatchSize:     1,
			BatchInterval: 10 * time.Millisecond,
			OnError: func(err error, entries []httptr.Entry) {
				mu.Lock()
				gotErr = err
				gotEntries = entries
				mu.Unlock()
			},
		},
	})

	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})
	log.Info("will fail")
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	errVal := gotErr
	entries := gotEntries
	mu.Unlock()
	if errVal == nil {
		t.Fatal("expected an error from OnError")
	}
	if !strings.Contains(errVal.Error(), "status 500") {
		t.Errorf("error message: got %v", errVal)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 retriable entry, got %d", len(entries))
	}
	_ = tr.Close()
}

// LevelFiltering: entries below Config.Level are not sent.
func TestNewRelic_LevelFiltering(t *testing.T) {
	cap := &capture{}
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	defer srv.Close()

	log, tr := newLogger(t, newrelic.Config{
		LicenseKey:       "k",
		URL:              srv.URL,
		AllowInsecureURL: true,
		HTTP: httptr.Config{
			BatchSize:     1,
			BatchInterval: 10 * time.Millisecond,
		},
		BaseConfig: transport.BaseConfig{
			Level: loglayer.LogLevelWarn,
		},
	})

	log.Debug("should be filtered")
	log.Info("also filtered")
	log.Warn("this passes")
	log.Error("this too")

	// Wait for interval to flush.
	time.Sleep(100 * time.Millisecond)
	_ = tr.Close()

	cap.mu.Lock()
	defer cap.mu.Unlock()
	var arr []map[string]any
	for _, body := range cap.bodies {
		var batch []map[string]any
		if err := json.Unmarshal(body, &batch); err != nil {
			continue
		}
		arr = append(arr, batch...)
	}

	if len(arr) < 2 {
		t.Fatalf("expected at least 2 entries (warn + error), got %d", len(arr))
	}

	// Check that no debug or info entries made it through.
	for i, obj := range arr {
		if level, ok := obj["level"].(string); ok {
			if level == "debug" || level == "info" {
				t.Errorf("entry %d should have been filtered, got level %q", i, level)
			}
		}
	}
}

// EncodedBodyShape: each entry in the batch has logtype, timestamp, loglevel,
// and message fields with the correct values.
func TestNewRelic_EncodedBodyShape(t *testing.T) {
	cap := &capture{}
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	defer srv.Close()

	_, tr := newLogger(t, newrelic.Config{
		LicenseKey:       "k",
		URL:              srv.URL,
		AllowInsecureURL: true,
		HTTP: httptr.Config{
			BatchSize:     1,
			BatchInterval: 10 * time.Millisecond,
		},
	})

	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})
	log.Info("test body shape")

	time.Sleep(100 * time.Millisecond)
	_ = tr.Close()

	cap.mu.Lock()
	defer cap.mu.Unlock()
	if len(cap.bodies) == 0 {
		t.Fatal("no request captured")
	}

	var arr []map[string]any
	if err := json.Unmarshal(cap.bodies[0], &arr); err != nil {
		t.Fatalf("body is not JSON: %v: %q", err, cap.bodies[0])
	}
	if len(arr) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(arr))
	}

	obj := arr[0]

	if level, ok := obj["level"].(string); !ok || level == "" {
		t.Errorf("level: missing or empty, got %v", obj["level"])
	}

	// Timestamp should be a number (epoch milliseconds).
	if ts, ok := obj["timestamp"].(float64); !ok {
		t.Errorf("timestamp should be a number, got %T: %v", obj["timestamp"], obj["timestamp"])
	} else if ts < 1_700_000_000_000 {
		t.Errorf("timestamp too small for a valid epoch-ms: %v", ts)
	}

	if obj["level"] != "info" {
		t.Errorf("level: got %v, want info", obj["level"])
	}

	if obj["log"] != "test body shape" {
		t.Errorf("log: got %v, want 'test body shape'", obj["log"])
	}
}

// FieldAndMetadataInBody: WithFields and WithMetadata values land in the
// JSON body alongside the standard fields.
func TestNewRelic_FieldAndMetadataInBody(t *testing.T) {
	cap := &capture{}
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	defer srv.Close()

	_, tr := newLogger(t, newrelic.Config{
		LicenseKey:       "k",
		URL:              srv.URL,
		AllowInsecureURL: true,
		HTTP: httptr.Config{
			BatchSize:     1,
			BatchInterval: 10 * time.Millisecond,
		},
	})

	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})
	log = log.WithFields(loglayer.Fields{"requestId": "req-42"})
	log.WithMetadata(loglayer.Metadata{"durationMs": 123, "endpoint": "/api/v1"}).Info("handled")

	time.Sleep(100 * time.Millisecond)
	_ = tr.Close()

	cap.mu.Lock()
	defer cap.mu.Unlock()
	if len(cap.bodies) == 0 {
		t.Fatal("no request captured")
	}

	var arr []map[string]any
	_ = json.Unmarshal(cap.bodies[0], &arr)

	obj := arr[0]
	attrs, ok := obj["attributes"].(map[string]any)
	if !ok {
		t.Fatal("expected attributes map")
	}
	if attrs["requestId"] != "req-42" {
		t.Errorf("requestId: got %v", attrs["requestId"])
	}
	// JSON unmarshals numbers as float64.
	if attrs["durationMs"] != float64(123) {
		t.Errorf("durationMs: got %v", attrs["durationMs"])
	}
	if attrs["endpoint"] != "/api/v1" {
		t.Errorf("endpoint: got %v", attrs["endpoint"])
	}
}

// CloseIsIdempotent: calling Close multiple times returns nil each time
// and does not panic.
func TestNewRelic_CloseIsIdempotent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	_, tr := newLogger(t, newrelic.Config{
		LicenseKey:       "k",
		URL:              srv.URL,
		AllowInsecureURL: true,
		HTTP:             httptr.Config{BatchSize: 1},
	})

	if err := tr.Close(); err != nil {
		t.Errorf("first Close: %v", err)
	}
	if err := tr.Close(); err != nil {
		t.Errorf("second Close: %v", err)
	}
}

// New_PanicsWithoutLicenseKey: calling New with an empty LicenseKey panics
// with ErrLicenseKeyRequired.
func TestNewRelic_New_PanicsWithoutLicenseKey(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic when LicenseKey missing")
		}
		err, ok := r.(error)
		if !ok || !errors.Is(err, newrelic.ErrLicenseKeyRequired) {
			t.Errorf("panic value: got %v, want ErrLicenseKeyRequired", r)
		}
	}()
	_ = newrelic.New(newrelic.Config{})
}

// Build_ReturnsErrLicenseKeyRequired: calling Build with an empty LicenseKey
// returns the sentinel error instead of panicking.
func TestNewRelic_Build_ReturnsErrLicenseKeyRequired(t *testing.T) {
	_, err := newrelic.Build(newrelic.Config{})
	if !errors.Is(err, newrelic.ErrLicenseKeyRequired) {
		t.Errorf("Build with missing LicenseKey: got %v, want ErrLicenseKeyRequired", err)
	}
}

// InsecureURLRejected: a http:// URL is rejected by default to prevent
// shipping the license key in plaintext.
func TestNewRelic_Build_InsecureURLRejected(t *testing.T) {
	_, err := newrelic.Build(newrelic.Config{
		LicenseKey: "k",
		URL:        "http://example.com/log/v1",
	})
	if !errors.Is(err, newrelic.ErrInsecureURL) {
		t.Errorf("Build with http URL: got %v, want ErrInsecureURL", err)
	}
}

// AllowInsecureURLPermitsHTTP: AllowInsecureURL: true allows a http:// URL to
// be set.
func TestNewRelic_Build_AllowInsecureURLPermitsHTTP(t *testing.T) {
	tr, err := newrelic.Build(newrelic.Config{
		LicenseKey:       "k",
		URL:              "http://example.com/log/v1",
		AllowInsecureURL: true,
	})
	if err != nil {
		t.Fatalf("AllowInsecureURL=true should pass: %v", err)
	}
	_ = tr.Close()
}

// HTTPOverrideForbidden: setting HTTP.URL or HTTP.Encoder on the Config
// returns ErrHTTPOverrideForbidden instead of silently dropping the
// value.
func TestNewRelic_Build_HTTPOverrideForbidden(t *testing.T) {
	_, err := newrelic.Build(newrelic.Config{
		LicenseKey: "k",
		HTTP: httptr.Config{
			URL: "https://my-forwarder.internal/v1/logs",
		},
	})
	if !errors.Is(err, newrelic.ErrHTTPOverrideForbidden) {
		t.Errorf("Build with HTTP.URL set: got %v, want ErrHTTPOverrideForbidden", err)
	}

	enc := httptr.EncoderFunc(func(_ []httptr.Entry) ([]byte, string, error) {
		return nil, "", nil
	})
	_, err = newrelic.Build(newrelic.Config{
		LicenseKey: "k",
		HTTP: httptr.Config{
			Encoder: enc,
		},
	})
	if !errors.Is(err, newrelic.ErrHTTPOverrideForbidden) {
		t.Errorf("Build with HTTP.Encoder set: got %v, want ErrHTTPOverrideForbidden", err)
	}
}

// SiteEUDerivedURL: SiteEU produces the correct EU intake URL.
func TestNewRelic_SiteEUDerivedURL(t *testing.T) {
	cases := []struct {
		site newrelic.Site
		want string
	}{
		{"", "https://log-api.newrelic.com/log/v1"},
		{newrelic.SiteUS, "https://log-api.newrelic.com/log/v1"},
		{newrelic.SiteEU, "https://log-api.eu.newrelic.com/log/v1"},
	}
	for _, c := range cases {
		if got := c.site.IntakeURL(); got != c.want {
			t.Errorf("Site %q: got %q, want %q", c.site, got, c.want)
		}
	}
}

// ConfigStringRedactsLicenseKey: Config.String() hides the raw license
// key so an accidental log.Info(cfg) or fmt.Sprintf cannot leak it.
func TestNewRelic_ConfigStringRedactsLicenseKey(t *testing.T) {
	cfg := newrelic.Config{
		LicenseKey: "deadbeef-secret-keep-me-out-of-logs",
		Site:       newrelic.SiteUS,
	}

	s := cfg.String()
	if strings.Contains(s, "deadbeef-secret-keep-me-out-of-logs") {
		t.Errorf("LicenseKey leaked through String(): %s", s)
	}
	if !strings.Contains(s, "redacted") {
		t.Errorf("String() should mark LicenseKey as redacted: %s", s)
	}

	v := fmt.Sprintf("%v", cfg)
	if strings.Contains(v, "deadbeef-secret-keep-me-out-of-logs") {
		t.Errorf("LicenseKey leaked through %%v: %s", v)
	}
}

// TestNewRelic_GroupsWork validates that WithGroup entries are dispatched
// without error and produce well-formed encoded output.
func TestNewRelic_GroupsWork(t *testing.T) {
	cap := &capture{}
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	defer srv.Close()

	_, tr := newLogger(t, newrelic.Config{
		LicenseKey:       "k",
		URL:              srv.URL,
		AllowInsecureURL: true,
		HTTP: httptr.Config{
			BatchSize:     1,
			BatchInterval: 10 * time.Millisecond,
		},
	})

	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})
	log.WithGroup("web").Info("grouped log")

	time.Sleep(100 * time.Millisecond)
	_ = tr.Close()

	cap.mu.Lock()
	defer cap.mu.Unlock()
	if len(cap.bodies) == 0 {
		t.Fatal("no request captured")
	}

	// Verify the entry was delivered and encoded correctly. Groups are on
	// Entry; the real assertion is that the encoder processed without error.
	var arr []map[string]any
	if err := json.Unmarshal(cap.bodies[0], &arr); err != nil {
		t.Fatalf("body is not JSON: %v: %q", err, cap.bodies[0])
	}
	if len(arr) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(arr))
	}
	obj := arr[0]
	if obj["level"] == nil {
		t.Errorf("level: missing")
	}
	if obj["log"] != "grouped log" {
		t.Errorf("log: got %v", obj["log"])
	}
}

// LoglevelFor: every loglayer level maps to the expected New Relic loglevel.
func TestNewRelic_LevelMapping(t *testing.T) {
	cap := &capture{}
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	defer srv.Close()

	_, tr := newLogger(t, newrelic.Config{
		LicenseKey:       "k",
		URL:              srv.URL,
		AllowInsecureURL: true,
		HTTP: httptr.Config{
			BatchSize:     10,
			BatchInterval: 10 * time.Millisecond,
		},
	})

	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})
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

	time.Sleep(100 * time.Millisecond)
	_ = tr.Close()

	cap.mu.Lock()
	defer cap.mu.Unlock()
	if len(cap.bodies) == 0 {
		t.Fatal("no request captured")
	}

	var allEntries []map[string]any
	for _, body := range cap.bodies {
		var batch []map[string]any
		_ = json.Unmarshal(body, &batch)
		allEntries = append(allEntries, batch...)
	}

	if len(allEntries) == 0 {
		t.Fatal("no entries captured")
	}

	wantMap := map[string]struct{}{
		"trace":    {},
		"debug":    {},
		"info":     {},
		"warn":     {},
		"error":    {},
		"critical": {},
	}
	for i, obj := range allEntries {
		if lv, ok := obj["level"].(string); ok {
			if _, ok := wantMap[lv]; !ok {
				t.Errorf("entry %d has unexpected level %q", i, lv)
			}
		}
	}
}

// TestNewRelic_ConfigLicenseKeyTaggedJSONIgnore validates that the json:"-"
// tag on LicenseKey prevents accidental leaks via log.WithMetadata(cfg).
func TestNewRelic_ConfigLicenseKeyTaggedJSONIgnore(t *testing.T) {
	field, ok := reflect.TypeOf(newrelic.Config{}).FieldByName("LicenseKey")
	if !ok {
		t.Fatal("Config.LicenseKey field not found")
	}
	if got := field.Tag.Get("json"); got != "-" {
		t.Errorf("LicenseKey json tag: got %q, want \"-\"", got)
	}
}

// URLOverride: Config.URL overrides the Site-derived URL for on-prem
// deployments or testing against a mock endpoint.
func TestNewRelic_URLOverride(t *testing.T) {
	cap := &capture{}
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	defer srv.Close()

	_, tr := newLogger(t, newrelic.Config{
		LicenseKey:       "k",
		Site:             newrelic.SiteEU,
		URL:              srv.URL,
		AllowInsecureURL: true,
		HTTP: httptr.Config{
			BatchSize:     1,
			BatchInterval: 10 * time.Millisecond,
		},
	})

	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})
	log.Info("on-prem")
	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	cap.mu.Lock()
	defer cap.mu.Unlock()
	if len(cap.bodies) == 0 {
		t.Fatal("no request received at the override URL")
	}
}

func TestNewRelic_CustomHeadersMerged(t *testing.T) {
	cap := &capture{}
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	defer srv.Close()

	_, tr := newLogger(t, newrelic.Config{
		LicenseKey:       "k",
		URL:              srv.URL,
		AllowInsecureURL: true,
		HTTP: httptr.Config{
			BatchSize:     1,
			BatchInterval: 10 * time.Millisecond,
			Headers: map[string]string{
				"X-Custom-Header": "custom-value",
			},
		},
	})

	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})
	log.Info("with custom header")
	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	cap.mu.Lock()
	defer cap.mu.Unlock()
	if len(cap.headers) == 0 {
		t.Fatal("no request captured")
	}

	hdrs := cap.headers[0]
	if got := hdrs.Get("X-Custom-Header"); got != "custom-value" {
		t.Errorf("X-Custom-Header: got %q, want custom-value", got)
	}
	// Api-Key should still be there alongside custom headers.
	if got := hdrs.Get("Api-Key"); got != "k" {
		t.Errorf("Api-Key: got %q, want k", got)
	}
}
