package loglayer

import "testing"

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
