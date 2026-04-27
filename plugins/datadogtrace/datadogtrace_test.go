package datadogtrace_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"go.loglayer.dev"
	"go.loglayer.dev/plugins/datadogtrace"
	"go.loglayer.dev/transport"
	lltest "go.loglayer.dev/transports/testing"
)

// fakeExtract returns a deterministic extractor for tests.
func fakeExtract(traceID, spanID uint64, ok bool) func(context.Context) (uint64, uint64, bool) {
	return func(context.Context) (uint64, uint64, bool) {
		return traceID, spanID, ok
	}
}

func setup(t *testing.T, plugin loglayer.Plugin) (*loglayer.LogLayer, *lltest.TestLoggingLibrary) {
	t.Helper()
	lib := &lltest.TestLoggingLibrary{}
	tr := lltest.New(lltest.Config{BaseConfig: transport.BaseConfig{ID: "test"}, Library: lib})
	log := loglayer.New(loglayer.Config{
		Transport:        tr,
		DisableFatalExit: true,
		Plugins:          []loglayer.Plugin{plugin},
	})
	return log, lib
}

func TestDatadogTrace_InjectsIDsWhenSpanPresent(t *testing.T) {
	log, lib := setup(t, datadogtrace.New(datadogtrace.Config{
		Extract: fakeExtract(0xDEADBEEF, 0xCAFE, true),
	}))

	log.WithCtx(context.Background()).Info("served")
	line := lib.PopLine()
	if line.Data["dd.trace_id"] != "3735928559" {
		t.Errorf("dd.trace_id: got %v, want 3735928559", line.Data["dd.trace_id"])
	}
	if line.Data["dd.span_id"] != "51966" {
		t.Errorf("dd.span_id: got %v, want 51966", line.Data["dd.span_id"])
	}
}

func TestDatadogTrace_NoCtxNoInjection(t *testing.T) {
	log, lib := setup(t, datadogtrace.New(datadogtrace.Config{
		Extract: fakeExtract(1, 1, true),
	}))

	log.Info("no context attached") // no WithCtx
	line := lib.PopLine()
	if _, has := line.Data["dd.trace_id"]; has {
		t.Errorf("trace_id should be absent when no Ctx: %v", line.Data)
	}
}

func TestDatadogTrace_NoSpanNoInjection(t *testing.T) {
	log, lib := setup(t, datadogtrace.New(datadogtrace.Config{
		Extract: fakeExtract(0, 0, false),
	}))

	log.WithCtx(context.Background()).Info("ctx but no span")
	line := lib.PopLine()
	if _, has := line.Data["dd.trace_id"]; has {
		t.Errorf("trace_id should be absent when no span: %v", line.Data)
	}
}

func TestDatadogTrace_OptionalReservedAttributes(t *testing.T) {
	log, lib := setup(t, datadogtrace.New(datadogtrace.Config{
		Service: "checkout-api",
		Env:     "prod",
		Version: "1.2.3",
		Extract: fakeExtract(7, 11, true),
	}))

	log.WithCtx(context.Background()).Info("hi")
	line := lib.PopLine()
	if line.Data["dd.service"] != "checkout-api" {
		t.Errorf("dd.service: got %v", line.Data["dd.service"])
	}
	if line.Data["dd.env"] != "prod" {
		t.Errorf("dd.env: got %v", line.Data["dd.env"])
	}
	if line.Data["dd.version"] != "1.2.3" {
		t.Errorf("dd.version: got %v", line.Data["dd.version"])
	}
}

func TestDatadogTrace_OmitsEmptyOptionalAttributes(t *testing.T) {
	log, lib := setup(t, datadogtrace.New(datadogtrace.Config{
		// Service/Env/Version unset
		Extract: fakeExtract(7, 11, true),
	}))

	log.WithCtx(context.Background()).Info("hi")
	line := lib.PopLine()
	for _, k := range []string{"dd.service", "dd.env", "dd.version"} {
		if _, has := line.Data[k]; has {
			t.Errorf("%s should be omitted when empty: %v", k, line.Data[k])
		}
	}
}

func TestDatadogTrace_PreservesUserData(t *testing.T) {
	log, lib := setup(t, datadogtrace.New(datadogtrace.Config{
		Extract: fakeExtract(7, 11, true),
	}))

	log = log.WithFields(loglayer.Fields{"requestId": "abc-123"})
	log.WithCtx(context.Background()).WithMetadata(loglayer.Metadata{"durationMs": 42}).Info("served")
	line := lib.PopLine()

	if line.Data["requestId"] != "abc-123" {
		t.Errorf("user field should pass through: %v", line.Data)
	}
	if line.Data["dd.trace_id"] != "7" {
		t.Errorf("trace_id: got %v", line.Data["dd.trace_id"])
	}
	if m, _ := line.Metadata.(loglayer.Metadata); m["durationMs"] != 42 {
		t.Errorf("metadata should pass through: %v", line.Metadata)
	}
}

func TestDatadogTrace_ExtractorPanicCallsOnError(t *testing.T) {
	var caught error
	plugin := datadogtrace.New(datadogtrace.Config{
		Extract: func(context.Context) (uint64, uint64, bool) {
			panic(errors.New("boom"))
		},
		OnError: func(err error) { caught = err },
	})
	log, lib := setup(t, plugin)

	log.WithCtx(context.Background()).Info("recovered") // must not panic
	if caught == nil {
		t.Fatal("OnError should have been called with the panic value")
	}
	// The framework wraps the panic via %w; the original "boom" should
	// still surface in the error message.
	if !strings.Contains(caught.Error(), "boom") {
		t.Errorf("recovered error should mention the original panic value: got %q", caught.Error())
	}
	line := lib.PopLine()
	if line == nil {
		t.Fatal("entry should still be emitted after panic recovery")
	}
	if _, has := line.Data["dd.trace_id"]; has {
		t.Errorf("trace_id should be absent on extractor panic: %v", line.Data)
	}
}

func TestDatadogTrace_ExtractorPanicWithoutOnErrorIsSilent(t *testing.T) {
	plugin := datadogtrace.New(datadogtrace.Config{
		Extract: func(context.Context) (uint64, uint64, bool) {
			panic("string-panic")
		},
		// OnError nil
	})
	log, lib := setup(t, plugin)

	// Must not panic.
	log.WithCtx(context.Background()).Info("recovered")
	if lib.Len() != 1 {
		t.Fatal("entry should still emit when OnError is nil")
	}
}

func TestDatadogTrace_PanicsWithoutExtract(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when Extract is nil")
		}
	}()
	datadogtrace.New(datadogtrace.Config{})
}

func TestDatadogTrace_DefaultID(t *testing.T) {
	p := datadogtrace.New(datadogtrace.Config{Extract: fakeExtract(1, 1, true)})
	if p.ID != "datadog-trace-injector" {
		t.Errorf("default ID: got %q, want %q", p.ID, "datadog-trace-injector")
	}
}

func TestDatadogTrace_CustomID(t *testing.T) {
	p := datadogtrace.New(datadogtrace.Config{
		ID:      "my-injector",
		Extract: fakeExtract(1, 1, true),
	})
	if p.ID != "my-injector" {
		t.Errorf("custom ID: got %q", p.ID)
	}
}
