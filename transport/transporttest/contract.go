package transporttest

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"go.loglayer.dev"
)

// FactoryOpts lets the contract drive a wrapper's Config knobs without
// knowing the concrete Config type. Wrappers translate these into their own
// fields inside the Factory closure. Zero values mean "leave as wrapper
// default" for both fields.
type FactoryOpts struct {
	MetadataFieldName string
	Level             loglayer.LogLevel
}

// Factory builds a fresh logger + buffer pair honoring the supplied opts.
type Factory func(opts FactoryOpts) (*loglayer.LogLayer, *bytes.Buffer)

// Expectations describes the per-wrapper rendering quirks the contract suite
// needs to know about.
type Expectations struct {
	// MessageKey is the JSON key the wrapper writes the joined message under.
	// "msg" for most wrappers; "message" for zerolog and phuslu.
	MessageKey string

	// LevelKey is the JSON key the wrapper writes the level under.
	LevelKey string

	// Levels maps each LogLevel to the string the wrapper renders for it.
	// Omit a level to skip it in the level-cycling test (some wrappers don't
	// have a native trace level).
	Levels map[loglayer.LogLevel]string

	// SkipFatal omits any test that calls log.Fatal(). Set this for wrappers
	// (phuslu) whose underlying library unconditionally calls os.Exit on
	// Fatal.
	SkipFatal bool
}

// ContractCase bundles everything RunContract needs.
type ContractCase struct {
	Name    string
	Factory Factory
	Expect  Expectations
}

// RunContract runs the shared wrapper-transport contract suite.
func RunContract(t *testing.T, c ContractCase) {
	t.Helper()
	t.Run(c.Name+"/SimpleMessage", func(t *testing.T) { t.Parallel(); testSimpleMessage(t, c) })
	t.Run(c.Name+"/MultipleMessages", func(t *testing.T) { t.Parallel(); testMultipleMessages(t, c) })
	t.Run(c.Name+"/Levels", func(t *testing.T) { t.Parallel(); testLevels(t, c) })
	if !c.Expect.SkipFatal {
		t.Run(c.Name+"/FatalDoesNotExit", func(t *testing.T) { t.Parallel(); testFatalDoesNotExit(t, c) })
	}
	t.Run(c.Name+"/MapMetadataMerged", func(t *testing.T) { t.Parallel(); testMapMetadataMerged(t, c) })
	t.Run(c.Name+"/StructMetadataNested", func(t *testing.T) { t.Parallel(); testStructMetadataNested(t, c) })
	t.Run(c.Name+"/CustomMetadataFieldName", func(t *testing.T) { t.Parallel(); testCustomMetadataFieldName(t, c) })
	t.Run(c.Name+"/FieldsMerged", func(t *testing.T) { t.Parallel(); testFieldsMerged(t, c) })
	t.Run(c.Name+"/WithError", func(t *testing.T) { t.Parallel(); testWithError(t, c) })
	t.Run(c.Name+"/LevelFiltering", func(t *testing.T) { t.Parallel(); testLevelFiltering(t, c) })
	t.Run(c.Name+"/MetadataOnly", func(t *testing.T) { t.Parallel(); testMetadataOnly(t, c) })
	t.Run(c.Name+"/ErrorOnly", func(t *testing.T) { t.Parallel(); testErrorOnly(t, c) })
	t.Run(c.Name+"/Raw", func(t *testing.T) { t.Parallel(); testRaw(t, c) })
	t.Run(c.Name+"/WithContextDoesNotBreakDispatch", func(t *testing.T) { t.Parallel(); testWithContextDoesNotBreakDispatch(t, c) })
}

// assertErrContains accepts either a map with a "message" key or a string,
// in both cases asserting the want substring is present.
func assertErrContains(t *testing.T, obj map[string]any, want string) {
	t.Helper()
	got, ok := obj["err"]
	if !ok {
		t.Fatalf("expected err field, got %v", obj)
	}
	switch v := got.(type) {
	case map[string]any:
		msg, _ := v["message"].(string)
		if !strings.Contains(msg, want) {
			t.Errorf("err.message: got %v, want substring %q", v["message"], want)
		}
	case string:
		if !strings.Contains(v, want) {
			t.Errorf("err string: got %v, want substring %q", v, want)
		}
	default:
		t.Errorf("unexpected err type %T", v)
	}
}

func testSimpleMessage(t *testing.T, c ContractCase) {
	log, buf := c.Factory(FactoryOpts{})
	log.Info("hello")
	obj := ParseJSONLine(t, buf)
	if obj[c.Expect.MessageKey] != "hello" {
		t.Errorf("%s: got %v, want %q", c.Expect.MessageKey, obj[c.Expect.MessageKey], "hello")
	}
	if want := c.Expect.Levels[loglayer.LogLevelInfo]; want != "" && obj[c.Expect.LevelKey] != want {
		t.Errorf("%s: got %v, want %q", c.Expect.LevelKey, obj[c.Expect.LevelKey], want)
	}
}

func testMultipleMessages(t *testing.T, c ContractCase) {
	log, buf := c.Factory(FactoryOpts{})
	log.Info("part1", "part2")
	obj := ParseJSONLine(t, buf)
	if obj[c.Expect.MessageKey] != "part1 part2" {
		t.Errorf("%s: got %v", c.Expect.MessageKey, obj[c.Expect.MessageKey])
	}
}

func testLevels(t *testing.T, c ContractCase) {
	cases := []struct {
		level loglayer.LogLevel
		emit  func(*loglayer.LogLayer)
	}{
		{loglayer.LogLevelDebug, func(l *loglayer.LogLayer) { l.Debug("x") }},
		{loglayer.LogLevelInfo, func(l *loglayer.LogLayer) { l.Info("x") }},
		{loglayer.LogLevelWarn, func(l *loglayer.LogLayer) { l.Warn("x") }},
		{loglayer.LogLevelError, func(l *loglayer.LogLayer) { l.Error("x") }},
		{loglayer.LogLevelFatal, func(l *loglayer.LogLayer) { l.Fatal("x") }},
	}
	for _, tc := range cases {
		if c.Expect.SkipFatal && tc.level == loglayer.LogLevelFatal {
			continue
		}
		want, ok := c.Expect.Levels[tc.level]
		if !ok {
			continue
		}
		log, buf := c.Factory(FactoryOpts{})
		tc.emit(log)
		obj := ParseJSONLine(t, buf)
		if obj[c.Expect.LevelKey] != want {
			t.Errorf("level %v: got %v, want %q", tc.level, obj[c.Expect.LevelKey], want)
		}
	}
}

func testFatalDoesNotExit(t *testing.T, c ContractCase) {
	log, buf := c.Factory(FactoryOpts{})
	log.Fatal("survives")
	obj := ParseJSONLine(t, buf)
	if want := c.Expect.Levels[loglayer.LogLevelFatal]; want != "" && obj[c.Expect.LevelKey] != want {
		t.Errorf("fatal level: got %v, want %q", obj[c.Expect.LevelKey], want)
	}
}

func testMapMetadataMerged(t *testing.T, c ContractCase) {
	log, buf := c.Factory(FactoryOpts{})
	log.WithMetadata(loglayer.Metadata{"requestId": "xyz", "n": 42}).Info("req")
	obj := ParseJSONLine(t, buf)
	if obj["requestId"] != "xyz" {
		t.Errorf("requestId: got %v", obj["requestId"])
	}
	if obj["n"] != float64(42) {
		t.Errorf("n: got %v", obj["n"])
	}
}

func testStructMetadataNested(t *testing.T, c ContractCase) {
	type user struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	log, buf := c.Factory(FactoryOpts{})
	log.WithMetadata(user{ID: 7, Name: "Alice"}).Info("hi")
	// charmlog renders the keyval value through a non-JSON pipeline so the
	// inner struct may not parse as a nested map[string]any. Loose substring
	// check covers that case; strict nested-map check runs when JSON is clean.
	out := buf.String()
	if !strings.Contains(out, "metadata") {
		t.Fatalf("expected \"metadata\" key in output, got %q", out)
	}
	if !strings.Contains(out, "Alice") {
		t.Errorf("expected struct field rendered, got %q", out)
	}
	obj, err := tryParseJSONLine(buf)
	if err != nil {
		return
	}
	if nested, ok := obj["metadata"].(map[string]any); ok {
		if nested["id"] != float64(7) || nested["name"] != "Alice" {
			t.Errorf("nested fields: got %v", nested)
		}
	}
}

func testCustomMetadataFieldName(t *testing.T, c ContractCase) {
	type user struct {
		ID int `json:"id"`
	}
	log, buf := c.Factory(FactoryOpts{MetadataFieldName: "user"})
	log.WithMetadata(user{ID: 9}).Info("hi")
	out := buf.String()
	if !strings.Contains(out, "user") {
		t.Errorf("expected 'user' key in output, got %q", out)
	}
}

func testFieldsMerged(t *testing.T, c ContractCase) {
	log, buf := c.Factory(FactoryOpts{})
	log = log.WithFields(loglayer.Fields{"service": "api"})
	log.Info("fields test")
	obj := ParseJSONLine(t, buf)
	if obj["service"] != "api" {
		t.Errorf("service: got %v", obj["service"])
	}
}

func testWithError(t *testing.T, c ContractCase) {
	log, buf := c.Factory(FactoryOpts{})
	log.WithError(errors.New("boom")).Error("failed")
	obj := ParseJSONLine(t, buf)
	assertErrContains(t, obj, "boom")
}

func testLevelFiltering(t *testing.T, c ContractCase) {
	log, buf := c.Factory(FactoryOpts{Level: loglayer.LogLevelError})
	log.Warn("dropped")
	if buf.Len() != 0 {
		t.Errorf("warn should be filtered, got: %q", buf.String())
	}
	log.Error("passes")
	obj := ParseJSONLine(t, buf)
	if obj[c.Expect.MessageKey] != "passes" {
		t.Errorf("%s: got %v", c.Expect.MessageKey, obj[c.Expect.MessageKey])
	}
}

func testMetadataOnly(t *testing.T, c ContractCase) {
	log, buf := c.Factory(FactoryOpts{})
	log.MetadataOnly(loglayer.Metadata{"status": "healthy"})
	obj := ParseJSONLine(t, buf)
	if obj["status"] != "healthy" {
		t.Errorf("status: got %v", obj["status"])
	}
	if want := c.Expect.Levels[loglayer.LogLevelInfo]; want != "" && obj[c.Expect.LevelKey] != want {
		t.Errorf("default level should be info-equivalent, got %v want %q", obj[c.Expect.LevelKey], want)
	}
}

func testErrorOnly(t *testing.T, c ContractCase) {
	log, buf := c.Factory(FactoryOpts{})
	log.ErrorOnly(errors.New("boom"))
	out := buf.String()
	if !strings.Contains(out, "boom") {
		t.Errorf("expected 'boom' in output, got %q", out)
	}
	if want := c.Expect.Levels[loglayer.LogLevelError]; want != "" && !strings.Contains(out, want) {
		t.Errorf("expected level %q in output, got %q", want, out)
	}
}

func testRaw(t *testing.T, c ContractCase) {
	log, buf := c.Factory(FactoryOpts{})
	log.Raw(loglayer.RawLogEntry{
		LogLevel: loglayer.LogLevelInfo,
		Messages: []any{"raw entry"},
		Metadata: loglayer.Metadata{"k": "v"},
	})
	obj := ParseJSONLine(t, buf)
	if obj[c.Expect.MessageKey] != "raw entry" {
		t.Errorf("%s: got %v", c.Expect.MessageKey, obj[c.Expect.MessageKey])
	}
	if obj["k"] != "v" {
		t.Errorf("k: got %v", obj["k"])
	}
}

func testWithContextDoesNotBreakDispatch(t *testing.T, c ContractCase) {
	log, buf := c.Factory(FactoryOpts{})
	log = log.WithContext(context.Background())
	log.Info("with ctx")
	obj := ParseJSONLine(t, buf)
	if obj[c.Expect.MessageKey] != "with ctx" {
		t.Errorf("%s: got %v", c.Expect.MessageKey, obj[c.Expect.MessageKey])
	}
}
