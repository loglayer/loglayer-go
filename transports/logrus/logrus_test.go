package logrus_test

import (
	"bytes"
	"errors"
	"testing"

	logrusbase "github.com/sirupsen/logrus"

	"go.loglayer.dev/loglayer"
	"go.loglayer.dev/loglayer/internal/transporttest"
	"go.loglayer.dev/loglayer/transport"
	lllogrus "go.loglayer.dev/loglayer/transports/logrus"
)

func newLogger(cfg lllogrus.Config) (*loglayer.LogLayer, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	base := logrusbase.New()
	base.Out = buf
	base.Formatter = &logrusbase.JSONFormatter{}
	base.Level = logrusbase.TraceLevel
	cfg.Logger = base
	if cfg.BaseConfig.ID == "" {
		cfg.BaseConfig.ID = "logrus"
	}
	t := lllogrus.New(cfg)
	return loglayer.New(loglayer.Config{DisableFatalExit: true, Transport: t}), buf
}


func TestLogrusSimpleMessage(t *testing.T) {
	log, buf := newLogger(lllogrus.Config{})
	log.Info("hello")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["msg"] != "hello" {
		t.Errorf("msg: got %v", obj["msg"])
	}
	if obj["level"] != "info" {
		t.Errorf("level: got %v", obj["level"])
	}
}

func TestLogrusMultipleMessages(t *testing.T) {
	log, buf := newLogger(lllogrus.Config{})
	log.Info("part1", "part2")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["msg"] != "part1 part2" {
		t.Errorf("msg: got %v", obj["msg"])
	}
}

func TestLogrusLevels(t *testing.T) {
	cases := []struct {
		fn    func(*loglayer.LogLayer)
		level string
	}{
		{func(l *loglayer.LogLayer) { l.Trace("x") }, "trace"},
		{func(l *loglayer.LogLayer) { l.Debug("x") }, "debug"},
		{func(l *loglayer.LogLayer) { l.Info("x") }, "info"},
		{func(l *loglayer.LogLayer) { l.Warn("x") }, "warning"},
		{func(l *loglayer.LogLayer) { l.Error("x") }, "error"},
		{func(l *loglayer.LogLayer) { l.Fatal("x") }, "fatal"},
	}
	for _, c := range cases {
		log, buf := newLogger(lllogrus.Config{})
		c.fn(log)
		obj := transporttest.ParseJSONLine(t, buf)
		if obj["level"] != c.level {
			t.Errorf("expected level %q, got %v", c.level, obj["level"])
		}
	}
}

func TestLogrusFatalDoesNotExit(t *testing.T) {
	// If the wrapper didn't neutralize ExitFunc the test process would
	// terminate; reaching the assertion is success.
	log, buf := newLogger(lllogrus.Config{})
	log.Fatal("survives")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["level"] != "fatal" {
		t.Errorf("expected fatal entry, got %v", obj["level"])
	}
}

func TestLogrusOriginalExitFuncNotMutated(t *testing.T) {
	// User-supplied logger keeps its original ExitFunc — only the wrapper copy
	// is neutralized.
	called := false
	user := logrusbase.New()
	user.Out = &bytes.Buffer{}
	user.Formatter = &logrusbase.JSONFormatter{}
	user.ExitFunc = func(int) { called = true }

	tr := lllogrus.New(lllogrus.Config{
		BaseConfig: transport.BaseConfig{ID: "logrus"},
		Logger:     user,
	})
	log := loglayer.New(loglayer.Config{DisableFatalExit: true, Transport: tr})
	log.Fatal("via wrapper")

	if called {
		t.Error("user's ExitFunc was invoked — wrapper should isolate it")
	}
}

func TestLogrusMapMetadataMerged(t *testing.T) {
	log, buf := newLogger(lllogrus.Config{})
	log.WithMetadata(loglayer.Metadata{"requestId": "xyz", "n": 42}).Info("req")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["requestId"] != "xyz" {
		t.Errorf("requestId: got %v", obj["requestId"])
	}
	if obj["n"] != float64(42) {
		t.Errorf("n: got %v", obj["n"])
	}
}

func TestLogrusStructMetadataNested(t *testing.T) {
	type user struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	log, buf := newLogger(lllogrus.Config{})
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

func TestLogrusCustomMetadataFieldName(t *testing.T) {
	type user struct {
		ID int `json:"id"`
	}
	log, buf := newLogger(lllogrus.Config{MetadataFieldName: "user"})
	log.WithMetadata(user{ID: 9}).Info("hi")
	obj := transporttest.ParseJSONLine(t, buf)
	if _, ok := obj["user"].(map[string]any); !ok {
		t.Errorf("expected 'user' key, got %v", obj)
	}
}

func TestLogrusContextMerged(t *testing.T) {
	log, buf := newLogger(lllogrus.Config{})
	log = log.WithFields(loglayer.Fields{"service": "api"})
	log.Info("ctx test")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["service"] != "api" {
		t.Errorf("service: got %v", obj["service"])
	}
}

func TestLogrusWithError(t *testing.T) {
	log, buf := newLogger(lllogrus.Config{})
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

func TestLogrusLevelFiltering(t *testing.T) {
	buf := &bytes.Buffer{}
	base := logrusbase.New()
	base.Out = buf
	base.Formatter = &logrusbase.JSONFormatter{}
	base.Level = logrusbase.TraceLevel
	tr := lllogrus.New(lllogrus.Config{
		BaseConfig: transport.BaseConfig{ID: "logrus", Level: loglayer.LogLevelError},
		Logger:     base,
	})
	log := loglayer.New(loglayer.Config{DisableFatalExit: true, Transport: tr})
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

func TestLogrusGetLoggerInstance(t *testing.T) {
	base := logrusbase.New()
	base.Out = &bytes.Buffer{}
	tr := lllogrus.New(lllogrus.Config{
		BaseConfig: transport.BaseConfig{ID: "logrus"},
		Logger:     base,
	})
	log := loglayer.New(loglayer.Config{DisableFatalExit: true, Transport: tr})
	if _, ok := log.GetLoggerInstance("logrus").(*logrusbase.Logger); !ok {
		t.Errorf("expected *logrus.Logger, got %T", log.GetLoggerInstance("logrus"))
	}
}
