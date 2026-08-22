package structured_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"go.loglayer.dev/transports/structured/v3"
	"go.loglayer.dev/v3"
	"go.loglayer.dev/v3/transport"
	"go.loglayer.dev/v3/transport/transporttest"
)

func newLogger(cfg structured.Config) (*loglayer.LogLayer, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	cfg.Writer = buf
	if cfg.BaseConfig.ID == "" {
		cfg.BaseConfig.ID = "structured"
	}
	t := structured.New(cfg)
	log := loglayer.New(loglayer.Config{DisableFatalExit: true, Transport: t})
	return log, buf
}

func TestStructuredAlwaysJSON(t *testing.T) {
	log, buf := newLogger(structured.Config{})
	log.Info("hello")
	line := strings.TrimSpace(buf.String())
	if !strings.HasPrefix(line, "{") || !strings.HasSuffix(line, "}") {
		t.Errorf("expected JSON object, got: %q", line)
	}
}

func TestStructuredDefaultFields(t *testing.T) {
	log, buf := newLogger(structured.Config{})
	log.Info("hello")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["msg"] != "hello" {
		t.Errorf("msg: got %v", obj["msg"])
	}
	if obj["level"] != "info" {
		t.Errorf("level: got %v", obj["level"])
	}
	if obj["time"] == nil {
		t.Error("expected time field")
	}
}

func TestStructuredLevel(t *testing.T) {
	log, buf := newLogger(structured.Config{})
	log.Error("err msg")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["level"] != "error" {
		t.Errorf("level: got %v", obj["level"])
	}
}

func TestStructuredCustomFields(t *testing.T) {
	log, buf := newLogger(structured.Config{
		MessageField: "message",
		DateField:    "timestamp",
		LevelField:   "severity",
	})
	log.Info("custom fields")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["message"] == nil {
		t.Errorf("expected 'message' field, got %v", obj)
	}
	if obj["timestamp"] == nil {
		t.Errorf("expected 'timestamp' field, got %v", obj)
	}
	if obj["severity"] == nil {
		t.Errorf("expected 'severity' field, got %v", obj)
	}
}

func TestStructuredWithMetadataMap(t *testing.T) {
	log, buf := newLogger(structured.Config{})
	log.WithMetadata(map[string]any{"requestId": "xyz"}).Info("req")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["metadata"].(map[string]any)["requestId"] != "xyz" {
		t.Errorf("metadata.requestId: got %v", obj["metadata"])
	}
}

func TestStructuredWithMetadataStruct(t *testing.T) {
	type user struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	log, buf := newLogger(structured.Config{})
	log.WithMetadata(user{ID: 7, Name: "Alice"}).Info("hi")
	obj := transporttest.ParseJSONLine(t, buf)
	md := obj["metadata"].(map[string]any)
	if md["id"] != float64(7) {
		t.Errorf("metadata.id: got %v", md["id"])
	}
	if md["name"] != "Alice" {
		t.Errorf("metadata.name: got %v", md["name"])
	}
}

func TestStructuredCustomLevelFn(t *testing.T) {
	log, buf := newLogger(structured.Config{
		LevelFn: func(l loglayer.LogLevel) string { return strings.ToUpper(l.String()) },
	})
	log.Warn("upper level")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["level"] != "WARN" {
		t.Errorf("level: got %v", obj["level"])
	}
}

func TestStructuredCustomDateFn(t *testing.T) {
	log, buf := newLogger(structured.Config{
		DateFn: func() string { return "2024-01-01" },
	})
	log.Info("fixed date")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["time"] != "2024-01-01" {
		t.Errorf("time: got %v", obj["time"])
	}
}

func TestStructuredMessageFn(t *testing.T) {
	log, buf := newLogger(structured.Config{
		MessageFn: func(p loglayer.TransportParams) string { return "formatted" },
	})
	log.Info("original")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["msg"] != "formatted" {
		t.Errorf("msg from MessageFn: got %v", obj["msg"])
	}
}

func TestStructuredLevelFiltering(t *testing.T) {
	buf := &bytes.Buffer{}
	t1 := structured.New(structured.Config{
		BaseConfig: transport.BaseConfig{
			ID:    "structured",
			Level: loglayer.LogLevelError,
		},
		Writer: buf,
	})
	log := loglayer.New(loglayer.Config{DisableFatalExit: true, Transport: t1})
	log.Warn("dropped")
	if buf.Len() != 0 {
		t.Errorf("warn should be filtered at error level, got: %q", buf.String())
	}
	log.Error("passes")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["msg"] != "passes" {
		t.Errorf("msg: got %v", obj["msg"])
	}
}

func TestStructuredMultipleMessages(t *testing.T) {
	log, buf := newLogger(structured.Config{})
	log.Info("part1", "part2")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["msg"] != "part1 part2" {
		t.Errorf("expected joined message, got: %v", obj["msg"])
	}
}

func TestStructuredWithFields(t *testing.T) {
	log, buf := newLogger(structured.Config{})
	log = log.WithFields(loglayer.Fields{"service": "api"})
	log.Info("ctx test")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["service"] != "api" {
		t.Errorf("service: got %v", obj["service"])
	}
}

// AddSource: the structured transport renders the captured Source under
// SourceFieldName as a nested {function, file, line} object via the
// json tags on loglayer.Source.
func TestStructuredSourceFieldRendered(t *testing.T) {
	buf := &bytes.Buffer{}
	tr := structured.New(structured.Config{Writer: buf})
	log := loglayer.New(loglayer.Config{
		Transport:        tr,
		Source:           loglayer.SourceConfig{Enabled: true},
		DisableFatalExit: true,
	})

	log.Info("hi") // capture site

	var obj map[string]any
	if err := json.Unmarshal(buf.Bytes(), &obj); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, buf.String())
	}
	src, ok := obj["source"].(map[string]any)
	if !ok {
		t.Fatalf("source key missing or wrong shape: %v", obj["source"])
	}
	if fn, _ := src["function"].(string); !strings.Contains(fn, "TestStructuredSourceFieldRendered") {
		t.Errorf("function: got %v", src["function"])
	}
	if file, _ := src["file"].(string); !strings.HasSuffix(file, "structured_test.go") {
		t.Errorf("file: got %v", src["file"])
	}
	if _, ok := src["line"].(float64); !ok {
		t.Errorf("line should be a number: got %v (%T)", src["line"], src["line"])
	}
}

// emit runs a log call through a fresh logger and returns the raw output line.
// No access to the Write hook, but the assembly and encoding paths are
// exercised exactly as in production.
func emit(cfg structured.Config, fn func(*loglayer.LogLayer)) string {
	log, buf := newLogger(cfg)
	fn(log)
	return strings.TrimSpace(buf.String())
}

// ANSI escapes, CR/LF, and bidi overrides (Trojan Source, CVE-2021-42574) in
// the message are stripped so a log line can't smuggle terminal control
// sequences or hide content. DateFn pins the timestamp so the whole line is
// compared byte-for-byte.
func TestStructuredSanitizesInjectionInMessage(t *testing.T) {
	cfg := structured.Config{DateFn: func() string { return "2026-04-26T12:00:00Z" }}
	if got := emit(cfg, func(log *loglayer.LogLayer) {
		log.Info("line1\r\nline2\x1b[31mreduser‮evil")
	}); got != `{"level":"info","time":"2026-04-26T12:00:00Z","msg":"line1line2[31mreduserevil"}` {
		t.Errorf("got %q", got)
	}
}

// Same guard on keys and string values when metadata flattens: each metadata
// entry becomes a top-level key/value, so both are sanitized. Non-string
// entries pass through untouched.
func TestStructuredSanitizesInjectionInFlattenedMetadata(t *testing.T) {
	buf := &bytes.Buffer{}
	t1 := structured.New(structured.Config{Writer: buf})
	log := loglayer.New(loglayer.Config{
		Transport:        t1,
		FlattenMetadata:  true,
		DisableFatalExit: true,
	})
	log.WithMetadata(map[string]any{
		"note\r\n\x1b[": "line1\r\nline2\x1b[31mred",
		"count":         7,
	}).Info("hi")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["note["] != "line1line2[31mred" {
		t.Errorf("note[: got %v", obj["note["])
	}
	if obj["count"] != float64(7) {
		t.Errorf("count: got %v", obj["count"])
	}
}

// String-typed top-level values (WithFields entries) sanitize the same way,
// including the keys; non-string values pass through untouched.
func TestStructuredSanitizesInjectionInTopLevelStringValues(t *testing.T) {
	log, buf := newLogger(structured.Config{})
	log = log.WithFields(loglayer.Fields{
		"user\r\n":   "user‮evil",
		"cleanInt":   42,
		"cleanBool":  true,
		"cleanFloat": 1.5,
	})
	log.Info("ok")
	obj := transporttest.ParseJSONLine(t, buf)
	if obj["user"] != "userevil" {
		t.Errorf("user: got %v", obj["user"])
	}
	if obj["cleanInt"] != float64(42) {
		t.Errorf("cleanInt: got %v", obj["cleanInt"])
	}
	if obj["cleanBool"] != true {
		t.Errorf("cleanBool: got %v", obj["cleanBool"])
	}
	if obj["cleanFloat"] != 1.5 {
		t.Errorf("cleanFloat: got %v", obj["cleanFloat"])
	}
}

// An empty joined message omits the msg key entirely, so
// WithFields(...)+Info("") renders level, time, and the fields only. A
// message that sanitizes to nothing (e.g. bare escape bytes) is omitted the
// same way.
func TestStructuredOmitsMsgWhenEmpty(t *testing.T) {
	log, buf := newLogger(structured.Config{})
	log = log.WithFields(loglayer.Fields{"service": "api"})
	log.Info("")
	obj := transporttest.ParseJSONLine(t, buf)
	if _, ok := obj["msg"]; ok {
		t.Errorf("msg should be omitted for empty message, got: %v", obj)
	}
	if obj["service"] != "api" {
		t.Errorf("service: got %v", obj["service"])
	}

	buf.Reset()
	log.Info("\x1b\x1b")
	obj = transporttest.ParseJSONLine(t, buf)
	if _, ok := obj["msg"]; ok {
		t.Errorf("msg should be omitted when sanitized message is empty, got: %v", obj)
	}
}

// SourceFieldName overrides the rendered key.
func TestStructuredSourceFieldNameOverride(t *testing.T) {
	buf := &bytes.Buffer{}
	tr := structured.New(structured.Config{Writer: buf})
	log := loglayer.New(loglayer.Config{
		Transport:        tr,
		Source:           loglayer.SourceConfig{Enabled: true, FieldName: "caller"},
		DisableFatalExit: true,
	})

	log.Info("hi")

	var obj map[string]any
	if err := json.Unmarshal(buf.Bytes(), &obj); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := obj["source"]; ok {
		t.Errorf("default key 'source' should not appear when override set: %v", obj)
	}
	if _, ok := obj["caller"]; !ok {
		t.Errorf("expected source under 'caller': %v", obj)
	}
}
