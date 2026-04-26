package charmlog_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	clog "github.com/charmbracelet/log"

	"go.loglayer.dev/loglayer"
	"go.loglayer.dev/loglayer/internal/transporttest"
	"go.loglayer.dev/loglayer/transport"
	llcharm "go.loglayer.dev/loglayer/transports/charmlog"
)

func newLogger(cfg llcharm.Config) (*loglayer.LogLayer, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	cl := clog.NewWithOptions(buf, clog.Options{
		Level:           clog.DebugLevel,
		ReportTimestamp: false,
		Formatter:       clog.JSONFormatter,
	})
	cfg.Logger = cl
	if cfg.BaseConfig.ID == "" {
		cfg.BaseConfig.ID = "charmlog"
	}
	t := llcharm.New(cfg)
	return loglayer.New(loglayer.Config{DisableFatalExit: true, Transport: t}), buf
}


func TestCharmSimpleMessage(t *testing.T) {
	log, buf := newLogger(llcharm.Config{})
	log.Info("hello")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["msg"] != "hello" {
		t.Errorf("msg: got %v", obj["msg"])
	}
	if obj["level"] != "info" {
		t.Errorf("level: got %v", obj["level"])
	}
}

func TestCharmMultipleMessages(t *testing.T) {
	log, buf := newLogger(llcharm.Config{})
	log.Info("part1", "part2")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["msg"] != "part1 part2" {
		t.Errorf("msg: got %v", obj["msg"])
	}
}

func TestCharmLevels(t *testing.T) {
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
		log, buf := newLogger(llcharm.Config{})
		c.fn(log)
		obj := transporttest.ParseJSONLine(t, buf)
		if obj["level"] != c.level {
			t.Errorf("expected level %q, got %v", c.level, obj["level"])
		}
	}
}

func TestCharmTraceMapsToDebug(t *testing.T) {
	log, buf := newLogger(llcharm.Config{})
	log.Trace("trace msg")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["level"] != "debug" {
		t.Errorf("trace should map to debug in charmbracelet/log, got %v", obj["level"])
	}
}

func TestCharmFatalDoesNotExit(t *testing.T) {
	log, buf := newLogger(llcharm.Config{})
	log.Fatal("survives")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["level"] != "fatal" {
		t.Errorf("expected fatal entry, got %v", obj["level"])
	}
}

func TestCharmMapMetadataMerged(t *testing.T) {
	log, buf := newLogger(llcharm.Config{})
	log.WithMetadata(loglayer.Metadata{"requestId": "xyz", "n": 42}).Info("req")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["requestId"] != "xyz" {
		t.Errorf("requestId: got %v", obj["requestId"])
	}
	if obj["n"] != float64(42) {
		t.Errorf("n: got %v", obj["n"])
	}
}

func TestCharmContextMerged(t *testing.T) {
	log, buf := newLogger(llcharm.Config{})
	log = log.WithFields(loglayer.Fields{"service": "api"})
	log.Info("ctx test")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["service"] != "api" {
		t.Errorf("service: got %v", obj["service"])
	}
}

func TestCharmWithError(t *testing.T) {
	log, buf := newLogger(llcharm.Config{})
	log.WithError(errors.New("boom")).Error("failed")
	obj := transporttest.ParseJSONLine(t, buf)
	// charmbracelet/log emits the err map nested under "err"
	got, ok := obj["err"]
	if !ok {
		t.Fatalf("expected err field, got %v", obj)
	}
	// JSON formatter renders the map either as nested object or as a string
	// depending on version — accept either.
	switch v := got.(type) {
	case map[string]any:
		if v["message"] != "boom" {
			t.Errorf("err.message: got %v", v["message"])
		}
	case string:
		if !strings.Contains(v, "boom") {
			t.Errorf("err string: got %v", v)
		}
	default:
		t.Errorf("unexpected err type %T", v)
	}
}

func TestCharmLevelFiltering(t *testing.T) {
	buf := &bytes.Buffer{}
	cl := clog.NewWithOptions(buf, clog.Options{
		Level:     clog.DebugLevel,
		Formatter: clog.JSONFormatter,
	})
	tr := llcharm.New(llcharm.Config{
		BaseConfig: transport.BaseConfig{ID: "charmlog", Level: loglayer.LogLevelError},
		Logger:     cl,
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

func TestCharmGetLoggerInstance(t *testing.T) {
	cl := clog.New(&bytes.Buffer{})
	tr := llcharm.New(llcharm.Config{
		BaseConfig: transport.BaseConfig{ID: "charmlog"},
		Logger:     cl,
	})
	log := loglayer.New(loglayer.Config{DisableFatalExit: true, Transport: tr})
	if _, ok := log.GetLoggerInstance("charmlog").(*clog.Logger); !ok {
		t.Errorf("expected *charmbracelet/log.Logger, got %T", log.GetLoggerInstance("charmlog"))
	}
}
