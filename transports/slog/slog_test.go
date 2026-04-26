package slog_test

import (
	"bytes"
	"context"
	"errors"
	stdlibslog "log/slog"
	"testing"

	"go.loglayer.dev/loglayer"
	"go.loglayer.dev/loglayer/internal/transporttest"
	"go.loglayer.dev/loglayer/transport"
	llslog "go.loglayer.dev/loglayer/transports/slog"
)

func newLogger(cfg llslog.Config) (*loglayer.LogLayer, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	cfg.Logger = stdlibslog.New(stdlibslog.NewJSONHandler(buf, &stdlibslog.HandlerOptions{
		Level: stdlibslog.LevelDebug,
	}))
	if cfg.BaseConfig.ID == "" {
		cfg.BaseConfig.ID = "slog"
	}
	t := llslog.New(cfg)
	return loglayer.New(loglayer.Config{
		Transport:        t,
		DisableFatalExit: true,
	}), buf
}


func TestSlogSimpleMessage(t *testing.T) {
	log, buf := newLogger(llslog.Config{})
	log.Info("hello")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["msg"] != "hello" {
		t.Errorf("msg: got %v", obj["msg"])
	}
	if obj["level"] != "INFO" {
		t.Errorf("level: got %v", obj["level"])
	}
}

func TestSlogMultipleMessages(t *testing.T) {
	log, buf := newLogger(llslog.Config{})
	log.Info("part1", "part2")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["msg"] != "part1 part2" {
		t.Errorf("msg: got %v", obj["msg"])
	}
}

func TestSlogLevels(t *testing.T) {
	cases := []struct {
		fn    func(*loglayer.LogLayer)
		level string
	}{
		{func(l *loglayer.LogLayer) { l.Debug("x") }, "DEBUG"},
		{func(l *loglayer.LogLayer) { l.Info("x") }, "INFO"},
		{func(l *loglayer.LogLayer) { l.Warn("x") }, "WARN"},
		{func(l *loglayer.LogLayer) { l.Error("x") }, "ERROR"},
	}
	for _, c := range cases {
		log, buf := newLogger(llslog.Config{})
		c.fn(log)
		obj := transporttest.ParseJSONLine(t, buf)
		if obj["level"] != c.level {
			t.Errorf("expected level %q, got %v", c.level, obj["level"])
		}
	}
}

func TestSlogTraceMapsToDebug(t *testing.T) {
	log, buf := newLogger(llslog.Config{})
	log.Trace("trace msg")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["level"] != "DEBUG" {
		t.Errorf("trace should map to DEBUG in slog, got %v", obj["level"])
	}
}

func TestSlogFatalAboveError(t *testing.T) {
	log, buf := newLogger(llslog.Config{})
	log.Fatal("survives")
	obj := transporttest.ParseJSONLine(t, buf)
	// slog renders custom levels as "ERROR+N".
	if obj["level"] != "ERROR+4" {
		t.Errorf("expected ERROR+4 for fatal, got %v", obj["level"])
	}
}

func TestSlogMapMetadataMerged(t *testing.T) {
	log, buf := newLogger(llslog.Config{})
	log.WithMetadata(loglayer.Metadata{"requestId": "xyz", "n": 42}).Info("req")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["requestId"] != "xyz" {
		t.Errorf("requestId: got %v", obj["requestId"])
	}
	if obj["n"] != float64(42) {
		t.Errorf("n: got %v", obj["n"])
	}
}

func TestSlogStructMetadataNested(t *testing.T) {
	type user struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	log, buf := newLogger(llslog.Config{})
	log.WithMetadata(user{ID: 7, Name: "Alice"}).Info("hi")
	obj := transporttest.ParseJSONLine(t, buf)
	nested, ok := obj["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("expected nested metadata field, got %v", obj)
	}
	// slog's JSON handler invokes MarshalJSON for structs, so json tags apply.
	if nested["id"] != float64(7) || nested["name"] != "Alice" {
		t.Errorf("nested fields: got %v", nested)
	}
}

func TestSlogCustomMetadataFieldName(t *testing.T) {
	type user struct{ ID int }
	log, buf := newLogger(llslog.Config{MetadataFieldName: "user"})
	log.WithMetadata(user{ID: 9}).Info("hi")
	obj := transporttest.ParseJSONLine(t, buf)
	if _, ok := obj["user"].(map[string]any); !ok {
		t.Errorf("expected 'user' key, got %v", obj)
	}
}

func TestSlogContextMerged(t *testing.T) {
	log, buf := newLogger(llslog.Config{})
	log = log.WithFields(loglayer.Fields{"service": "api"})
	log.Info("ctx test")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["service"] != "api" {
		t.Errorf("service: got %v", obj["service"])
	}
}

func TestSlogWithError(t *testing.T) {
	log, buf := newLogger(llslog.Config{})
	log.WithError(errors.New("boom")).Error("failed")
	obj := transporttest.ParseJSONLine(t, buf)
	errField, ok := obj["err"].(map[string]any)
	if !ok {
		t.Fatalf("expected err field, got %v", obj)
	}
	if errField["message"] != "boom" {
		t.Errorf("err.message: got %v", errField["message"])
	}
}

func TestSlogWithCtxPassedToHandler(t *testing.T) {
	// Verify the context.Context attached via WithCtx reaches slog's handler.
	type ctxKey struct{}
	ctx := context.WithValue(context.Background(), ctxKey{}, "trace-id-123")

	var captured context.Context
	handler := &spyHandler{onHandle: func(c context.Context, _ stdlibslog.Record) error {
		captured = c
		return nil
	}}
	logger := stdlibslog.New(handler)
	tr := llslog.New(llslog.Config{
		BaseConfig: transport.BaseConfig{ID: "slog"},
		Logger:     logger,
	})
	log := loglayer.New(loglayer.Config{
		Transport:        tr,
		DisableFatalExit: true,
	})

	log.WithCtx(ctx).Info("with ctx")
	if captured == nil || captured.Value(ctxKey{}) != "trace-id-123" {
		t.Errorf("Ctx not propagated to slog handler: %v", captured)
	}
}

func TestSlogLevelFiltering(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := stdlibslog.New(stdlibslog.NewJSONHandler(buf, &stdlibslog.HandlerOptions{
		Level: stdlibslog.LevelDebug,
	}))
	tr := llslog.New(llslog.Config{
		BaseConfig: transport.BaseConfig{ID: "slog", Level: loglayer.LogLevelError},
		Logger:     logger,
	})
	log := loglayer.New(loglayer.Config{
		Transport:        tr,
		DisableFatalExit: true,
	})
	log.Warn("dropped")
	if buf.Len() != 0 {
		t.Errorf("warn should be filtered, got: %q", buf.String())
	}
	log.Error("passes")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["msg"] != "passes" {
		t.Errorf("msg: got %v", obj["msg"])
	}
}

func TestSlogGetLoggerInstance(t *testing.T) {
	logger := stdlibslog.New(stdlibslog.NewJSONHandler(&bytes.Buffer{}, nil))
	tr := llslog.New(llslog.Config{
		BaseConfig: transport.BaseConfig{ID: "slog"},
		Logger:     logger,
	})
	log := loglayer.New(loglayer.Config{
		Transport:        tr,
		DisableFatalExit: true,
	})
	if _, ok := log.GetLoggerInstance("slog").(*stdlibslog.Logger); !ok {
		t.Errorf("expected *slog.Logger, got %T", log.GetLoggerInstance("slog"))
	}
}

// spyHandler captures the context.Context handed to Handle so tests can
// verify Ctx propagation end-to-end.
type spyHandler struct {
	onHandle func(context.Context, stdlibslog.Record) error
}

func (h *spyHandler) Enabled(_ context.Context, _ stdlibslog.Level) bool { return true }
func (h *spyHandler) Handle(ctx context.Context, r stdlibslog.Record) error {
	return h.onHandle(ctx, r)
}
func (h *spyHandler) WithAttrs(_ []stdlibslog.Attr) stdlibslog.Handler { return h }
func (h *spyHandler) WithGroup(_ string) stdlibslog.Handler            { return h }
