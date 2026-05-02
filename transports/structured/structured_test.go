package structured_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"go.loglayer.dev/transports/structured/v2"
	"go.loglayer.dev/v2"
	"go.loglayer.dev/v2/transport"
	"go.loglayer.dev/v2/transport/transporttest"
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
	if obj["requestId"] != "xyz" {
		t.Errorf("requestId: got %v", obj["requestId"])
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
	if obj["id"] != float64(7) {
		t.Errorf("id: got %v", obj["id"])
	}
	if obj["name"] != "Alice" {
		t.Errorf("name: got %v", obj["name"])
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
