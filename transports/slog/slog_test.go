package slog_test

import (
	"bytes"
	"context"
	stdlibslog "log/slog"
	"testing"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
	"go.loglayer.dev/transport/transporttest"
	llslog "go.loglayer.dev/transports/slog"
)

func factory(opts transporttest.FactoryOpts) (*loglayer.LogLayer, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	logger := stdlibslog.New(stdlibslog.NewJSONHandler(buf, &stdlibslog.HandlerOptions{
		Level: stdlibslog.LevelDebug,
	}))
	cfg := llslog.Config{
		BaseConfig:        transport.BaseConfig{ID: "slog", Level: opts.Level},
		Logger:            logger,
		MetadataFieldName: opts.MetadataFieldName,
	}
	tr := llslog.New(cfg)
	return loglayer.New(loglayer.Config{Transport: tr, DisableFatalExit: true}), buf
}

func TestSlogContract(t *testing.T) {
	transporttest.RunContract(t, transporttest.ContractCase{
		Name:    "slog",
		Factory: factory,
		Expect: transporttest.Expectations{
			MessageKey: "msg",
			LevelKey:   "level",
			Levels: map[loglayer.LogLevel]string{
				// Trace omitted: slog has no native Trace; tested separately
				// (maps to DEBUG). Fatal omitted: slog renders custom levels
				// as "ERROR+N"; covered by TestSlogFatalAboveError.
				loglayer.LogLevelDebug: "DEBUG",
				loglayer.LogLevelInfo:  "INFO",
				loglayer.LogLevelWarn:  "WARN",
				loglayer.LogLevelError: "ERROR",
			},
		},
	})
}

func TestSlogTraceMapsToDebug(t *testing.T) {
	log, buf := factory(transporttest.FactoryOpts{})
	log.Trace("trace msg")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["level"] != "DEBUG" {
		t.Errorf("trace should map to DEBUG in slog, got %v", obj["level"])
	}
}

func TestSlogFatalAboveError(t *testing.T) {
	log, buf := factory(transporttest.FactoryOpts{})
	log.Fatal("survives")
	obj := transporttest.ParseJSONLine(t, buf)
	// slog renders custom levels as "ERROR+N".
	if obj["level"] != "ERROR+4" {
		t.Errorf("expected ERROR+4 for fatal, got %v", obj["level"])
	}
}

func TestSlogWithCtxPassedToHandler(t *testing.T) {
	// Slog is the one wrapper that *does* forward ctx to the underlying
	// handler. This positive test verifies the forwarding; the contract's
	// WithCtxDoesNotBreakDispatch covers the negative shape on the others.
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

	log.WithCtx(ctx).Info("with ctx")
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
