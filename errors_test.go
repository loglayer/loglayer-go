package loglayer_test

import (
	"errors"
	"testing"

	"go.loglayer.dev"
	"go.loglayer.dev/transport/transporttest"
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

func TestErrorOnlyCopyMsgPolicy(t *testing.T) {
	t.Run("Default uses config setting (true)", func(t *testing.T) {
		log, lib := setupWithConfig(t, loglayer.Config{CopyMsgOnOnlyError: true})
		log.ErrorOnly(errors.New("from config"), loglayer.ErrorOnlyOpts{}) // CopyMsgDefault
		line := lib.PopLine()
		if !transporttest.MessageContains(line.Messages, "from config") {
			t.Errorf("expected message copied via config: got %v", line.Messages)
		}
	})

	t.Run("Disabled overrides config true", func(t *testing.T) {
		log, lib := setupWithConfig(t, loglayer.Config{CopyMsgOnOnlyError: true})
		log.ErrorOnly(errors.New("suppressed"), loglayer.ErrorOnlyOpts{CopyMsg: loglayer.CopyMsgDisabled})
		line := lib.PopLine()
		if len(line.Messages) != 0 {
			t.Errorf("CopyMsgDisabled should suppress message; got %v", line.Messages)
		}
	})

	t.Run("Enabled overrides config false", func(t *testing.T) {
		log, lib := setupWithConfig(t, loglayer.Config{CopyMsgOnOnlyError: false})
		log.ErrorOnly(errors.New("forced"), loglayer.ErrorOnlyOpts{CopyMsg: loglayer.CopyMsgEnabled})
		line := lib.PopLine()
		if !transporttest.MessageContains(line.Messages, "forced") {
			t.Errorf("CopyMsgEnabled should force message: got %v", line.Messages)
		}
	})
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
