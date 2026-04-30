package phuslu_test

import (
	"bytes"
	"testing"

	plog "github.com/phuslu/log"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
	"go.loglayer.dev/transport/transporttest"
	llphuslu "go.loglayer.dev/transports/phuslu"
)

func factory(opts transporttest.FactoryOpts) (*loglayer.LogLayer, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	pl := &plog.Logger{
		Level:  plog.TraceLevel,
		Writer: &plog.IOWriter{Writer: buf},
	}
	tr := llphuslu.New(llphuslu.Config{
		BaseConfig: transport.BaseConfig{ID: "phuslu", Level: opts.Level},
		Logger:     pl,
	})
	return transporttest.NewLogger(tr, opts), buf
}

func TestPhusluContract(t *testing.T) {
	transporttest.RunContract(t, transporttest.ContractCase{
		Name:    "phuslu",
		Factory: factory,
		Expect: transporttest.Expectations{
			MessageKey: "message",
			LevelKey:   "level",
			Levels: map[loglayer.LogLevel]string{
				loglayer.LogLevelTrace: "trace",
				loglayer.LogLevelDebug: "debug",
				loglayer.LogLevelInfo:  "info",
				loglayer.LogLevelWarn:  "warn",
				loglayer.LogLevelError: "error",
				loglayer.LogLevelPanic: "panic", // phuslu's panic() is recoverable
			},
			SkipFatal: true, // phuslu/log always os.Exits on Fatal
		},
	})
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
