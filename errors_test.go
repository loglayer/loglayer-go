package loglayer_test

import (
	"errors"
	"testing"

	"go.loglayer.dev"
	"go.loglayer.dev/internal/transporttest"
)

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
