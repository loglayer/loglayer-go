package loglayer

// discardTransport is a no-op transport used by NewMock. It accepts every entry
// and produces no output. Level filtering is irrelevant because SendToLogger
// does nothing.
type discardTransport struct{}

func (discardTransport) ID() string                   { return "mock" }
func (discardTransport) IsEnabled() bool              { return true }
func (discardTransport) SendToLogger(TransportParams) {}
func (discardTransport) GetLoggerInstance() any       { return nil }

// NewMock returns a *LogLayer that silently discards every entry. Use it in
// tests when you need to pass a logger to code under test but don't care about
// its output.
//
// The returned value is the same concrete *LogLayer type as a production
// logger, so it drops into anywhere the real one fits. All methods behave
// normally — context, metadata, child loggers, level changes — they just
// produce no output.
//
// DisableFatalExit is enabled so log.Fatal(...) in code under test does not
// terminate the test process.
//
// To assert on what was logged, use the transports/testing transport instead.
func NewMock() *LogLayer {
	return New(Config{
		Transport:        discardTransport{},
		DisableFatalExit: true,
	})
}
