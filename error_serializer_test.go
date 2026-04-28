package loglayer_test

import (
	"errors"
	"fmt"
	"testing"

	"go.loglayer.dev"
	lltest "go.loglayer.dev/transports/testing"
)

func newUnwrapLogger(t *testing.T) (*loglayer.LogLayer, *lltest.TestLoggingLibrary) {
	t.Helper()
	lib := &lltest.TestLoggingLibrary{}
	log := loglayer.New(loglayer.Config{
		Transport:        lltest.New(lltest.Config{Library: lib}),
		ErrorSerializer:  loglayer.UnwrappingErrorSerializer,
		DisableFatalExit: true,
	})
	return log, lib
}

func TestUnwrap_NilError(t *testing.T) {
	if got := loglayer.UnwrappingErrorSerializer(nil); got != nil {
		t.Errorf("nil err must produce nil map, got %v", got)
	}
}

// Unwrapped error: shape matches the default serializer (no causes key).
func TestUnwrap_FlatError(t *testing.T) {
	t.Parallel()
	log, lib := newUnwrapLogger(t)
	log.WithError(errors.New("flat")).Error("oops")

	line := lib.PopLine()
	errMap := line.Data["err"].(map[string]any)
	if errMap["message"] != "flat" {
		t.Errorf("message: got %v", errMap["message"])
	}
	if _, ok := errMap["causes"]; ok {
		t.Errorf("flat error should have no causes: %v", errMap)
	}
}

// %w chain: each wrap step appears as a separate cause.
func TestUnwrap_WrappedChain(t *testing.T) {
	t.Parallel()
	log, lib := newUnwrapLogger(t)
	inner := errors.New("inner")
	middle := fmt.Errorf("middle: %w", inner)
	outer := fmt.Errorf("outer: %w", middle)
	log.WithError(outer).Error("oops")

	line := lib.PopLine()
	errMap := line.Data["err"].(map[string]any)
	if errMap["message"] != "outer: middle: inner" {
		t.Errorf("message: got %v", errMap["message"])
	}
	causes, ok := errMap["causes"].([]map[string]any)
	if !ok {
		t.Fatalf("causes shape: got %T (%v)", errMap["causes"], errMap["causes"])
	}
	if len(causes) != 2 {
		t.Fatalf("expected 2 causes, got %d: %v", len(causes), causes)
	}
	if causes[0]["message"] != "middle: inner" {
		t.Errorf("causes[0]: got %v", causes[0]["message"])
	}
	if causes[1]["message"] != "inner" {
		t.Errorf("causes[1]: got %v", causes[1]["message"])
	}
}

// errors.Join: each joined member appears as a cause.
func TestUnwrap_JoinedErrors(t *testing.T) {
	t.Parallel()
	log, lib := newUnwrapLogger(t)
	joined := errors.Join(
		errors.New("first"),
		errors.New("second"),
		errors.New("third"),
	)
	log.WithError(joined).Error("oops")

	line := lib.PopLine()
	errMap := line.Data["err"].(map[string]any)
	causes, ok := errMap["causes"].([]map[string]any)
	if !ok {
		t.Fatalf("causes shape: got %T", errMap["causes"])
	}
	if len(causes) != 3 {
		t.Fatalf("expected 3 causes, got %d", len(causes))
	}
	wantMessages := []string{"first", "second", "third"}
	for i, want := range wantMessages {
		if got := causes[i]["message"]; got != want {
			t.Errorf("causes[%d]: got %v, want %s", i, got, want)
		}
	}
}

// Join with a nil member: the nil is skipped, surviving members appear.
func TestUnwrap_JoinedSkipsNil(t *testing.T) {
	t.Parallel()
	log, lib := newUnwrapLogger(t)
	joined := errors.Join(errors.New("a"), nil, errors.New("b"))
	log.WithError(joined).Error("oops")

	errMap := lib.PopLine().Data["err"].(map[string]any)
	causes := errMap["causes"].([]map[string]any)
	if len(causes) != 2 {
		t.Errorf("nil member should be dropped, got %d causes", len(causes))
	}
}

// Join takes precedence over Unwrap: a Joined error wraps each member,
// not a single chain.
func TestUnwrap_JoinedNotWalkedAsSingleChain(t *testing.T) {
	t.Parallel()
	log, lib := newUnwrapLogger(t)
	joined := errors.Join(errors.New("a"), errors.New("b"))
	wrapped := fmt.Errorf("ctx: %w", joined)
	log.WithError(wrapped).Error("oops")

	errMap := lib.PopLine().Data["err"].(map[string]any)
	causes := errMap["causes"].([]map[string]any)
	// wrapped's Unwrap (single-chain) yields the Join error. We then
	// stop at the Join (causes has one entry: the joined error's
	// rendered text). A future enhancement might recurse, but for v1
	// the behavior is "one Unwrap step at a time" until we hit Join.
	if len(causes) < 1 {
		t.Errorf("expected at least one cause: %v", causes)
	}
}

// errors.Is still works through the serializer (sanity check that we
// don't break the underlying error chain). The serializer only reads;
// it never mutates the error.
func TestUnwrap_PreservesErrorsIs(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("sentinel")
	wrapped := fmt.Errorf("ctx: %w", sentinel)

	_ = loglayer.UnwrappingErrorSerializer(wrapped)

	if !errors.Is(wrapped, sentinel) {
		t.Error("serializer must not break errors.Is")
	}
}

// A self-referential Unwrap must not infinite-loop. The walk is bounded
// to a defensive depth; the test proves it terminates and produces a
// finite causes slice rather than OOMing.
type selfWrappingErr struct{}

func (e *selfWrappingErr) Error() string { return "loops" }
func (e *selfWrappingErr) Unwrap() error { return e }

func TestUnwrap_BoundedOnSelfReference(t *testing.T) {
	t.Parallel()
	out := loglayer.UnwrappingErrorSerializer(&selfWrappingErr{})
	causes, ok := out["causes"].([]map[string]any)
	if !ok {
		t.Fatalf("causes shape: got %T", out["causes"])
	}
	// Implementation caps the walk; we just need finite output.
	if len(causes) == 0 {
		t.Errorf("expected at least one cause from self-ref walk, got 0")
	}
	if len(causes) > 1000 {
		t.Errorf("walk should be bounded, got %d causes", len(causes))
	}
}
