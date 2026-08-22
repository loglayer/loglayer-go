package betterstack

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	httptr "go.loglayer.dev/transports/http/v3"
	"go.loglayer.dev/v3"
)

type capture struct {
	mu      sync.Mutex
	bodies  [][]byte
	headers []http.Header
}

func (c *capture) handler(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	c.mu.Lock()
	defer c.mu.Unlock()
	w.WriteHeader(http.StatusAccepted)
	c.bodies = append(c.bodies, body)
	c.headers = append(c.headers, r.Header.Clone())
}

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
	target := strings.TrimRight(u.target, "/") + req.URL.Path
	parsed, err := http.NewRequestWithContext(req.Context(), req.Method, target, req.Body)
	if err != nil {
		return nil, err
	}
	parsed.Header = req.Header.Clone()
	parsed.ContentLength = req.ContentLength
	return u.base.RoundTrip(parsed)
}

func TestBuild(t *testing.T) {
	t.Parallel()

	cap := &capture{}
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	defer srv.Close()

	cfg := Config{
		SourceToken:      "test-token",
		URL:              srv.URL,
		AllowInsecureURL: true,
		HTTP: httptr.Config{
			BatchSize:     10,
			BatchInterval: time.Hour,
			Client:        rewriteClient(srv),
		},
	}
	tr := New(cfg)
	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})

	log.Info("test")
	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if len(cap.bodies) != 1 {
		t.Fatalf("expected 1 batch, got %d", len(cap.bodies))
	}

	var arr []map[string]any
	if err := json.Unmarshal(cap.bodies[0], &arr); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if len(arr) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(arr))
	}
	if arr[0]["message"] != "test" {
		t.Errorf("message: got %q", arr[0]["message"])
	}
}

func TestBuild_EmptySourceToken(t *testing.T) {
	t.Parallel()

	cfg := Config{}
	tr, err := Build(cfg)

	if tr != nil {
		t.Errorf("expected nil transport, got %v", tr)
	}

	var requiredErr ErrSourceTokenRequired
	if !errors.As(err, &requiredErr) {
		t.Fatalf("expected ErrSourceTokenRequired, got %T: %v", err, err)
	}
}

func TestEncoder_Shape(t *testing.T) {
	t.Parallel()

	cap := &capture{}
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	defer srv.Close()

	cfg := Config{
		SourceToken:      "test-token",
		URL:              srv.URL,
		AllowInsecureURL: true,
		HTTP: httptr.Config{
			BatchSize:     10,
			BatchInterval: time.Hour,
			Client:        rewriteClient(srv),
		},
	}
	tr := New(cfg)
	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})

	log = log.WithFields(map[string]any{"userId": "123"})
	log.WithMetadata(map[string]any{"traceId": "abc"}).Info("test")
	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if len(cap.bodies) != 1 {
		t.Fatalf("expected 1 batch, got %d", len(cap.bodies))
	}

	var arr []map[string]any
	if err := json.Unmarshal(cap.bodies[0], &arr); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if len(arr) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(arr))
	}

	obj := arr[0]

	if obj["message"] != "test" {
		t.Errorf("message: got %q", obj["message"])
	}
	if obj["level"] != "info" {
		t.Errorf("level: got %q", obj["level"])
	}
	if _, ok := obj["dt"].(string); !ok {
		t.Error("expected dt field to be present")
	}
	if obj["userId"] != "123" {
		t.Errorf("userId: got %v", obj["userId"])
	}
	// The metadata map nests under "metadata" per the v3 core default.
	if obj["metadata"].(map[string]any)["traceId"] != "abc" {
		t.Errorf("metadata.traceId: got %v", obj["metadata"])
	}
}

func TestEncoder_TimestampAlwaysPresent(t *testing.T) {
	t.Parallel()

	cap := &capture{}
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	defer srv.Close()

	cfg := Config{
		SourceToken:      "test-token",
		URL:              srv.URL,
		AllowInsecureURL: true,
		HTTP: httptr.Config{
			BatchSize:     10,
			BatchInterval: time.Hour,
			Client:        rewriteClient(srv),
		},
	}
	tr := New(cfg)
	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})

	log.Info("test")
	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	var arr []map[string]any
	if err := json.Unmarshal(cap.bodies[0], &arr); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	obj := arr[0]

	if _, ok := obj["dt"]; !ok {
		t.Error("expected dt field to be present")
	}
}

func TestEncoder_CustomTimestampField(t *testing.T) {
	t.Parallel()

	cap := &capture{}
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	defer srv.Close()

	cfg := Config{
		SourceToken:      "test-token",
		URL:              srv.URL,
		AllowInsecureURL: true,
		TimestampField:   "timestamp",
		HTTP: httptr.Config{
			BatchSize:     10,
			BatchInterval: time.Hour,
			Client:        rewriteClient(srv),
		},
	}
	tr := New(cfg)
	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})

	log.Info("test")
	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	var arr []map[string]any
	if err := json.Unmarshal(cap.bodies[0], &arr); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	obj := arr[0]

	if _, ok := obj["timestamp"]; !ok {
		t.Error("expected timestamp field in payload")
	}
}

func TestString_RedactsSourceToken(t *testing.T) {
	t.Parallel()

	cfg := Config{
		SourceToken: "secret-token-123",
		URL:         "https://in.logs.betterstack.com",
	}

	s := cfg.String()

	if strings.Contains(s, "secret-token-123") {
		t.Error("String() should not expose the source token")
	}

	if !strings.Contains(s, "***redacted***") {
		t.Error("String() should show ***redacted*** for the token")
	}
}

func TestEncoder_AllLogLevels(t *testing.T) {
	t.Parallel()

	levelMap := map[loglayer.LogLevel]string{
		loglayer.LogLevelTrace: "trace",
		loglayer.LogLevelDebug: "debug",
		loglayer.LogLevelInfo:  "info",
		loglayer.LogLevelWarn:  "warn",
		loglayer.LogLevelError: "error",
		loglayer.LogLevelFatal: "fatal",
		loglayer.LogLevelPanic: "panic",
	}

	for level, expected := range levelMap {
		level, expected := level, expected

		t.Run(level.String(), func(t *testing.T) {
			t.Parallel()

			cap := &capture{}
			srv := httptest.NewServer(http.HandlerFunc(cap.handler))
			defer srv.Close()

			cfg := Config{
				SourceToken:      "test-token",
				URL:              srv.URL,
				AllowInsecureURL: true,
				HTTP: httptr.Config{
					BatchSize:     10,
					BatchInterval: time.Hour,
					Client:        rewriteClient(srv),
				},
			}
			tr := New(cfg)
			log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})

			switch level {
			case loglayer.LogLevelTrace:
				log.Trace("test")
			case loglayer.LogLevelDebug:
				log.Debug("test")
			case loglayer.LogLevelInfo:
				log.Info("test")
			case loglayer.LogLevelWarn:
				log.Warn("test")
			case loglayer.LogLevelError:
				log.Error("test")
			case loglayer.LogLevelFatal:
				log.Fatal("test")
			case loglayer.LogLevelPanic:
				defer func() { _ = recover() }()
				log.Panic("test")
			}

			if err := tr.Close(); err != nil {
				t.Fatalf("Close: %v", err)
			}

			var arr []map[string]any
			if err := json.Unmarshal(cap.bodies[0], &arr); err != nil {
				t.Fatalf("body is not JSON: %v", err)
			}
			obj := arr[0]

			if obj["level"] != expected {
				t.Errorf("expected level '%s' for %v, got %q", expected, level, obj["level"])
			}
		})
	}
}

func TestBuild_HTTPHeadersMerge(t *testing.T) {
	t.Parallel()

	cap := &capture{}
	srv := httptest.NewServer(http.HandlerFunc(cap.handler))
	defer srv.Close()

	cfg := Config{
		SourceToken:      "test-token",
		URL:              srv.URL,
		AllowInsecureURL: true,
		HTTP: httptr.Config{
			BatchSize:     10,
			BatchInterval: time.Hour,
			Client:        rewriteClient(srv),
			Headers: map[string]string{
				"X-Custom-Header": "custom-value",
				"Authorization":   "Bearer should-be-overridden", // should be replaced
				"Content-Type":    "should-also-be-overridden",   // should be replaced
			},
		},
	}
	tr := New(cfg)
	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})
	log.Info("test")

	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	arrHeaders := cap.headers
	if len(arrHeaders) != 1 {
		t.Fatalf("expected 1 request headers, got %d", len(arrHeaders))
	}

	reqHeaders := arrHeaders[0]

	authHeader := reqHeaders.Get("Authorization")
	contentTypeHeader := reqHeaders.Get("Content-Type")

	if authHeader != "Bearer test-token" {
		t.Errorf("Expected Authorization header 'Bearer test-token', got %q", authHeader)
	}
	if contentTypeHeader != "application/json" {
		t.Errorf("Expected Content-Type header 'application/json', got %q", contentTypeHeader)
	}

	customHeader := reqHeaders.Get("X-Custom-Header")
	if customHeader != "custom-value" {
		t.Errorf("Expected X-Custom-Header 'custom-value' to be preserved, got %q", customHeader)
	}
}
