package axiom

import (
	"errors"
	"testing"

	"github.com/axiomhq/axiom-go/axiom"

	"go.loglayer.dev/v2"
	"go.loglayer.dev/v2/transport"
)

func TestBuild_NilClientReturnsError(t *testing.T) {
	_, err := Build(Config{})
	if !errors.Is(err, ErrClientRequired) {
		t.Errorf("got %v, want ErrClientRequired", err)
	}
}

func TestNew_NilClientPanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic")
		}
		err, ok := r.(error)
		if !ok || !errors.Is(err, ErrClientRequired) {
			t.Errorf("panic value: got %v, want ErrClientRequired", r)
		}
	}()
	New(Config{})
}

func TestBuild_EmptyDatasetNameReturnsError(t *testing.T) {
	_, err := Build(Config{
		Client: &axiom.Client{},
	})
	if !errors.Is(err, ErrDatasetNameRequired) {
		t.Errorf("got %v, want ErrDatasetNameRequired", err)
	}
}

func TestNew_EmptyDatasetNamePanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic")
		}
		err, ok := r.(error)
		if !ok || !errors.Is(err, ErrDatasetNameRequired) {
			t.Errorf("panic value: got %v, want ErrDatasetNameRequired", r)
		}
	}()
	New(Config{
		Client: &axiom.Client{},
	})
}

// fakeClient is a non-nil *axiom.Client so Build's nil-check passes.
// Only buildEntry is exercised below; Ingest paths that would dereference
// Client are not invoked in unit tests.
var fakeClient = &axiom.Client{}

func TestBuildEntry_PayloadShape(t *testing.T) {
	tr, err := Build(Config{
		Client:      fakeClient,
		DatasetName: "testlogs",
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	params := loglayer.TransportParams{
		LogLevel: loglayer.LogLevelInfo,
		Messages: []any{"served"},
		Data:     loglayer.Data{"requestId": "abc"},
		Metadata: loglayer.Metadata{"durationMs": 42},
	}
	entry := tr.buildEntry(params)

	if entry["msg"] != "served" {
		t.Errorf("msg: got %v, want %q", entry["msg"], "served")
	}
	if entry["requestId"] != "abc" {
		t.Errorf("requestId: got %v, want %q", entry["requestId"], "abc")
	}
	if entry["durationMs"] != 42 {
		t.Errorf("durationMs: got %v, want 42", entry["durationMs"])
	}
}

func TestBuildEntry_CustomMessageField(t *testing.T) {
	tr, err := Build(Config{
		Client:      fakeClient,
		DatasetName: "testlogs",
		MessageField: "message",
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	entry := tr.buildEntry(loglayer.TransportParams{
		LogLevel: loglayer.LogLevelInfo,
		Messages: []any{"hello"},
	})
	if entry["message"] != "hello" {
		t.Errorf("message: got %v, want %q", entry["message"], "hello")
	}
	if _, exists := entry["msg"]; exists {
		t.Error("default 'msg' key should not be set when MessageField is overridden")
	}
}

func TestBuildEntry_MergeFieldsAndMetadata(t *testing.T) {
	tr, err := Build(Config{
		Client:      fakeClient,
		DatasetName: "testlogs",
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	params := loglayer.TransportParams{
		LogLevel: loglayer.LogLevelWarn,
		Messages: []any{"warning"},
		Data:     loglayer.Data{"field1": "a", "field2": 123},
		Metadata: loglayer.Metadata{"meta1": true, "meta2": "b"},
	}
	entry := tr.buildEntry(params)

	if entry["msg"] != "warning" {
		t.Errorf("msg: got %v, want %q", entry["msg"], "warning")
	}
	if entry["field1"] != "a" {
		t.Errorf("field1: got %v, want %q", entry["field1"], "a")
	}
	if entry["field2"] != 123 {
		t.Errorf("field2: got %v, want 123", entry["field2"])
	}
	if entry["meta1"] != true {
		t.Errorf("meta1: got %v, want true", entry["meta1"])
	}
	if entry["meta2"] != "b" {
		t.Errorf("meta2: got %v, want 'b'", entry["meta2"])
	}
}

func TestBuildEntry_NonMapMetadataNestsUnderKey(t *testing.T) {
	tr, err := Build(Config{
		Client:      fakeClient,
		DatasetName: "testlogs",
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	entry := tr.buildEntry(loglayer.TransportParams{
		LogLevel: loglayer.LogLevelInfo,
		Messages: []any{"x"},
		Metadata: 42, // scalar, not a map
	})
	if entry["metadata"] != 42 {
		t.Errorf("scalar metadata should nest under 'metadata': got %v", entry["metadata"])
	}
}

func TestBuildEntry_MetadataFieldNameNests(t *testing.T) {
	tr, err := Build(Config{
		Client:      fakeClient,
		DatasetName: "testlogs",
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	params := loglayer.TransportParams{
		LogLevel: loglayer.LogLevelInfo,
		Messages: []any{"x"},
		Metadata: loglayer.Metadata{"k": "v"},
		Schema:   loglayer.Schema{MetadataFieldName: "customMeta"},
	}
	entry := tr.buildEntry(params)

	if entry["customMeta"] == nil {
		t.Fatal("metadata should be nested under customMeta")
	}
	// Check for either Metadata or map[string]any type since the interface
	// stores the actual type.
	switch m := entry["customMeta"].(type) {
	case loglayer.Metadata:
		if m["k"] != "v" {
			t.Errorf("nested metadata: got %v, want k=v", m)
		}
	case map[string]any:
		if m["k"] != "v" {
			t.Errorf("nested metadata: got %v, want k=v", m)
		}
	default:
		t.Errorf("nested metadata type: got %T, want Metadata or map[string]any", entry["customMeta"])
	}
}

func TestBuildEntry_WithPrefix(t *testing.T) {
	tr, err := Build(Config{
		Client:      fakeClient,
		DatasetName: "testlogs",
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	params := loglayer.TransportParams{
		LogLevel:  loglayer.LogLevelInfo,
		Prefix:    "[web]",
		Messages:  []any{"request completed"},
		Data:      loglayer.Data{},
		Metadata:  nil,
		Schema:    loglayer.Schema{},
	}
	// Apply prefix before calling buildEntry (simulating what SendToLogger does)
	params.Messages = transport.JoinPrefixAndMessages(params.Prefix, params.Messages)

	entry := tr.buildEntry(params)

	if entry["msg"] != "[web] request completed" {
		t.Errorf("msg with prefix: got %q, want %q", entry["msg"], "[web] request completed")
	}
}

func TestBuildEntry_WithError(t *testing.T) {
	tr, err := Build(Config{
		Client:      fakeClient,
		DatasetName: "testlogs",
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	params := loglayer.TransportParams{
		LogLevel: loglayer.LogLevelError,
		Messages: []any{"connection failed"},
		Data:     loglayer.Data{"err": map[string]any{"message": "connection refused"}},
		Metadata: nil,
	}
	params.Err = errors.New("connection refused") // kept for compatibility

	entry := tr.buildEntry(params)

	if entry["msg"] != "connection failed" {
		t.Errorf("msg: got %q", entry["msg"])
	}
	if val, ok := entry["err"]; !ok || val == nil {
		t.Errorf("err field should be populated")
	} else if m, ok := val.(map[string]any); !ok || m["message"] != "connection refused" {
		t.Errorf("err value: got %v", val)
	}
}

func TestGetLoggerInstance(t *testing.T) {
	client := &axiom.Client{}
	tr, err := Build(Config{
		Client:      client,
		DatasetName: "testlogs",
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	instance := tr.GetLoggerInstance()
	if instance != client {
		t.Errorf("GetLoggerInstance: got %p, want %p", instance, client)
	}
}
