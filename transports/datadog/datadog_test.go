package datadog_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"go.loglayer.dev"
	"go.loglayer.dev/transports/datadog"
	httptr "go.loglayer.dev/transports/http"
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

func TestDatadog_RealEncoderShape(t *testing.T) {
	cap := &capture{}
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	defer srv.Close()

	// Use the real datadog.New wired against srv. We achieve the URL
	// override by injecting the test server URL at the http layer via a
	// custom Client that rewrites the request URL.
	cfg := datadog.Config{
		APIKey:   "secret-abc",
		Source:   "go",
		Service:  "test-svc",
		Hostname: "h1",
		Tags:     "env:test,team:platform",
		HTTP: httptr.Config{
			BatchSize:     10,
			BatchInterval: time.Hour,
			Client:        rewriteClient(srv),
		},
	}
	tr := datadog.New(cfg)
	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})

	log = log.WithFields(loglayer.Fields{"requestId": "abc"})
	log.WithMetadata(loglayer.Metadata{"durationMs": 42}).Info("served")
	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if len(cap.bodies) != 1 {
		t.Fatalf("expected 1 batch, got %d", len(cap.bodies))
	}

	// Header check
	hdrs := cap.headers[0]
	if hdrs.Get("DD-API-KEY") != "secret-abc" {
		t.Errorf("DD-API-KEY: got %q", hdrs.Get("DD-API-KEY"))
	}
	if hdrs.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type: got %q", hdrs.Get("Content-Type"))
	}

	// Body check
	var arr []map[string]any
	if err := json.Unmarshal(cap.bodies[0], &arr); err != nil {
		t.Fatalf("body is not JSON: %v: %q", err, cap.bodies[0])
	}
	if len(arr) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(arr))
	}
	obj := arr[0]
	want := map[string]any{
		"message":   "served",
		"status":    "info",
		"ddsource":  "go",
		"service":   "test-svc",
		"hostname":  "h1",
		"ddtags":    "env:test,team:platform",
		"requestId": "abc",
	}
	for k, v := range want {
		if obj[k] != v {
			t.Errorf("entry[%q]: got %v, want %v", k, obj[k], v)
		}
	}
	if obj["durationMs"] != float64(42) {
		t.Errorf("durationMs: got %v", obj["durationMs"])
	}
	if _, hasDate := obj["date"]; !hasDate {
		t.Errorf("expected date field, got %v", obj)
	}
}

func TestDatadog_StatusForLevels(t *testing.T) {
	cap := &capture{}
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	defer srv.Close()

	tr := datadog.New(datadog.Config{
		APIKey: "k",
		Source: "go",
		HTTP: httptr.Config{
			BatchSize:     10,
			BatchInterval: time.Hour,
			Client:        rewriteClient(srv),
		},
	})
	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})

	log.Trace("t")
	log.Debug("d")
	log.Info("i")
	log.Warn("w")
	log.Error("e")
	log.Fatal("f")
	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	var arr []map[string]any
	_ = json.Unmarshal(cap.bodies[0], &arr)
	wantStatuses := []string{"debug", "debug", "info", "warning", "error", "critical"}
	for i, want := range wantStatuses {
		if arr[i]["status"] != want {
			t.Errorf("entry %d status: got %v, want %s", i, arr[i]["status"], want)
		}
	}
}

func TestDatadog_OmitsEmptyOptionalFields(t *testing.T) {
	cap := &capture{}
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	defer srv.Close()

	tr := datadog.New(datadog.Config{
		APIKey: "k",
		// no Source, Service, Hostname, Tags
		HTTP: httptr.Config{
			BatchSize:     10,
			BatchInterval: time.Hour,
			Client:        rewriteClient(srv),
		},
	})
	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})
	log.Info("hello")
	_ = tr.Close()

	var arr []map[string]any
	_ = json.Unmarshal(cap.bodies[0], &arr)
	for _, key := range []string{"ddsource", "service", "hostname", "ddtags"} {
		if _, present := arr[0][key]; present {
			t.Errorf("expected %q to be absent when not configured, got %v", key, arr[0][key])
		}
	}
}

func TestDatadog_PanicsWithoutAPIKey(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when APIKey missing")
		}
	}()
	_ = datadog.New(datadog.Config{})
}

func TestDatadog_SiteIntakeURL(t *testing.T) {
	cases := []struct {
		site datadog.Site
		want string
	}{
		{"", "https://http-intake.logs.datadoghq.com/api/v2/logs"},
		{datadog.SiteUS1, "https://http-intake.logs.datadoghq.com/api/v2/logs"},
		{datadog.SiteUS3, "https://http-intake.logs.us3.datadoghq.com/api/v2/logs"},
		{datadog.SiteUS5, "https://http-intake.logs.us5.datadoghq.com/api/v2/logs"},
		{datadog.SiteEU, "https://http-intake.logs.datadoghq.eu/api/v2/logs"},
		{datadog.SiteAP1, "https://http-intake.logs.ap1.datadoghq.com/api/v2/logs"},
	}
	for _, c := range cases {
		if got := c.site.IntakeURL(); got != c.want {
			t.Errorf("Site %q: got %q, want %q", c.site, got, c.want)
		}
	}
}

func TestDatadog_NonMapMetadataNestedUnderKey(t *testing.T) {
	cap := &capture{}
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	defer srv.Close()

	tr := datadog.New(datadog.Config{
		APIKey: "k",
		HTTP: httptr.Config{
			BatchSize:     10,
			BatchInterval: time.Hour,
			Client:        rewriteClient(srv),
		},
	})
	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})

	type ev struct {
		Op string `json:"op"`
	}
	log.WithMetadata(ev{Op: "load"}).Info("did")
	_ = tr.Close()

	var arr []map[string]any
	_ = json.Unmarshal(cap.bodies[0], &arr)
	meta, ok := arr[0]["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("expected nested metadata object, got %T: %v", arr[0]["metadata"], arr[0]["metadata"])
	}
	if meta["op"] != "load" {
		t.Errorf("metadata.op: got %v", meta["op"])
	}
}

// rewriteClient returns an http.Client whose Transport rewrites every
// request URL to point at srv. Lets the production code path use the real
// Datadog intake URL while tests stay offline.
func rewriteClient(srv *httptest.Server) *http.Client {
	base := srv.Client().Transport
	if base == nil {
		base = http.DefaultTransport
	}
	return &http.Client{
		Timeout: 5 * time.Second,
		Transport: &urlRewriter{
			base:   base,
			target: srv.URL,
		},
	}
}

type urlRewriter struct {
	base   http.RoundTripper
	target string
}

func (u *urlRewriter) RoundTrip(req *http.Request) (*http.Response, error) {
	// Replace the request URL host/scheme with srv.URL while preserving the
	// path so the production endpoint suffix (e.g. /api/v2/logs) is captured.
	target := strings.TrimRight(u.target, "/") + req.URL.Path
	parsed, err := http.NewRequestWithContext(req.Context(), req.Method, target, req.Body)
	if err != nil {
		return nil, err
	}
	parsed.Header = req.Header.Clone()
	parsed.ContentLength = req.ContentLength
	return u.base.RoundTrip(parsed)
}
