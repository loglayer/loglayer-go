package zap_test

import (
	"bytes"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
	"go.loglayer.dev/transport/transporttest"
	llzap "go.loglayer.dev/transports/zap"
)

func factory(opts transporttest.FactoryOpts) (*loglayer.LogLayer, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	enc := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	core := zapcore.NewCore(enc, zapcore.AddSync(buf), zapcore.DebugLevel)
	tr := llzap.New(llzap.Config{
		BaseConfig: transport.BaseConfig{ID: "zap", Level: opts.Level},
		Logger:     zap.New(core),
	})
	return transporttest.NewLogger(tr, opts), buf
}

func TestZapContract(t *testing.T) {
	transporttest.RunContract(t, transporttest.ContractCase{
		Name:    "zap",
		Factory: factory,
		Expect: transporttest.Expectations{
			MessageKey: "msg",
			LevelKey:   "level",
			Levels: map[loglayer.LogLevel]string{
				loglayer.LogLevelTrace: "debug", // zap has no Trace; mapped to lowest
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
