package zap_test

import (
	"bytes"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.loglayer.dev"
	"go.loglayer.dev/internal/transporttest"
	"go.loglayer.dev/transport"
	llzap "go.loglayer.dev/transports/zap"
)

func factory(opts transporttest.FactoryOpts) (*loglayer.LogLayer, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	enc := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	core := zapcore.NewCore(enc, zapcore.AddSync(buf), zapcore.DebugLevel)
	cfg := llzap.Config{
		BaseConfig:        transport.BaseConfig{ID: "zap", Level: opts.Level},
		Logger:            zap.New(core),
		MetadataFieldName: opts.MetadataFieldName,
	}
	tr := llzap.New(cfg)
	return loglayer.New(loglayer.Config{DisableFatalExit: true, Transport: tr}), buf
}

func TestZapContract(t *testing.T) {
	transporttest.RunContract(t, transporttest.ContractCase{
		Name:    "zap",
		Factory: factory,
		Expect: transporttest.Expectations{
			MessageKey: "msg",
			LevelKey:   "level",
			Levels: map[loglayer.LogLevel]string{
				// Trace omitted: zap maps Trace to Debug; tested separately.
				loglayer.LogLevelDebug: "debug",
				loglayer.LogLevelInfo:  "info",
				loglayer.LogLevelWarn:  "warn",
				loglayer.LogLevelError: "error",
				loglayer.LogLevelFatal: "fatal",
			},
		},
	})
}

func TestZapTraceMapsToDebug(t *testing.T) {
	log, buf := factory(transporttest.FactoryOpts{})
	log.Trace("trace msg")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["level"] != "debug" {
		t.Errorf("trace should map to debug in zap, got %v", obj["level"])
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
