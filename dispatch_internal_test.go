package loglayer

import (
	"testing"
	"time"
)

// stubTransport is a minimal Transport implementation for in-package tests
// that can't import sub-packages (which would cause an import cycle).
type stubTransport struct {
	id     string
	closes int
}

func (s *stubTransport) ID() string                   { return s.id }
func (s *stubTransport) IsEnabled() bool              { return true }
func (s *stubTransport) SetEnabled(bool)              {}
func (s *stubTransport) MinLevel() LogLevel           { return LogLevelDebug }
func (s *stubTransport) ShouldProcess(LogLevel) bool  { return true }
func (s *stubTransport) SendToLogger(TransportParams) {}
func (s *stubTransport) GetLoggerInstance() any       { return nil }
func (s *stubTransport) Close() error                 { s.closes++; return nil }

// TestFatal_FlushesBeforeExit asserts that a fatal-level dispatch closes
// every transport (draining HTTP/Datadog queues) before the os.Exit call.
// We swap the package-level osExit hook so the test can observe both the
// flush and the exit code without terminating the test runner.
func TestFatal_FlushesBeforeExit(t *testing.T) {
	exitCalled := -1
	orig := osExit
	osExit = func(code int) { exitCalled = code }
	defer func() { osExit = orig }()

	tr := &stubTransport{id: "t"}
	log := New(Config{Transport: tr}) // DisableFatalExit NOT set

	log.Fatal("goodbye")

	if tr.closes != 1 {
		t.Errorf("Fatal should close every transport once, got %d", tr.closes)
	}
	if exitCalled != 1 {
		t.Errorf("Fatal should call osExit(1), got %d", exitCalled)
	}
}

// hangingTransport's Close blocks until released. Used to verify the
// flush-with-timeout path: a wedged endpoint must not hang Fatal or
// any transport mutator forever.
type hangingTransport struct {
	stubTransport
	release chan struct{}
}

func newHanging(id string) *hangingTransport {
	return &hangingTransport{
		stubTransport: stubTransport{id: id},
		release:       make(chan struct{}),
	}
}

func (h *hangingTransport) Close() error {
	<-h.release
	return nil
}

// TestFatal_TimeoutCapsHangingClose asserts that a Fatal call returns
// (and the test's osExit hook fires) within TransportCloseTimeout even
// if a transport's Close blocks indefinitely.
func TestFatal_TimeoutCapsHangingClose(t *testing.T) {
	exitCalled := -1
	orig := osExit
	osExit = func(code int) { exitCalled = code }
	defer func() { osExit = orig }()

	hung := newHanging("hung")
	defer close(hung.release) // unblock the goroutine on test exit so it doesn't leak past the test

	log := New(Config{
		Transport:             hung,
		TransportCloseTimeout: 50 * time.Millisecond,
	})

	start := time.Now()
	log.Fatal("goodbye")
	elapsed := time.Since(start)

	if elapsed > 500*time.Millisecond {
		t.Errorf("Fatal should return within the timeout window, took %v", elapsed)
	}
	if exitCalled != 1 {
		t.Errorf("Fatal should still call osExit(1) after the timeout, got %d", exitCalled)
	}
}

// TestRemoveTransport_TimeoutCapsHangingClose asserts the same
// timeout policy applies to the mutator path: a SIGUSR1-style hot-swap
// against a wedged endpoint must not hang the operator goroutine.
func TestRemoveTransport_TimeoutCapsHangingClose(t *testing.T) {
	hung := newHanging("hung")
	defer close(hung.release)
	keep := &stubTransport{id: "keep"}

	log := New(Config{
		Transports:            []Transport{keep, hung},
		DisableFatalExit:      true,
		TransportCloseTimeout: 50 * time.Millisecond,
	})

	start := time.Now()
	log.RemoveTransport("hung")
	elapsed := time.Since(start)

	if elapsed > 500*time.Millisecond {
		t.Errorf("RemoveTransport should return within the timeout window, took %v", elapsed)
	}
}
