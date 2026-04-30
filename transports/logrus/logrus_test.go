package logrus_test

import (
	"bytes"
	"testing"

	logrusbase "github.com/sirupsen/logrus"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
	"go.loglayer.dev/transport/transporttest"
	lllogrus "go.loglayer.dev/transports/logrus"
)

func factory(opts transporttest.FactoryOpts) (*loglayer.LogLayer, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	base := logrusbase.New()
	base.Out = buf
	base.Formatter = &logrusbase.JSONFormatter{}
	base.Level = logrusbase.TraceLevel
	tr := lllogrus.New(lllogrus.Config{
		BaseConfig: transport.BaseConfig{ID: "logrus", Level: opts.Level},
		Logger:     base,
	})
	return transporttest.NewLogger(tr, opts), buf
}

func TestLogrusContract(t *testing.T) {
	transporttest.RunContract(t, transporttest.ContractCase{
		Name:    "logrus",
		Factory: factory,
		Expect: transporttest.Expectations{
			MessageKey: "msg",
			LevelKey:   "level",
			Levels: map[loglayer.LogLevel]string{
				loglayer.LogLevelTrace: "trace",
				loglayer.LogLevelDebug: "debug",
				loglayer.LogLevelInfo:  "info",
				loglayer.LogLevelWarn:  "warning", // logrus uses "warning" not "warn"
				loglayer.LogLevelError: "error",
				loglayer.LogLevelFatal: "fatal",
				loglayer.LogLevelPanic: "panic",
			},
		},
	})
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
