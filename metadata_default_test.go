package loglayer

import "testing"

func TestMetadataFieldNameDefaultsToMetadata(t *testing.T) {
	log := New(Config{Transport: discardTransport{}, DisableFatalExit: true})
	if got := log.config.MetadataFieldName; got != "metadata" {
		t.Fatalf("MetadataFieldName = %q, want %q", got, "metadata")
	}
}

func TestFlattenMetadataKeepsEmpty(t *testing.T) {
	log := New(Config{Transport: discardTransport{}, DisableFatalExit: true, FlattenMetadata: true})
	if got := log.config.MetadataFieldName; got != "" {
		t.Fatalf("MetadataFieldName = %q, want empty (v2 shape)", got)
	}
}

func TestExplicitMetadataFieldNameWins(t *testing.T) {
	log := New(Config{Transport: discardTransport{}, DisableFatalExit: true, MetadataFieldName: "user", FlattenMetadata: true})
	if got := log.config.MetadataFieldName; got != "user" {
		t.Fatalf("MetadataFieldName = %q, want %q", got, "user")
	}
}

func TestChildInheritsMetadataFieldName(t *testing.T) {
	log := New(Config{Transport: discardTransport{}, DisableFatalExit: true})
	child := log.Child()
	if got := child.config.MetadataFieldName; got != "metadata" {
		t.Fatalf("child MetadataFieldName = %q, want %q", got, "metadata")
	}
}
