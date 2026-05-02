package slog_test

import (
	"bytes"
	"context"
	stdlibslog "log/slog"
	"testing"

	llslog "go.loglayer.dev/transports/slog/v2"
	"go.loglayer.dev/v2"
	"go.loglayer.dev/v2/transport"
	"go.loglayer.dev/v2/transport/transporttest"
)

func factory(opts transporttest.FactoryOpts) (*loglayer.LogLayer, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	logger := stdlibslog.New(stdlibslog.NewJSONHandler(buf, &stdlibslog.HandlerOptions{
		Level: stdlibslog.LevelDebug - 4, // allow LogLevelTrace through
	}))
	tr := llslog.New(llslog.Config{
		BaseConfig: transport.BaseConfig{ID: "slog", Level: opts.Level},
		Logger:     logger,
	})
	return transporttest.NewLogger(tr, opts), buf
}

func TestSlogContract(t *testing.T) {
	transporttest.RunContract(t, transporttest.ContractCase{
		Name:    "slog",
		Factory: factory,
		Expect: transporttest.Expectations{
			MessageKey: "msg",
			LevelKey:   "level",
			Levels: map[loglayer.LogLevel]string{
				// slog has no Trace/Fatal/Panic; they render as DEBUG-/ERROR+ offsets.
				loglayer.LogLevelTrace: "DEBUG-4",
				loglayer.LogLevelDebug: "DEBUG",
				loglayer.LogLevelInfo:  "INFO",
				loglayer.LogLevelWarn:  "WARN",
				loglayer.LogLevelError: "ERROR",
				loglayer.LogLevelFatal: "ERROR+4",
				loglayer.LogLevelPanic: "ERROR+8",
			},
		},
	})
}

func TestSlogWithContextPassedToHandler(t *testing.T) {
	// Slog is the one wrapper that *does* forward ctx to the underlying
	// handler. This positive test verifies the forwarding; the contract's
	// WithContextDoesNotBreakDispatch covers the negative shape on the others.
	type ctxKey struct{}
	ctx := context.WithValue(context.Background(), ctxKey{}, "trace-id-123")

	var captured context.Context
	handler := &spyHandler{onHandle: func(c context.Context, _ stdlibslog.Record) error {
		captured = c
		return nil
	}}
	logger := stdlibslog.New(handler)
	tr := llslog.New(llslog.Config{
		BaseConfig: transport.BaseConfig{ID: "slog"},
		Logger:     logger,
	})
	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})

	log.WithContext(ctx).Info("with ctx")
	if captured == nil || captured.Value(ctxKey{}) != "trace-id-123" {
		t.Errorf("Ctx not propagated to slog handler: %v", captured)
	}
}

func TestSlogGetLoggerInstance(t *testing.T) {
	logger := stdlibslog.New(stdlibslog.NewJSONHandler(&bytes.Buffer{}, nil))
	tr := llslog.New(llslog.Config{
		BaseConfig: transport.BaseConfig{ID: "slog"},
		Logger:     logger,
	})
	log := loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true})
	if _, ok := log.GetLoggerInstance("slog").(*stdlibslog.Logger); !ok {
		t.Errorf("expected *slog.Logger, got %T", log.GetLoggerInstance("slog"))
	}
}

// spyHandler captures the context.Context handed to Handle so tests can
// verify Ctx propagation end-to-end.
type spyHandler struct {
	onHandle func(context.Context, stdlibslog.Record) error
}

func (h *spyHandler) Enabled(_ context.Context, _ stdlibslog.Level) bool { return true }
func (h *spyHandler) Handle(ctx context.Context, r stdlibslog.Record) error {
	return h.onHandle(ctx, r)
}
func (h *spyHandler) WithAttrs(_ []stdlibslog.Attr) stdlibslog.Handler { return h }
func (h *spyHandler) WithGroup(_ string) stdlibslog.Handler            { return h }
