package loglayer_test

// Core tests + shared test helpers. Per-feature tests live in:
//   fields_test.go      WithFields / WithoutFields / mute / get / FieldsKey
//   metadata_test.go    WithMetadata / MetadataOnly / mute
//   errors_test.go      WithError / ErrorOnly / ErrorSerializer
//   levels_test.go      SetLevel / Enable / Disable / IsLevelEnabled
//   transports_test.go  Add / Remove / WithFresh / GetLoggerInstance / multi
// Plus from_context_test.go, mock_test.go, concurrency_test.go, coverage_test.go.

import (
	"testing"

	"go.loglayer.dev/v2"
	"go.loglayer.dev/v2/internal/lltest"
	"go.loglayer.dev/v2/transport"
	"go.loglayer.dev/v2/transport/transporttest"
)

func setup(t *testing.T) (*loglayer.LogLayer, *lltest.TestLoggingLibrary) {
	t.Helper()
	lib := &lltest.TestLoggingLibrary{}
	trans := lltest.New(lltest.Config{
		BaseConfig: transport.BaseConfig{ID: "test"},
		Library:    lib,
	})
	log := loglayer.New(loglayer.Config{
		Transport:        trans,
		DisableFatalExit: true,
	})
	return log, lib
}

func setupWithConfig(t *testing.T, cfg loglayer.Config) (*loglayer.LogLayer, *lltest.TestLoggingLibrary) {
	t.Helper()
	lib := &lltest.TestLoggingLibrary{}
	trans := lltest.New(lltest.Config{
		BaseConfig: transport.BaseConfig{ID: "test"},
		Library:    lib,
	})
	cfg.Transport = trans
	cfg.DisableFatalExit = true
	log := loglayer.New(cfg)
	return log, lib
}

func assertLine(t *testing.T, lib *lltest.TestLoggingLibrary, wantLevel loglayer.LogLevel, wantMsg string) *lltest.LogLine {
	t.Helper()
	line := lib.PopLine()
	if line == nil {
		t.Fatalf("expected a log line at level %s but got none", wantLevel)
	}
	if line.Level != wantLevel {
		t.Errorf("level: got %s, want %s", line.Level, wantLevel)
	}
	if wantMsg != "" {
		if !transporttest.MessageContains(line.Messages, wantMsg) {
			t.Errorf("message %q not found in messages: %v", wantMsg, line.Messages)
		}
	}
	return line
}

// metadataMap returns line.Metadata as a map, or nil if it isn't one.
func metadataMap(line *lltest.LogLine) map[string]any {
	if line == nil {
		return nil
	}
	m, _ := line.Metadata.(map[string]any)
	return m
}

func TestBasicLogLevels(t *testing.T) {
	log, lib := setup(t)

	log.Info("hello info")
	assertLine(t, lib, loglayer.LogLevelInfo, "hello info")

	log.Warn("hello warn")
	assertLine(t, lib, loglayer.LogLevelWarn, "hello warn")

	log.Error("hello error")
	assertLine(t, lib, loglayer.LogLevelError, "hello error")

	log.Debug("hello debug")
	assertLine(t, lib, loglayer.LogLevelDebug, "hello debug")

	log.Fatal("hello fatal")
	assertLine(t, lib, loglayer.LogLevelFatal, "hello fatal")
}

func TestMultipleMessages(t *testing.T) {
	log, lib := setup(t)
	log.Info("a", "b", "c")
	line := lib.PopLine()
	if line == nil {
		t.Fatal("expected a log line")
	}
	if line.Level != loglayer.LogLevelInfo {
		t.Errorf("level: got %s, want info", line.Level)
	}
	if len(line.Messages) != 3 {
		t.Errorf("expected 3 messages, got %d", len(line.Messages))
	}
}

func TestPrefix(t *testing.T) {
	log, lib := setup(t)
	prefixed := log.WithPrefix("[app]")
	prefixed.Info("started")
	// v2: the prefix is no longer mutated into Messages[0]; it
	// flows through TransportParams.Prefix and the transport
	// renders it however it likes. Assert the message arrives
	// untouched plus the prefix is exposed on params.Prefix.
	line := assertLine(t, lib, loglayer.LogLevelInfo, "started")
	if line.Prefix != "[app]" {
		t.Errorf("Prefix = %q, want %q", line.Prefix, "[app]")
	}
}

func TestPrefixDoesNotAffectParent(t *testing.T) {
	log, lib := setup(t)
	child := log.WithPrefix("[child]")
	_ = child
	log.Info("parent")
	assertLine(t, lib, loglayer.LogLevelInfo, "parent")
}

func TestPrefixSurfacedOnTransportParams(t *testing.T) {
	// v2: Prefix is exposed verbatim on TransportParams.Prefix and
	// is NO LONGER prepended into Messages[0]. Transports that
	// want the v1 "prefix prepended into the message" behavior
	// call transport.JoinPrefixAndMessages explicitly.
	log, lib := setup(t)
	prefixed := log.WithPrefix("[auth]")
	prefixed.Info("starting")

	line := lib.PopLine()
	if line.Prefix != "[auth]" {
		t.Errorf("Prefix = %q, want %q", line.Prefix, "[auth]")
	}
	if got, _ := line.Messages[0].(string); got != "starting" {
		t.Errorf("Messages[0] = %q, want %q (prefix no longer auto-prepended)", got, "starting")
	}
}

func TestPrefixEmptyWhenNotSet(t *testing.T) {
	log, lib := setup(t)
	log.Info("no prefix")

	line := lib.PopLine()
	if line.Prefix != "" {
		t.Errorf("Prefix = %q, want empty", line.Prefix)
	}
}

func TestPrefixSurfacedOnAllPluginHookParams(t *testing.T) {
	// Every dispatch-time hook param struct must carry Prefix
	// alongside TransportParams. A regression that leaves Prefix
	// unset on (say) ShouldSendParams would slip past the
	// TransportParams-only test, so install one hook of each kind
	// and capture each variant's Prefix.
	var (
		dataPrefix  string
		msgPrefix   string
		levelPrefix string
		gatePrefix  string
	)
	plugins := []loglayer.Plugin{
		loglayer.NewDataHook("capture-data", func(p loglayer.BeforeDataOutParams) loglayer.Data {
			dataPrefix = p.Prefix
			return nil
		}),
		loglayer.NewMessageHook("capture-msg", func(p loglayer.BeforeMessageOutParams) []any {
			msgPrefix = p.Prefix
			return nil
		}),
		loglayer.NewLevelHook("capture-level", func(p loglayer.TransformLogLevelParams) (loglayer.LogLevel, bool) {
			levelPrefix = p.Prefix
			return 0, false
		}),
		loglayer.NewSendGate("capture-gate", func(p loglayer.ShouldSendParams) bool {
			gatePrefix = p.Prefix
			return true
		}),
	}
	cfg := loglayer.Config{Plugins: plugins}
	log, _ := setupWithConfig(t, cfg)
	log.WithPrefix("[auth]").Info("hello")

	cases := map[string]string{
		"BeforeDataOutParams":     dataPrefix,
		"BeforeMessageOutParams":  msgPrefix,
		"TransformLogLevelParams": levelPrefix,
		"ShouldSendParams":        gatePrefix,
	}
	for name, got := range cases {
		if got != "[auth]" {
			t.Errorf("%s.Prefix = %q, want %q", name, got, "[auth]")
		}
	}
}

func TestChildInheritsFields(t *testing.T) {
	log, lib := setup(t)
	log = log.WithFields(loglayer.Fields{"parent": "yes"})
	child := log.Child()
	child.Info("from child")
	line := lib.PopLine()
	if line.Data["parent"] != "yes" {
		t.Errorf("child should inherit parent fields, got %v", line.Data)
	}
}

func TestChildFieldsIsolated(t *testing.T) {
	log, lib := setup(t)
	log = log.WithFields(loglayer.Fields{"shared": "v"})
	child := log.Child()
	_ = child.WithFields(loglayer.Fields{"child_only": "x"}) // discarded: testing isolation, not the result

	log.Info("parent log")
	line := lib.PopLine()
	if line.Data["child_only"] != nil {
		t.Errorf("parent should not see child-only fields: %v", line.Data)
	}
}

func TestChildInheritsLevels(t *testing.T) {
	log, lib := setup(t)
	log.SetLevel(loglayer.LogLevelError)
	child := log.Child()
	child.Info("dropped by inherited level")
	if lib.Len() != 0 {
		t.Errorf("child should inherit parent level, got %d lines", lib.Len())
	}
}

func TestRaw(t *testing.T) {
	log, lib := setup(t)
	log.Raw(loglayer.RawLogEntry{
		LogLevel: loglayer.LogLevelWarn,
		Messages: []any{"raw message"},
		Metadata: map[string]any{"k": "v"},
	})
	line := lib.PopLine()
	if line == nil {
		t.Fatal("expected a line from Raw")
	}
	if line.Level != loglayer.LogLevelWarn {
		t.Errorf("Raw level: got %s", line.Level)
	}
	m := metadataMap(line)
	if m["k"] != "v" {
		t.Errorf("Raw metadata: got %v", line.Metadata)
	}
}

func TestRawCustomFields(t *testing.T) {
	log, lib := setup(t)
	log = log.WithFields(loglayer.Fields{"logger_ctx": "ignored"})
	log.Raw(loglayer.RawLogEntry{
		LogLevel: loglayer.LogLevelInfo,
		Messages: []any{"raw"},
		Fields:   loglayer.Fields{"override": "yes"},
	})
	line := lib.PopLine()
	if line.Data["override"] != "yes" {
		t.Errorf("Raw custom fields: got %v", line.Data)
	}
	if line.Data["logger_ctx"] != nil {
		t.Errorf("Raw custom fields should override logger fields: got %v", line.Data)
	}
}
