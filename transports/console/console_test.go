package console_test

import (
	"bytes"
	"strings"
	"testing"

	"go.loglayer.dev"
	"go.loglayer.dev/transport"
	"go.loglayer.dev/transports/console"
)

func newLogger(cfg console.Config) (*loglayer.LogLayer, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	cfg.Writer = buf
	if cfg.BaseConfig.ID == "" {
		cfg.BaseConfig.ID = "console"
	}
	t := console.New(cfg)
	log := loglayer.New(loglayer.Config{DisableFatalExit: true, Transport: t})
	return log, buf
}

func TestConsoleBasicOutput(t *testing.T) {
	log, buf := newLogger(console.Config{})
	log.Info("hello world")
	if !strings.Contains(buf.String(), "hello world") {
		t.Errorf("expected 'hello world' in output, got: %q", buf.String())
	}
}

func TestConsoleWithData(t *testing.T) {
	log, buf := newLogger(console.Config{})
	log.WithMetadata(map[string]any{"k": "v"}).Info("with data")
	out := buf.String()
	if !strings.Contains(out, "with data") {
		t.Errorf("expected message in output, got: %q", out)
	}
	if !strings.Contains(out, "k") {
		t.Errorf("expected key 'k' in output, got: %q", out)
	}
}

func TestConsoleLogfmtOutput(t *testing.T) {
	log, buf := newLogger(console.Config{})
	log.WithMetadata(map[string]any{
		"id":     42,
		"name":   "alice",
		"with s": "value with space",
	}).Info("event")
	out := strings.TrimSpace(buf.String())

	// Logfmt convention: message first, then sorted key=value pairs.
	if !strings.HasPrefix(out, "event ") {
		t.Errorf("expected message before fields, got: %q", out)
	}
	if !strings.Contains(out, "id=42") {
		t.Errorf("expected unquoted scalar 'id=42', got: %q", out)
	}
	if !strings.Contains(out, "name=alice") {
		t.Errorf("expected unquoted 'name=alice', got: %q", out)
	}
	if !strings.Contains(out, `"with s"="value with space"`) {
		t.Errorf("expected quoted key + value with space, got: %q", out)
	}
	// Sorted: id < name < with s.
	idIdx := strings.Index(out, "id=")
	nameIdx := strings.Index(out, "name=")
	if !(idIdx < nameIdx) {
		t.Errorf("expected sorted keys (id before name), got: %q", out)
	}
}

func TestConsoleLogfmtNestedJSON(t *testing.T) {
	log, buf := newLogger(console.Config{})
	log.WithMetadata(map[string]any{
		"nested": map[string]any{"a": 1, "b": "two"},
	}).Info("event")
	out := strings.TrimSpace(buf.String())
	// Nested values are JSON-encoded and string-quoted.
	if !strings.Contains(out, `nested="{`) {
		t.Errorf("expected JSON-encoded nested value, got: %q", out)
	}
}

func TestConsoleMessageField(t *testing.T) {
	log, buf := newLogger(console.Config{
		MessageField: "msg",
		DateField:    "ts",
		LevelField:   "level",
	})
	log.Info("structured")
	out := buf.String()
	for _, want := range []string{"msg", "ts", "level", "structured"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output, got: %q", want, out)
		}
	}
}

func TestConsoleMessageFieldStringify(t *testing.T) {
	log, buf := newLogger(console.Config{
		MessageField: "msg",
		Stringify:    true,
	})
	log.Info("json out")
	out := strings.TrimSpace(buf.String())
	if !strings.HasPrefix(out, "{") || !strings.HasSuffix(out, "}") {
		t.Errorf("expected JSON object, got: %q", out)
	}
	if !strings.Contains(out, `"msg"`) {
		t.Errorf("expected msg key in JSON, got: %q", out)
	}
}

func TestConsoleLevelField(t *testing.T) {
	log, buf := newLogger(console.Config{
		LevelField: "severity",
	})
	log.Warn("leveled")
	out := buf.String()
	if !strings.Contains(out, "severity") {
		t.Errorf("expected 'severity' key in output, got: %q", out)
	}
	if !strings.Contains(out, "warn") {
		t.Errorf("expected 'warn' level in output, got: %q", out)
	}
}

func TestConsoleDateField(t *testing.T) {
	log, buf := newLogger(console.Config{
		DateField: "timestamp",
	})
	log.Info("dated")
	if !strings.Contains(buf.String(), "timestamp") {
		t.Errorf("expected 'timestamp' field, got: %q", buf.String())
	}
}

func TestConsoleCustomLevelFn(t *testing.T) {
	log, buf := newLogger(console.Config{
		LevelField: "lvl",
		LevelFn:    func(l loglayer.LogLevel) string { return "CUSTOM_" + l.String() },
	})
	log.Info("custom level fn")
	if !strings.Contains(buf.String(), "CUSTOM_info") {
		t.Errorf("expected CUSTOM_info in output, got: %q", buf.String())
	}
}

func TestConsoleCustomDateFn(t *testing.T) {
	log, buf := newLogger(console.Config{
		DateField: "ts",
		DateFn:    func() string { return "fixed-date" },
	})
	log.Info("custom date")
	if !strings.Contains(buf.String(), "fixed-date") {
		t.Errorf("expected 'fixed-date' in output, got: %q", buf.String())
	}
}

func TestConsoleMessageFn(t *testing.T) {
	log, buf := newLogger(console.Config{
		MessageFn: func(p loglayer.TransportParams) string {
			return "overridden"
		},
	})
	log.Info("original")
	out := buf.String()
	if !strings.Contains(out, "overridden") {
		t.Errorf("expected 'overridden' from MessageFn, got: %q", out)
	}
}

func TestConsoleLevelFiltering(t *testing.T) {
	buf := &bytes.Buffer{}
	t1 := console.New(console.Config{
		BaseConfig: transport.BaseConfig{
			ID:    "console",
			Level: loglayer.LogLevelWarn,
		},
		Writer: buf,
	})
	log := loglayer.New(loglayer.Config{DisableFatalExit: true, Transport: t1})
	log.Info("filtered")
	if buf.Len() != 0 {
		t.Errorf("info should be filtered at warn level, got: %q", buf.String())
	}
	log.Warn("passes")
	if !strings.Contains(buf.String(), "passes") {
		t.Errorf("warn should pass, got: %q", buf.String())
	}
}
