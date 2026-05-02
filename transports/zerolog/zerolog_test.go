package zerolog_test

import (
	"bytes"
	"testing"

	zlog "github.com/rs/zerolog"

	llzero "go.loglayer.dev/transports/zerolog/v2"
	"go.loglayer.dev/v2"
	"go.loglayer.dev/v2/transport"
	"go.loglayer.dev/v2/transport/transporttest"
)

func factory(opts transporttest.FactoryOpts) (*loglayer.LogLayer, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	z := zlog.New(buf).Level(zlog.TraceLevel)
	tr := llzero.New(llzero.Config{
		BaseConfig: transport.BaseConfig{ID: "zerolog", Level: opts.Level},
		Logger:     &z,
	})
	return transporttest.NewLogger(tr, opts), buf
}

func TestZerologContract(t *testing.T) {
	transporttest.RunContract(t, transporttest.ContractCase{
		Name:    "zerolog",
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
				loglayer.LogLevelFatal: "fatal",
				loglayer.LogLevelPanic: "panic",
			},
		},
	})
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
