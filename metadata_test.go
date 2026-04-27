package loglayer_test

import (
	"testing"

	"go.loglayer.dev"
)

func TestWithMetadataMap(t *testing.T) {
	log, lib := setup(t)
	log.WithMetadata(map[string]any{"userId": 42}).Info("meta")
	line := lib.PopLine()
	m := metadataMap(line)
	if m == nil {
		t.Fatalf("expected metadata map, got %T: %v", line.Metadata, line.Metadata)
	}
	if m["userId"] != 42 {
		t.Errorf("userId: got %v", m["userId"])
	}
}

func TestWithMetadataStruct(t *testing.T) {
	type user struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	log, lib := setup(t)
	log.WithMetadata(user{ID: 7, Name: "Alice"}).Info("struct meta")
	line := lib.PopLine()
	u, ok := line.Metadata.(user)
	if !ok {
		t.Fatalf("expected raw struct in Metadata, got %T", line.Metadata)
	}
	if u.ID != 7 || u.Name != "Alice" {
		t.Errorf("struct fields wrong: %+v", u)
	}
}

func TestMetadataChainingReplaces(t *testing.T) {
	log, lib := setup(t)
	log.WithMetadata(map[string]any{"a": 1}).
		WithMetadata(map[string]any{"b": 2}).
		Info("chain")
	line := lib.PopLine()
	m := metadataMap(line)
	if m == nil {
		t.Fatalf("expected metadata map, got %T", line.Metadata)
	}
	if _, present := m["a"]; present {
		t.Errorf("WithMetadata should replace, but a is still present: %v", m)
	}
	if m["b"] != 2 {
		t.Errorf("expected b=2, got %v", m["b"])
	}
}

func TestMuteMetadata(t *testing.T) {
	log, lib := setup(t)
	log.MuteMetadata()
	log.WithMetadata(map[string]any{"secret": "x"}).Info("muted meta")
	line := lib.PopLine()
	if line.Metadata != nil {
		t.Errorf("metadata should be muted, got %v", line.Metadata)
	}
}

func TestMetadataOnly(t *testing.T) {
	log, lib := setup(t)
	log.MetadataOnly(map[string]any{"only": true})
	line := lib.PopLine()
	if line == nil {
		t.Fatal("expected a log line")
	}
	if line.Level != loglayer.LogLevelInfo {
		t.Errorf("MetadataOnly default level: got %s", line.Level)
	}
	m := metadataMap(line)
	if m == nil || m["only"] != true {
		t.Errorf("metadata only: got %v", line.Metadata)
	}
}

func TestMetadataOnlyCustomLevel(t *testing.T) {
	log, lib := setup(t)
	log.MetadataOnly(map[string]any{"k": 1}, loglayer.MetadataOnlyOpts{LogLevel: loglayer.LogLevelWarn})
	line := lib.PopLine()
	if line.Level != loglayer.LogLevelWarn {
		t.Errorf("MetadataOnly custom level: got %s", line.Level)
	}
}
