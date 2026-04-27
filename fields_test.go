package loglayer_test

import (
	"testing"

	"go.loglayer.dev"
)

func TestWithFields(t *testing.T) {
	log, lib := setup(t)
	log = log.WithFields(loglayer.Fields{"requestId": "abc"})
	log.Info("req")
	line := lib.PopLine()
	if len(line.Data) == 0 {
		t.Fatal("expected data populated")
	}
	if line.Data["requestId"] != "abc" {
		t.Errorf("requestId: got %v, want abc", line.Data["requestId"])
	}
}

func TestWithFieldsMerges(t *testing.T) {
	log, lib := setup(t)
	log = log.WithFields(loglayer.Fields{"a": 1})
	log = log.WithFields(loglayer.Fields{"b": 2})
	log.Info("merge")
	line := lib.PopLine()
	if line.Data["a"] != 1 || line.Data["b"] != 2 {
		t.Errorf("merged fields: got %v", line.Data)
	}
}

func TestWithoutFieldsAll(t *testing.T) {
	log, lib := setup(t)
	log = log.WithFields(loglayer.Fields{"x": 1})
	log = log.WithoutFields()
	log.Info("cleared")
	line := lib.PopLine()
	if line.Data["x"] != nil {
		t.Errorf("expected fields cleared, got %v", line.Data)
	}
}

func TestWithoutFieldsKeys(t *testing.T) {
	log, lib := setup(t)
	log = log.WithFields(loglayer.Fields{"keep": "yes", "drop": "no"})
	log = log.WithoutFields("drop")
	log.Info("partial")
	line := lib.PopLine()
	if line.Data["drop"] != nil {
		t.Errorf("drop key should be removed")
	}
	if line.Data["keep"] != "yes" {
		t.Errorf("keep key should remain, got %v", line.Data["keep"])
	}
}

func TestMuteFields(t *testing.T) {
	log, lib := setup(t)
	log = log.WithFields(loglayer.Fields{"secret": "hidden"})
	log.MuteFields()
	log.Info("muted")
	line := lib.PopLine()
	if line.Data["secret"] != nil {
		t.Errorf("fields should be muted, got %v", line.Data)
	}
}

func TestUnmuteFields(t *testing.T) {
	log, lib := setup(t)
	log = log.WithFields(loglayer.Fields{"k": "v"})
	log.MuteFields().UnmuteFields()
	log.Info("unmuted")
	line := lib.PopLine()
	if line.Data["k"] != "v" {
		t.Errorf("fields should be restored, got %v", line.Data)
	}
}

func TestGetFields(t *testing.T) {
	log, _ := setup(t)
	log = log.WithFields(loglayer.Fields{"foo": "bar"})
	fields := log.GetFields()
	if fields["foo"] != "bar" {
		t.Errorf("GetFields: got %v", fields)
	}
}

func TestFieldsKey(t *testing.T) {
	log, lib := setupWithConfig(t, loglayer.Config{FieldsKey: "ctx"})
	log = log.WithFields(loglayer.Fields{"id": 1})
	log.Info("nested")
	line := lib.PopLine()
	nested, ok := line.Data["ctx"].(map[string]any)
	if !ok {
		t.Fatalf("expected nested ctx field, got %v", line.Data)
	}
	if nested["id"] != 1 {
		t.Errorf("nested id: got %v", nested["id"])
	}
}

// Muting fields and metadata are independent: muting one should not
// suppress the other in the same emission.
func TestMute_FieldsMutedMetadataFlows(t *testing.T) {
	log, lib := setup(t)
	log = log.WithFields(loglayer.Fields{"hidden": "field"})
	log.MuteFields()

	log.WithMetadata(map[string]any{"visible": "meta"}).Info("emit")
	line := lib.PopLine()
	if line.Data["hidden"] != nil {
		t.Errorf("muted fields should not appear: %v", line.Data)
	}
	m, _ := line.Metadata.(map[string]any)
	if m["visible"] != "meta" {
		t.Errorf("metadata should still flow when only fields are muted: %v", line.Metadata)
	}
}

func TestMute_MetadataMutedFieldsFlow(t *testing.T) {
	log, lib := setup(t)
	log = log.WithFields(loglayer.Fields{"visible": "field"})
	log.MuteMetadata()

	log.WithMetadata(map[string]any{"hidden": "meta"}).Info("emit")
	line := lib.PopLine()
	if line.Data["visible"] != "field" {
		t.Errorf("fields should still flow when only metadata is muted: %v", line.Data)
	}
	if line.Metadata != nil {
		t.Errorf("muted metadata should be nil: %v", line.Metadata)
	}
}
