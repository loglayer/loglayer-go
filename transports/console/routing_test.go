package console

import (
	"io"
	"os"
	"testing"

	"go.loglayer.dev/v2"
)

func TestWriterRouting(t *testing.T) {
	tr := New(Config{})
	cases := []struct {
		level loglayer.LogLevel
		want  io.Writer
	}{
		{loglayer.LogLevelTrace, os.Stdout},
		{loglayer.LogLevelDebug, os.Stdout},
		{loglayer.LogLevelInfo, os.Stdout},
		{loglayer.LogLevelWarn, os.Stderr},
		{loglayer.LogLevelError, os.Stderr},
		{loglayer.LogLevelFatal, os.Stderr},
		{loglayer.LogLevelPanic, os.Stderr},
	}
	for _, c := range cases {
		if got := tr.writer(c.level); got != c.want {
			t.Errorf("writer(%v): got %v, want %v", c.level, got, c.want)
		}
	}
}
