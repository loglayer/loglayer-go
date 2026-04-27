package phuslu_test

import (
	"bytes"
	"errors"
	"testing"

	plog "github.com/phuslu/log"

	"go.loglayer.dev"
	"go.loglayer.dev/internal/transporttest"
	"go.loglayer.dev/transport"
	llphuslu "go.loglayer.dev/transports/phuslu"
)

func newLogger(cfg llphuslu.Config) (*loglayer.LogLayer, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	pl := &plog.Logger{
		Level:  plog.TraceLevel,
		Writer: &plog.IOWriter{Writer: buf},
	}
	cfg.Logger = pl
	if cfg.BaseConfig.ID == "" {
		cfg.BaseConfig.ID = "phuslu"
	}
	t := llphuslu.New(cfg)
	return loglayer.New(loglayer.Config{DisableFatalExit: true, Transport: t}), buf
}

func TestPhusluSimpleMessage(t *testing.T) {
	log, buf := newLogger(llphuslu.Config{})
	log.Info("hello")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["message"] != "hello" {
		t.Errorf("message: got %v", obj["message"])
	}
	if obj["level"] != "info" {
		t.Errorf("level: got %v", obj["level"])
	}
}

func TestPhusluMultipleMessages(t *testing.T) {
	log, buf := newLogger(llphuslu.Config{})
	log.Info("part1", "part2")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["message"] != "part1 part2" {
		t.Errorf("message: got %v", obj["message"])
	}
}

func TestPhusluLevels(t *testing.T) {
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
		// Skip fatal — phuslu always calls os.Exit on FatalLevel and there is
		// no clean way to suppress it (see phuslu.go).
		if c.level == "fatal" {
			continue
		}
		log, buf := newLogger(llphuslu.Config{})
		c.fn(log)
		obj := transporttest.ParseJSONLine(t, buf)
		if obj["level"] != c.level {
			t.Errorf("expected level %q, got %v", c.level, obj["level"])
		}
	}
}

// Note: phuslu's Logger.WithLevel(FatalLevel).Msg() calls os.Exit, so there is
// no test that exercises log.Fatal() through this wrapper. The phuslu transport
// docs document this limitation.

func TestPhusluMapMetadataMerged(t *testing.T) {
	log, buf := newLogger(llphuslu.Config{})
	log.WithMetadata(loglayer.Metadata{"requestId": "xyz", "n": 42}).Info("req")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["requestId"] != "xyz" {
		t.Errorf("requestId: got %v", obj["requestId"])
	}
	if obj["n"] != float64(42) {
		t.Errorf("n: got %v", obj["n"])
	}
}

func TestPhusluStructMetadataNested(t *testing.T) {
	type user struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	log, buf := newLogger(llphuslu.Config{})
	log.WithMetadata(user{ID: 7, Name: "Alice"}).Info("hi")
	obj := transporttest.ParseJSONLine(t, buf)
	nested, ok := obj["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("expected nested metadata, got %v", obj)
	}
	if nested["id"] != float64(7) || nested["name"] != "Alice" {
		t.Errorf("nested fields: got %v", nested)
	}
}

func TestPhusluCustomMetadataFieldName(t *testing.T) {
	type user struct {
		ID int `json:"id"`
	}
	log, buf := newLogger(llphuslu.Config{MetadataFieldName: "user"})
	log.WithMetadata(user{ID: 9}).Info("hi")
	obj := transporttest.ParseJSONLine(t, buf)
	if _, ok := obj["user"].(map[string]any); !ok {
		t.Errorf("expected 'user' key, got %v", obj)
	}
}

func TestPhusluContextMerged(t *testing.T) {
	log, buf := newLogger(llphuslu.Config{})
	log = log.WithFields(loglayer.Fields{"service": "api"})
	log.Info("ctx test")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["service"] != "api" {
		t.Errorf("service: got %v", obj["service"])
	}
}

func TestPhusluWithError(t *testing.T) {
	log, buf := newLogger(llphuslu.Config{})
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

func TestPhusluLevelFiltering(t *testing.T) {
	buf := &bytes.Buffer{}
	pl := &plog.Logger{
		Level:  plog.TraceLevel,
		Writer: &plog.IOWriter{Writer: buf},
	}
	tr := llphuslu.New(llphuslu.Config{
		BaseConfig: transport.BaseConfig{ID: "phuslu", Level: loglayer.LogLevelError},
		Logger:     pl,
	})
	log := loglayer.New(loglayer.Config{DisableFatalExit: true, Transport: tr})
	log.Warn("dropped")
	if buf.Len() != 0 {
		t.Errorf("warn should be filtered, got: %q", buf.String())
	}
	log.Error("passes")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["message"] != "passes" {
		t.Errorf("message: got %v", obj["message"])
	}
}

func TestPhusluGetLoggerInstance(t *testing.T) {
	pl := &plog.Logger{Writer: &plog.IOWriter{Writer: &bytes.Buffer{}}}
	tr := llphuslu.New(llphuslu.Config{
		BaseConfig: transport.BaseConfig{ID: "phuslu"},
		Logger:     pl,
	})
	log := loglayer.New(loglayer.Config{DisableFatalExit: true, Transport: tr})
	if _, ok := log.GetLoggerInstance("phuslu").(*plog.Logger); !ok {
		t.Errorf("expected *phuslu/log.Logger, got %T", log.GetLoggerInstance("phuslu"))
	}
}
