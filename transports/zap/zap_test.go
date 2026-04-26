package zap_test

import (
	"bytes"
	"errors"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.loglayer.dev/loglayer"
	"go.loglayer.dev/loglayer/internal/transporttest"
	"go.loglayer.dev/loglayer/transport"
	llzap "go.loglayer.dev/loglayer/transports/zap"
)

func newLogger(cfg llzap.Config) (*loglayer.LogLayer, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	enc := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	core := zapcore.NewCore(enc, zapcore.AddSync(buf), zapcore.DebugLevel)
	cfg.Logger = zap.New(core)
	if cfg.BaseConfig.ID == "" {
		cfg.BaseConfig.ID = "zap"
	}
	t := llzap.New(cfg)
	return loglayer.New(loglayer.Config{DisableFatalExit: true, Transport: t}), buf
}


func TestZapSimpleMessage(t *testing.T) {
	log, buf := newLogger(llzap.Config{})
	log.Info("hello")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["msg"] != "hello" {
		t.Errorf("msg: got %v", obj["msg"])
	}
	if obj["level"] != "info" {
		t.Errorf("level: got %v", obj["level"])
	}
}

func TestZapMultipleMessages(t *testing.T) {
	log, buf := newLogger(llzap.Config{})
	log.Info("part1", "part2")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["msg"] != "part1 part2" {
		t.Errorf("msg: got %v", obj["msg"])
	}
}

func TestZapLevels(t *testing.T) {
	cases := []struct {
		fn    func(*loglayer.LogLayer)
		level string
	}{
		{func(l *loglayer.LogLayer) { l.Debug("x") }, "debug"},
		{func(l *loglayer.LogLayer) { l.Info("x") }, "info"},
		{func(l *loglayer.LogLayer) { l.Warn("x") }, "warn"},
		{func(l *loglayer.LogLayer) { l.Error("x") }, "error"},
		{func(l *loglayer.LogLayer) { l.Fatal("x") }, "fatal"},
	}
	for _, c := range cases {
		log, buf := newLogger(llzap.Config{})
		c.fn(log)
		obj := transporttest.ParseJSONLine(t, buf)
		if obj["level"] != c.level {
			t.Errorf("expected level %q, got %v", c.level, obj["level"])
		}
	}
}

func TestZapTraceMapsToDebug(t *testing.T) {
	log, buf := newLogger(llzap.Config{})
	log.Trace("trace msg")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["level"] != "debug" {
		t.Errorf("trace should map to debug in zap, got %v", obj["level"])
	}
}

func TestZapFatalDoesNotExit(t *testing.T) {
	// If the fatal hook weren't neutralized this would terminate the process.
	log, buf := newLogger(llzap.Config{})
	log.Fatal("survives")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["level"] != "fatal" {
		t.Errorf("expected fatal entry, got %v", obj["level"])
	}
}

func TestZapMapMetadataMerged(t *testing.T) {
	log, buf := newLogger(llzap.Config{})
	log.WithMetadata(map[string]any{"requestId": "xyz", "n": 42}).Info("req")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["requestId"] != "xyz" {
		t.Errorf("requestId: got %v", obj["requestId"])
	}
	if obj["n"] != float64(42) {
		t.Errorf("n: got %v", obj["n"])
	}
}

func TestZapStructMetadataNested(t *testing.T) {
	type user struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	log, buf := newLogger(llzap.Config{})
	log.WithMetadata(user{ID: 7, Name: "Alice"}).Info("hi")
	obj := transporttest.ParseJSONLine(t, buf)
	nested, ok := obj["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("expected nested metadata field, got %v", obj)
	}
	if nested["id"] != float64(7) || nested["name"] != "Alice" {
		t.Errorf("nested fields: got %v", nested)
	}
}

func TestZapCustomMetadataFieldName(t *testing.T) {
	type user struct {
		ID int `json:"id"`
	}
	log, buf := newLogger(llzap.Config{MetadataFieldName: "user"})
	log.WithMetadata(user{ID: 9}).Info("hi")
	obj := transporttest.ParseJSONLine(t, buf)
	if _, ok := obj["user"].(map[string]any); !ok {
		t.Errorf("expected 'user' key, got %v", obj)
	}
}

func TestZapContextMerged(t *testing.T) {
	log, buf := newLogger(llzap.Config{})
	log = log.WithFields(loglayer.Fields{"service": "api"})
	log.Info("ctx test")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["service"] != "api" {
		t.Errorf("service: got %v", obj["service"])
	}
}

func TestZapWithError(t *testing.T) {
	log, buf := newLogger(llzap.Config{})
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

func TestZapLevelFiltering(t *testing.T) {
	buf := &bytes.Buffer{}
	enc := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	core := zapcore.NewCore(enc, zapcore.AddSync(buf), zapcore.DebugLevel)
	tr := llzap.New(llzap.Config{
		BaseConfig: transport.BaseConfig{ID: "zap", Level: loglayer.LogLevelError},
		Logger:     zap.New(core),
	})
	log := loglayer.New(loglayer.Config{DisableFatalExit: true, Transport: tr})
	log.Warn("dropped")
	if buf.Len() != 0 {
		t.Errorf("warn should be filtered at error level, got: %q", buf.String())
	}
	log.Error("passes")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["msg"] != "passes" {
		t.Errorf("msg: got %v", obj["msg"])
	}
}

func TestZapGetLoggerInstance(t *testing.T) {
	enc := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	core := zapcore.NewCore(enc, zapcore.AddSync(&bytes.Buffer{}), zapcore.DebugLevel)
	tr := llzap.New(llzap.Config{
		BaseConfig: transport.BaseConfig{ID: "zap"},
		Logger:     zap.New(core),
	})
	log := loglayer.New(loglayer.Config{DisableFatalExit: true, Transport: tr})
	inst := log.GetLoggerInstance("zap")
	if _, ok := inst.(*zap.Logger); !ok {
		t.Errorf("expected *zap.Logger, got %T", inst)
	}
}
