package zerolog_test

import (
	"bytes"
	"errors"
	"testing"

	zlog "github.com/rs/zerolog"

	"go.loglayer.dev"
	"go.loglayer.dev/internal/transporttest"
	"go.loglayer.dev/transport"
	llzero "go.loglayer.dev/transports/zerolog"
)

func newLogger(cfg llzero.Config) (*loglayer.LogLayer, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	z := zlog.New(buf)
	cfg.Logger = &z
	if cfg.BaseConfig.ID == "" {
		cfg.BaseConfig.ID = "zerolog"
	}
	t := llzero.New(cfg)
	return loglayer.New(loglayer.Config{DisableFatalExit: true, Transport: t}), buf
}

func TestZerologSimpleMessage(t *testing.T) {
	log, buf := newLogger(llzero.Config{})
	log.Info("hello")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["message"] != "hello" {
		t.Errorf("message: got %v", obj["message"])
	}
	if obj["level"] != "info" {
		t.Errorf("level: got %v", obj["level"])
	}
}

func TestZerologMultipleMessages(t *testing.T) {
	log, buf := newLogger(llzero.Config{})
	log.Info("part1", "part2")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["message"] != "part1 part2" {
		t.Errorf("message: got %v", obj["message"])
	}
}

func TestZerologLevels(t *testing.T) {
	cases := []struct {
		fn    func(*loglayer.LogLayer)
		level string
	}{
		{func(l *loglayer.LogLayer) { l.Trace("x") }, "trace"},
		{func(l *loglayer.LogLayer) { l.Debug("x") }, "debug"},
		{func(l *loglayer.LogLayer) { l.Info("x") }, "info"},
		{func(l *loglayer.LogLayer) { l.Warn("x") }, "warn"},
		{func(l *loglayer.LogLayer) { l.Error("x") }, "error"},
		{func(l *loglayer.LogLayer) { l.Fatal("x") }, "fatal"},
	}
	for _, c := range cases {
		buf := &bytes.Buffer{}
		z := zlog.New(buf).Level(zlog.TraceLevel)
		log := loglayer.New(loglayer.Config{DisableFatalExit: true,
			Transport: llzero.New(llzero.Config{
				BaseConfig: transport.BaseConfig{ID: "zerolog"},
				Logger:     &z,
			}),
		})
		c.fn(log)
		obj := transporttest.ParseJSONLine(t, buf)
		if obj["level"] != c.level {
			t.Errorf("expected level %q, got %v", c.level, obj["level"])
		}
	}
}

func TestZerologFatalDoesNotExit(t *testing.T) {
	// If our transport accidentally called zerolog.Fatal() the test process
	// would terminate. Reaching the assertion proves we routed via WithLevel.
	log, buf := newLogger(llzero.Config{})
	log.Fatal("survives")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["level"] != "fatal" {
		t.Errorf("expected fatal level entry, got %v", obj["level"])
	}
}

func TestZerologMapMetadataMerged(t *testing.T) {
	log, buf := newLogger(llzero.Config{})
	log.WithMetadata(map[string]any{"requestId": "xyz", "n": 42}).Info("req")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["requestId"] != "xyz" {
		t.Errorf("requestId: got %v", obj["requestId"])
	}
	if obj["n"] != float64(42) {
		t.Errorf("n: got %v", obj["n"])
	}
}

func TestZerologStructMetadataNested(t *testing.T) {
	type user struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	log, buf := newLogger(llzero.Config{})
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

func TestZerologCustomMetadataFieldName(t *testing.T) {
	type user struct {
		ID int `json:"id"`
	}
	log, buf := newLogger(llzero.Config{MetadataFieldName: "user"})
	log.WithMetadata(user{ID: 9}).Info("hi")
	obj := transporttest.ParseJSONLine(t, buf)
	if _, ok := obj["user"].(map[string]any); !ok {
		t.Errorf("expected 'user' key, got %v", obj)
	}
}

func TestZerologContextMerged(t *testing.T) {
	log, buf := newLogger(llzero.Config{})
	log = log.WithFields(loglayer.Fields{"service": "api"})
	log.Info("ctx test")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["service"] != "api" {
		t.Errorf("service: got %v", obj["service"])
	}
}

func TestZerologWithError(t *testing.T) {
	log, buf := newLogger(llzero.Config{})
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

func TestZerologLevelFiltering(t *testing.T) {
	buf := &bytes.Buffer{}
	z := zlog.New(buf).Level(zlog.TraceLevel)
	tr := llzero.New(llzero.Config{
		BaseConfig: transport.BaseConfig{ID: "zerolog", Level: loglayer.LogLevelError},
		Logger:     &z,
	})
	log := loglayer.New(loglayer.Config{DisableFatalExit: true, Transport: tr})
	log.Warn("dropped")
	if buf.Len() != 0 {
		t.Errorf("warn should be filtered at error level, got: %q", buf.String())
	}
	log.Error("passes")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["message"] != "passes" {
		t.Errorf("message: got %v", obj["message"])
	}
}

func TestZerologGetLoggerInstance(t *testing.T) {
	buf := &bytes.Buffer{}
	z := zlog.New(buf)
	tr := llzero.New(llzero.Config{
		BaseConfig: transport.BaseConfig{ID: "zerolog"},
		Logger:     &z,
	})
	log := loglayer.New(loglayer.Config{DisableFatalExit: true, Transport: tr})
	inst := log.GetLoggerInstance("zerolog")
	if _, ok := inst.(*zlog.Logger); !ok {
		t.Errorf("expected *zerolog.Logger, got %T", inst)
	}
}
