package loglayer_test

import (
	"errors"
	"testing"

	"go.loglayer.dev/loglayer"
	"go.loglayer.dev/loglayer/internal/transporttest"
	"go.loglayer.dev/loglayer/transport"
	lltest "go.loglayer.dev/loglayer/transports/testing"
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

	log.Trace("hello trace")
	assertLine(t, lib, loglayer.LogLevelTrace, "hello trace")

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
	assertLine(t, lib, loglayer.LogLevelInfo, "[app] started")
}

func TestPrefixDoesNotAffectParent(t *testing.T) {
	log, lib := setup(t)
	child := log.WithPrefix("[child]")
	_ = child
	log.Info("parent")
	assertLine(t, lib, loglayer.LogLevelInfo, "parent")
}


func TestWithFields(t *testing.T) {
	log, lib := setup(t)
	log = log.WithFields(loglayer.Fields{"requestId": "abc"})
	log.Info("req")
	line := lib.PopLine()
	if !line.HasData {
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

func TestClearFieldsAll(t *testing.T) {
	log, lib := setup(t)
	log = log.WithFields(loglayer.Fields{"x": 1})
	log = log.ClearFields()
	log.Info("cleared")
	line := lib.PopLine()
	if line.HasData && line.Data["x"] != nil {
		t.Errorf("expected fields cleared, got %v", line.Data)
	}
}

func TestClearFieldsKeys(t *testing.T) {
	log, lib := setup(t)
	log = log.WithFields(loglayer.Fields{"keep": "yes", "drop": "no"})
	log = log.ClearFields("drop")
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
	if line.HasData && line.Data["secret"] != nil {
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
	log.MetadataOnly(map[string]any{"k": 1}, loglayer.LogLevelWarn)
	line := lib.PopLine()
	if line.Level != loglayer.LogLevelWarn {
		t.Errorf("MetadataOnly custom level: got %s", line.Level)
	}
}


func TestWithError(t *testing.T) {
	log, lib := setup(t)
	err := errors.New("boom")
	log.WithError(err).Error("failed")
	line := lib.PopLine()
	errData, ok := line.Data["err"].(map[string]any)
	if !ok {
		t.Fatalf("expected err field, got %v", line.Data)
	}
	if errData["message"] != "boom" {
		t.Errorf("error message: got %v", errData["message"])
	}
}

func TestErrorFieldName(t *testing.T) {
	log, lib := setupWithConfig(t, loglayer.Config{ErrorFieldName: "error"})
	log.WithError(errors.New("oops")).Error("err test")
	line := lib.PopLine()
	if line.Data["error"] == nil {
		t.Errorf("expected 'error' field, got %v", line.Data)
	}
}

func TestErrorOnly(t *testing.T) {
	log, lib := setup(t)
	log.ErrorOnly(errors.New("only error"))
	line := lib.PopLine()
	if line.Level != loglayer.LogLevelError {
		t.Errorf("ErrorOnly default level: got %s", line.Level)
	}
	if line.Data["err"] == nil {
		t.Errorf("expected err field, got %v", line.Data)
	}
}

func TestErrorOnlyCustomLevel(t *testing.T) {
	log, lib := setup(t)
	log.ErrorOnly(errors.New("fatal err"), loglayer.ErrorOnlyOpts{LogLevel: loglayer.LogLevelFatal})
	line := lib.PopLine()
	if line.Level != loglayer.LogLevelFatal {
		t.Errorf("ErrorOnly custom level: got %s", line.Level)
	}
}

func TestErrorOnlyCopyMsg(t *testing.T) {
	log, lib := setupWithConfig(t, loglayer.Config{CopyMsgOnOnlyError: true})
	log.ErrorOnly(errors.New("copied message"))
	line := lib.PopLine()
	if !transporttest.MessageContains(line.Messages, "copied message") {
		t.Errorf("message %q not found in %v", "copied message", line.Messages)
	}
}

func TestCustomErrorSerializer(t *testing.T) {
	log, lib := setupWithConfig(t, loglayer.Config{
		ErrorSerializer: func(err error) map[string]any {
			return map[string]any{"type": "custom", "msg": err.Error()}
		},
	})
	log.WithError(errors.New("serialized")).Error("custom ser")
	line := lib.PopLine()
	errData, ok := line.Data["err"].(map[string]any)
	if !ok {
		t.Fatalf("expected err field, got %v", line.Data)
	}
	if errData["type"] != "custom" {
		t.Errorf("custom serializer type: got %v", errData["type"])
	}
}


func TestSetLevel(t *testing.T) {
	log, lib := setup(t)
	log.SetLevel(loglayer.LogLevelWarn)

	log.Info("should be dropped")
	log.Debug("should be dropped")
	if lib.Len() != 0 {
		t.Errorf("expected no lines below warn, got %d", lib.Len())
	}

	log.Warn("should appear")
	log.Error("should appear")
	log.Fatal("should appear")
	if lib.Len() != 3 {
		t.Errorf("expected 3 lines at/above warn, got %d", lib.Len())
	}
}

func TestDisableLogging(t *testing.T) {
	log, lib := setup(t)
	log.DisableLogging()
	log.Info("silenced")
	log.Error("also silenced")
	if lib.Len() != 0 {
		t.Errorf("expected no lines after DisableLogging, got %d", lib.Len())
	}
}

func TestEnableLogging(t *testing.T) {
	log, lib := setup(t)
	log.DisableLogging()
	log.EnableLogging()
	log.Info("back on")
	if lib.Len() != 1 {
		t.Errorf("expected 1 line after re-enable, got %d", lib.Len())
	}
}

func TestDisableIndividualLevel(t *testing.T) {
	log, lib := setup(t)
	log.DisableLevel(loglayer.LogLevelDebug)
	log.Debug("dropped")
	log.Info("kept")
	if lib.Len() != 1 {
		t.Errorf("expected 1 line (info), got %d", lib.Len())
	}
}

func TestIsLevelEnabled(t *testing.T) {
	log, _ := setup(t)
	log.SetLevel(loglayer.LogLevelWarn)
	if log.IsLevelEnabled(loglayer.LogLevelInfo) {
		t.Error("info should be disabled after SetLevel(warn)")
	}
	if !log.IsLevelEnabled(loglayer.LogLevelError) {
		t.Error("error should be enabled after SetLevel(warn)")
	}
}

func TestEnabledConfig(t *testing.T) {
	f := false
	log, lib := setupWithConfig(t, loglayer.Config{Enabled: &f})
	log.Info("should not appear")
	if lib.Len() != 0 {
		t.Errorf("expected no lines when Enabled=false, got %d", lib.Len())
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
	child = child.WithFields(loglayer.Fields{"child_only": "x"})

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


func TestMultipleTransports(t *testing.T) {
	lib1 := &lltest.TestLoggingLibrary{}
	lib2 := &lltest.TestLoggingLibrary{}
	t1 := lltest.New(lltest.Config{BaseConfig: transport.BaseConfig{ID: "t1"}, Library: lib1})
	t2 := lltest.New(lltest.Config{BaseConfig: transport.BaseConfig{ID: "t2"}, Library: lib2})
	log := loglayer.New(loglayer.Config{DisableFatalExit: true, Transports: []loglayer.Transport{t1, t2}})

	log.Info("broadcast")
	if lib1.Len() != 1 || lib2.Len() != 1 {
		t.Errorf("both transports should receive the log: t1=%d t2=%d", lib1.Len(), lib2.Len())
	}
}

func TestAddTransport(t *testing.T) {
	log, lib1 := setup(t)
	lib2 := &lltest.TestLoggingLibrary{}
	t2 := lltest.New(lltest.Config{BaseConfig: transport.BaseConfig{ID: "t2"}, Library: lib2})
	log.AddTransport(t2)
	log.Info("both")
	if lib1.Len() != 1 || lib2.Len() != 1 {
		t.Errorf("both transports: t1=%d t2=%d", lib1.Len(), lib2.Len())
	}
}

func TestRemoveTransport(t *testing.T) {
	lib1 := &lltest.TestLoggingLibrary{}
	lib2 := &lltest.TestLoggingLibrary{}
	t1 := lltest.New(lltest.Config{BaseConfig: transport.BaseConfig{ID: "t1"}, Library: lib1})
	t2 := lltest.New(lltest.Config{BaseConfig: transport.BaseConfig{ID: "t2"}, Library: lib2})
	log := loglayer.New(loglayer.Config{DisableFatalExit: true, Transports: []loglayer.Transport{t1, t2}})

	removed := log.RemoveTransport("t2")
	if !removed {
		t.Error("RemoveTransport should return true for existing ID")
	}
	log.Info("only t1")
	if lib1.Len() != 1 || lib2.Len() != 0 {
		t.Errorf("after remove: t1=%d t2=%d", lib1.Len(), lib2.Len())
	}
}

func TestRemoveTransportMissing(t *testing.T) {
	log, _ := setup(t)
	if log.RemoveTransport("nonexistent") {
		t.Error("RemoveTransport should return false for missing ID")
	}
}

func TestWithFreshTransports(t *testing.T) {
	log, oldLib := setup(t)
	newLib := &lltest.TestLoggingLibrary{}
	newTrans := lltest.New(lltest.Config{BaseConfig: transport.BaseConfig{ID: "new"}, Library: newLib})
	log.WithFreshTransports(newTrans)
	log.Info("new transport only")
	if oldLib.Len() != 0 || newLib.Len() != 1 {
		t.Errorf("after replace: old=%d new=%d", oldLib.Len(), newLib.Len())
	}
}

func TestGetLoggerInstance(t *testing.T) {
	lib := &lltest.TestLoggingLibrary{}
	trans := lltest.New(lltest.Config{BaseConfig: transport.BaseConfig{ID: "t"}, Library: lib})
	log := loglayer.New(loglayer.Config{DisableFatalExit: true, Transport: trans})
	instance := log.GetLoggerInstance("t")
	if instance == nil {
		t.Error("GetLoggerInstance should return the underlying library")
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
