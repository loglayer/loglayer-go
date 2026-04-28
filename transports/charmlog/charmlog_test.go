package charmlog_test

import (
	"bytes"
	"testing"

	clog "github.com/charmbracelet/log"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
	"go.loglayer.dev/transport/transporttest"
	llcharm "go.loglayer.dev/transports/charmlog"
)

func factory(opts transporttest.FactoryOpts) (*loglayer.LogLayer, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	cl := clog.NewWithOptions(buf, clog.Options{
		Level:           clog.DebugLevel,
		ReportTimestamp: false,
		Formatter:       clog.JSONFormatter,
	})
	cfg := llcharm.Config{
		BaseConfig:        transport.BaseConfig{ID: "charmlog", Level: opts.Level},
		Logger:            cl,
		MetadataFieldName: opts.MetadataFieldName,
	}
	tr := llcharm.New(cfg)
	return loglayer.New(loglayer.Config{DisableFatalExit: true, Transport: tr}), buf
}

func TestCharmContract(t *testing.T) {
	transporttest.RunContract(t, transporttest.ContractCase{
		Name:    "charmlog",
		Factory: factory,
		Expect: transporttest.Expectations{
			MessageKey: "msg",
			LevelKey:   "level",
			Levels: map[loglayer.LogLevel]string{
				// Trace omitted: charmbracelet/log has no Trace; tested separately.
				loglayer.LogLevelDebug: "debug",
				loglayer.LogLevelInfo:  "info",
				loglayer.LogLevelWarn:  "warn",
				loglayer.LogLevelError: "error",
				loglayer.LogLevelFatal: "fatal",
			},
		},
	})
}

func TestCharmTraceMapsToDebug(t *testing.T) {
	log, buf := factory(transporttest.FactoryOpts{})
	log.Trace("trace msg")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["level"] != "debug" {
		t.Errorf("trace should map to debug in charmbracelet/log, got %v", obj["level"])
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
