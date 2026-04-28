package loglayer_test

import (
	"testing"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
	lltest "go.loglayer.dev/transports/testing"
)

// A wrapped DataHook plugin must NOT register for unrelated hooks. If
// it did, a chain like [wrapped(DataHook), realMetadataHook] would have
// the wrapper's pass-through OnMetadataCalled run first; with nil input
// it'd return nil and short-circuit the rest of the chain.
func TestWithErrorReporter_DoesNotInterfereWithUnrelatedHooks(t *testing.T) {
	lib := &lltest.TestLoggingLibrary{}
	tr := lltest.New(lltest.Config{
		BaseConfig: transport.BaseConfig{ID: "test"},
		Library:    lib,
	})

	calledMeta := false
	wrapped := loglayer.WithErrorReporter(
		loglayer.NewDataHook("data-only", func(p loglayer.BeforeDataOutParams) loglayer.Data {
			return nil
		}),
		func(error) {},
	)
	metaHook := loglayer.NewMetadataHook("real-meta", func(m any) any {
		calledMeta = true
		return loglayer.Metadata{"injected": true}
	})

	log := loglayer.New(loglayer.Config{
		Transport:        tr,
		DisableFatalExit: true,
		Plugins:          []loglayer.Plugin{wrapped, metaHook},
	})

	log.WithMetadata(nil).Info("test")
	if !calledMeta {
		t.Fatal("metadata hook should run; wrapper must not register for unrelated hooks")
	}
}
