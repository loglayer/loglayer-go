package loglayer_test

import (
	"testing"

	"go.uber.org/goleak"
)

// TestMain wraps the root-package suite in goleak.VerifyTestMain so any
// goroutine spawned by core dispatch, plugin pipelines, or
// concurrency_test.go hot-swap tests must be reaped before TestMain
// returns. The core itself doesn't spawn goroutines on the hot path;
// this catches regressions if that contract changes.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
